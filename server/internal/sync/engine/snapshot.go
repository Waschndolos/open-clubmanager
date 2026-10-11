package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/replica"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// base is a verified snapshot, ready to be loaded.
type base struct {
	path, hash string
	hdr        *segment.SnapshotHeader
	state      *replica.Snapshot
}

// hasBase reports whether the scope's starting state is present.
func (e *Engine) hasBase(ctx context.Context, s db.Scope) (bool, error) {
	var n int
	err := e.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_scopes WHERE scope = ?`, s).Scan(&n)
	return n > 0, err
}

// ensureBase loads the starting snapshot of a scope this device has no
// state for yet (§13.1, §15.1, §17.2 step 6). It returns false while no
// valid snapshot is available; the scope is then incomplete and its
// segments are not applied, because they build on the snapshot's state.
func (e *Engine) ensureBase(ctx context.Context, chain []*acl.Version, s db.Scope, rep *Report) (bool, error) {
	ok, err := e.hasBase(ctx, s)
	if err != nil || ok {
		return ok, err
	}
	b, err := e.findSnapshot(ctx, chain, s)
	if err != nil || b == nil {
		return false, err
	}
	err = e.inTx(ctx, func(tx *sql.Tx) error {
		return e.loadBase(ctx, tx, s, b)
	})
	if err != nil {
		return false, err
	}
	rep.Loaded = append(rep.Loaded, s)
	return true, nil
}

// loadBase applies a verified snapshot in tx and records the stream
// positions it covers. This device's own stream is skipped: its position is
// the next seq to write, and its own ops are re-applied from history.
func (e *Engine) loadBase(ctx context.Context, tx *sql.Tx, s db.Scope, b *base) error {
	a := replica.NewApplier(ctx, tx)
	if err := a.Load(s, b.state); err != nil {
		return err
	}
	for d, fr := range b.hdr.Streams {
		if d == e.self() || fr.Seq == 0 {
			continue
		}
		if err := setPosition(ctx, tx, d, s, fr.Seq, *fr.Hash, StatusOK, ""); err != nil {
			return err
		}
	}
	if err := replica.SetBase(ctx, tx, s, b.path, b.hash); err != nil {
		return err
	}
	if _, err := a.RetryPending([]db.Scope{s}); err != nil {
		return err
	}
	return a.Finish()
}

// findSnapshot returns the verified snapshot to start a scope from, or nil.
//
// The spec wants the snapshot pinned by the enrollment checkpoint (§8.4,
// §15.1). The identity layer does not hand out checkpoints yet, so the
// closest safe choice is made instead: among the snapshots that pass every
// check of §15.1, the oldest one (normally the initial snapshot of §15.4,
// whose empty frontiers mean every later change is replayed and verified
// segment by segment). Newer snapshots only matter once compaction deletes
// segments, which is not implemented either.
func (e *Engine) findSnapshot(ctx context.Context, chain []*acl.Version, s db.Scope) (*base, error) {
	files, err := e.Tr.List(ctx, "snapshots/"+string(s)+"/")
	if err != nil {
		return nil, err
	}
	type cand struct {
		path string
		en   layout.Entry
	}
	var cands []cand
	for _, f := range files {
		en, ok := layout.Parse(f.Path)
		if ok && en.Kind == layout.KindSnapshot && en.Scope == s {
			cands = append(cands, cand{f.Path, en})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].en.HLC != cands[j].en.HLC {
			return cands[i].en.HLC < cands[j].en.HLC
		}
		return cands[i].path < cands[j].path
	})
	for _, c := range cands {
		raw, err := e.Tr.Get(ctx, c.path)
		if errors.Is(err, transport.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		b, err := e.verifySnapshot(ctx, chain, c.en, raw)
		if err != nil {
			return nil, err
		}
		if b != nil {
			b.path = c.path
			return b, nil
		}
	}
	return nil, nil
}

// verifySnapshot checks a snapshot as §15.1 requires and returns nil if it
// cannot be used (invalid, or waiting for a key, an ACL version or an app
// update).
func (e *Engine) verifySnapshot(ctx context.Context, chain []*acl.Version, en layout.Entry, raw []byte) (*base, error) {
	head := chain[len(chain)-1]
	f, err := segment.Parse(raw)
	if err != nil {
		return nil, nil
	}
	h, err := f.SnapshotHeader()
	if err != nil {
		return nil, nil
	}
	if h.Scope != en.Scope || h.Epoch != en.Epoch || h.HLC != en.HLC || h.DeviceID != en.DeviceID ||
		h.ClubID != e.Club.Genesis().ClubID {
		return nil, nil
	}
	v := versionAt(chain, h.ACLVersion)
	if v == nil {
		return nil, nil
	}
	// The author is an active admin device that can read the scope, both in
	// the named version and in the latest one.
	for _, a := range []*acl.Version{v, head} {
		if !a.IsAdmin(h.DeviceID) || !a.Perm(h.DeviceID, h.Scope).CanRead() {
			return nil, nil
		}
	}
	pub, err := wire.ParsePublicKey(head.Device(h.DeviceID).SignPub)
	if err != nil || f.Verify(pub) != nil {
		return nil, nil
	}
	if h.Schema > segment.SchemaVersion || h.Epoch != v.Epochs[h.Scope] {
		return nil, nil
	}
	// Frontiers: known devices, revoked ones not beyond their cutoff, and
	// matching the segment files that are still there.
	for d, fr := range h.Streams {
		dev := head.Device(d)
		if dev == nil {
			return nil, nil
		}
		if !dev.Active() {
			c, ok := dev.Cutoff[h.Scope]
			if fr.Seq > 0 && (!ok || fr.Seq > c.Seq || (fr.Seq == c.Seq && *c.Hash != *fr.Hash)) {
				return nil, nil
			}
		}
		if fr.Seq > 0 {
			seg, err := e.Tr.Get(ctx, layout.SegmentPath(d, h.Scope, fr.Seq))
			if err == nil && wire.Hash(seg) != *fr.Hash {
				return nil, nil
			}
			if err != nil && !errors.Is(err, transport.ErrNotExist) {
				return nil, err
			}
		}
	}
	key, err := e.key(ctx, h.Scope, h.Epoch)
	if err != nil || key == nil {
		return nil, err
	}
	plain, err := f.Open(key, h.Nonce)
	if err != nil {
		return nil, nil
	}
	state, err := replica.DecodeSnapshot(ctx, e.DB, h.Scope, plain)
	if err != nil {
		return nil, nil
	}
	return &base{hash: f.Hash(), hdr: h, state: state}, nil
}

// revalidate re-evaluates the streams of revoked devices against their final
// cutoffs (§17.4): if this device has applied a segment beyond a cutoff, or a
// different segment at the cutoff, the affected scopes are rebuilt from
// their snapshot, the remaining accepted prefixes and this device's own
// changes. members.core is referenced by every other scope, so rebuilding it
// rebuilds all of them.
func (e *Engine) revalidate(ctx context.Context, chain []*acl.Version, rep *Report) error {
	head := chain[len(chain)-1]
	var affected []db.Scope
	for _, d := range head.Devices {
		if d.Active() {
			continue
		}
		rows, err := e.DB.QueryContext(ctx, `SELECT scope, seq, COALESCE(hash, '') FROM sync_streams WHERE device_id = ?`, d.DeviceID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var s db.Scope
			var seq uint64
			var hash string
			if err := rows.Scan(&s, &seq, &hash); err != nil {
				rows.Close()
				return err
			}
			c, ok := d.Cutoff[s]
			if seq > 0 && (!ok || seq > c.Seq || (seq == c.Seq && *c.Hash != hash)) {
				affected = addScope(affected, s)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	if len(affected) == 0 {
		return nil
	}
	self := e.self()
	scopes := affected
	for _, s := range affected {
		if s == db.ScopeMembersCore {
			scopes = readable(head, self)
		}
	}
	scopes = sortedScopes(scopes)
	bases := map[db.Scope]*base{}
	for _, s := range scopes {
		b, err := e.findSnapshot(ctx, chain, s)
		if err != nil {
			return err
		}
		if b == nil {
			// Without a starting state the scope cannot be rebuilt. It is
			// reported incomplete and retried with the next sync.
			rep.Incomplete = addScope(rep.Incomplete, s)
			return nil
		}
		bases[s] = b
	}
	outbox, err := e.readOutbox(ctx)
	if err != nil {
		return err
	}
	err = e.inTx(ctx, func(tx *sql.Tx) error {
		return e.rebuild(ctx, tx, scopes, bases, outbox)
	})
	if err != nil {
		return err
	}
	rep.Rebuilt = append(rep.Rebuilt, scopes...)
	return nil
}

// rebuild resets the derived state of scopes and rebuilds it from their
// snapshots and this device's own changes; the streams of other devices are
// then replayed by the following pull from the snapshot frontiers.
func (e *Engine) rebuild(ctx context.Context, tx *sql.Tx, scopes []db.Scope, bases map[db.Scope]*base, outbox []outboxEntry) error {
	self := e.self()
	var tables []string
	for _, t := range db.SyncedTables {
		for _, s := range scopes {
			if t.Scope == s {
				tables = append(tables, t.Name)
			}
		}
	}
	for i := len(tables) - 1; i >= 0; i-- {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+tables[i]); err != nil {
			return err
		}
	}
	for _, t := range tables {
		for _, q := range []string{`DELETE FROM sync_field_clock WHERE tbl = ?`, `DELETE FROM sync_conflicts WHERE tbl = ?`} {
			if _, err := tx.ExecContext(ctx, q, t); err != nil {
				return err
			}
		}
	}
	for _, s := range scopes {
		for _, q := range []string{
			`DELETE FROM sync_pending WHERE scope = ?`,
			`DELETE FROM sync_history WHERE scope = ? AND device_id != ?`,
			`DELETE FROM sync_streams WHERE scope = ? AND device_id != ?`,
		} {
			if _, err := tx.ExecContext(ctx, q, s, self); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM sync_scopes WHERE scope = ?`, s); err != nil {
			return err
		}
	}
	for _, s := range scopes {
		if err := e.loadBase(ctx, tx, s, bases[s]); err != nil {
			return err
		}
	}
	// This device's own changes: those already written to segments (from
	// history, in order) and those still in the outbox.
	a := replica.NewApplier(ctx, tx)
	apply := func(s db.Scope, ch replica.Change) error {
		res, err := a.Apply(ch)
		if err != nil {
			return err
		}
		if res.Pending != "" {
			return replica.AddPending(ctx, tx, s, ch, res.Pending)
		}
		return nil
	}
	for _, s := range scopes {
		rows, err := tx.QueryContext(ctx,
			`SELECT clock_epoch, op FROM sync_history WHERE scope = ? AND device_id = ? ORDER BY id`, s, self)
		if err != nil {
			return err
		}
		type own struct {
			epoch uint64
			raw   string
		}
		var owns []own
		for rows.Next() {
			var o own
			if err := rows.Scan(&o.epoch, &o.raw); err != nil {
				rows.Close()
				return err
			}
			owns = append(owns, o)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, o := range owns {
			parsed, err := decodeOp(o.raw)
			if err != nil {
				return fmt.Errorf("own history: %w", err)
			}
			if err := apply(s, replica.FromOp(parsed, o.epoch, self)); err != nil {
				return err
			}
		}
		for _, en := range outbox {
			if en.scope == s {
				if err := apply(s, replica.FromOp(en.op, en.clockEpoch, self)); err != nil {
					return err
				}
			}
		}
	}
	if _, err := a.RetryPending(scopes); err != nil {
		return err
	}
	return a.Finish()
}
