package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/replica"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// position is the state of one stream in sync_streams.
type position struct {
	seq      uint64
	hash     string
	status   string
	failures int
}

func (e *Engine) position(ctx context.Context, device string, s db.Scope) (position, error) {
	p := position{status: StatusOK}
	var hash sql.NullString
	err := e.DB.QueryRowContext(ctx,
		`SELECT seq, hash, status, failures FROM sync_streams WHERE device_id = ? AND scope = ?`,
		device, s).Scan(&p.seq, &hash, &p.status, &p.failures)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	p.hash = hash.String
	return p, err
}

// setStatus updates the status of a stream without moving its position.
func (e *Engine) setStatus(ctx context.Context, device string, s db.Scope, status string, failures int, detail string) error {
	_, err := e.DB.ExecContext(ctx,
		`INSERT INTO sync_streams (device_id, scope, seq, status, failures, detail) VALUES (?, ?, 0, ?, ?, ?)
		 ON CONFLICT (device_id, scope) DO UPDATE SET status = excluded.status, failures = excluded.failures,
		   detail = excluded.detail`,
		device, s, status, failures, detail)
	return err
}

// streamFiles maps device -> scope -> seq -> path of the segments found.
type streamFiles map[string]map[db.Scope]map[uint64]string

// listSegments lists segments/ once (§13.1 step 1) and keeps only names that
// match the layout exactly (§6).
func (e *Engine) listSegments(ctx context.Context) (streamFiles, error) {
	files, err := e.Tr.List(ctx, "segments/")
	if err != nil {
		return nil, err
	}
	out := streamFiles{}
	for _, f := range files {
		en, ok := layout.Parse(f.Path)
		if !ok || en.Kind != layout.KindSegment {
			continue
		}
		if out[en.DeviceID] == nil {
			out[en.DeviceID] = map[db.Scope]map[uint64]string{}
		}
		if out[en.DeviceID][en.Scope] == nil {
			out[en.DeviceID][en.Scope] = map[uint64]string{}
		}
		out[en.DeviceID][en.Scope][en.Seq] = f.Path
	}
	return out, nil
}

// pull applies the segments of other devices for every readable scope
// (§13.1), after loading the scope's starting snapshot if needed.
func (e *Engine) pull(ctx context.Context, chain []*acl.Version, rep *Report) error {
	head := chain[len(chain)-1]
	self := e.self()
	files, err := e.listSegments(ctx)
	if err != nil {
		return err
	}
	var ready []db.Scope
	for _, s := range readable(head, self) {
		ok, err := e.ensureBase(ctx, chain, s, rep)
		if err != nil {
			return err
		}
		if !ok {
			rep.Incomplete = addScope(rep.Incomplete, s)
			continue
		}
		ready = append(ready, s)
		devices := make([]string, 0, len(files))
		for d, byScope := range files {
			if d != self && len(byScope[s]) > 0 {
				devices = append(devices, d)
			}
		}
		sort.Strings(devices)
		for _, d := range devices {
			if err := e.pullStream(ctx, chain, d, s, files[d][s], rep); err != nil {
				return err
			}
		}
	}
	// §13.3 step 5: retry changes waiting for rows after each round.
	return e.inTx(ctx, func(tx *sql.Tx) error {
		a := replica.NewApplier(ctx, tx)
		if _, err := a.RetryPending(ready); err != nil {
			return err
		}
		return a.Finish()
	})
}

// verdict of verifying one segment.
type verdict int

const (
	accept verdict = iota
	wait           // §13.2 rules 7, 10, a newer ACL, missing cutoff prefix: not an error
	reject         // rules 1–5, 6 (invalid cutoff/hash), 8, 9: retried, then quarantined
	fatal          // rejected for good without retry (deterministic, not a partial file)
)

type checked struct {
	verdict verdict
	reason  string
	hdr     *segment.Header
	hash    string
	ops     []op.Op
	// clockEpoch of the scope in the segment's ACL version (§12).
	clockEpoch uint64
}

// pullStream processes one stream strictly in seq order (§13.1 steps 2–3).
func (e *Engine) pullStream(ctx context.Context, chain []*acl.Version, device string, s db.Scope, seqs map[uint64]string, rep *Report) error {
	head := chain[len(chain)-1]
	pos, err := e.position(ctx, device, s)
	if err != nil {
		return err
	}

	// §13.2 rule 6: a device revoked in the latest ACL is accepted only up
	// to its cutoff, and only once the prefix up to the cutoff is present and
	// chains to the cutoff hash.
	var cut *acl.Cutoff
	if d := head.Device(device); d != nil && !d.Active() {
		c, ok := d.Cutoff[s]
		if !ok {
			if _, next := seqs[pos.seq+1]; next {
				return e.setStatus(ctx, device, s, StatusQuarantined, pos.failures,
					"revoked device without cutoff for this scope")
			}
			return nil
		}
		if pos.seq >= c.Seq {
			// Also ends a quarantine: revoking a blocked device at its last
			// valid segment is how a blocked stream is recovered (§17.4).
			return e.setStatus(ctx, device, s, StatusOK, 0, "revoked; stream ends at its cutoff")
		}
		if pos.status == StatusQuarantined {
			return nil
		}
		cut = &c
		v, reason, err := e.checkCutoffPrefix(ctx, pos, c, seqs)
		if err != nil {
			return err
		}
		switch v {
		case wait:
			return e.setStatus(ctx, device, s, StatusWaiting, pos.failures, reason)
		case reject:
			return e.fail(ctx, device, s, pos, reason)
		}
	}

	if pos.status == StatusQuarantined {
		return nil
	}
	for {
		next := pos.seq + 1
		if cut != nil && next > cut.Seq {
			return e.setStatus(ctx, device, s, StatusOK, 0, "revoked; stream ends at its cutoff")
		}
		path, ok := seqs[next]
		if !ok {
			for q := range seqs {
				if q > next {
					return e.setStatus(ctx, device, s, StatusWaiting, pos.failures,
						fmt.Sprintf("segment %d missing", next))
				}
			}
			if pos.status != StatusOK || pos.failures > 0 {
				return e.setStatus(ctx, device, s, StatusOK, 0, "")
			}
			return nil
		}
		raw, err := e.Tr.Get(ctx, path)
		if errors.Is(err, transport.ErrNotExist) {
			return e.setStatus(ctx, device, s, StatusWaiting, pos.failures, fmt.Sprintf("segment %d vanished", next))
		}
		if err != nil {
			return err
		}
		c, err := e.verify(ctx, chain, device, s, next, pos.hash, raw)
		if err != nil {
			return err
		}
		switch c.verdict {
		case wait:
			return e.setStatus(ctx, device, s, StatusWaiting, pos.failures, c.reason)
		case reject:
			return e.fail(ctx, device, s, pos, fmt.Sprintf("segment %d: %s", next, c.reason))
		case fatal:
			return e.setStatus(ctx, device, s, StatusQuarantined, pos.failures, fmt.Sprintf("segment %d: %s", next, c.reason))
		}
		if err := e.apply(ctx, s, c); err != nil {
			return err
		}
		rep.Applied++
		pos = position{seq: next, hash: c.hash, status: StatusOK}
	}
}

// fail counts a rejection of the next segment and quarantines the stream
// after MaxAttempts consecutive ones (§13.2: a file still being synced
// fails the signature check, so the device retries a few times first).
func (e *Engine) fail(ctx context.Context, device string, s db.Scope, pos position, reason string) error {
	failures := pos.failures + 1
	if failures >= e.maxAttempts() {
		return e.setStatus(ctx, device, s, StatusQuarantined, failures, reason)
	}
	return e.setStatus(ctx, device, s, StatusWaiting, failures, "retrying: "+reason)
}

// checkCutoffPrefix verifies that the segments after pos up to the cutoff are
// present and chain to the cutoff hash (§13.2 rule 6). Only the hash chain is
// checked here; every segment is fully verified when it is applied.
func (e *Engine) checkCutoffPrefix(ctx context.Context, pos position, c acl.Cutoff, seqs map[uint64]string) (verdict, string, error) {
	prev := pos.hash
	for q := pos.seq + 1; q <= c.Seq; q++ {
		path, ok := seqs[q]
		if !ok {
			return wait, fmt.Sprintf("revoked device: segment %d up to the cutoff missing", q), nil
		}
		raw, err := e.Tr.Get(ctx, path)
		if errors.Is(err, transport.ErrNotExist) {
			return wait, fmt.Sprintf("revoked device: segment %d up to the cutoff missing", q), nil
		}
		if err != nil {
			return reject, "", err
		}
		f, err := segment.Parse(raw)
		if err != nil {
			return reject, fmt.Sprintf("segment %d: %v", q, err), nil
		}
		h, err := f.SegmentHeader()
		if err != nil {
			return reject, fmt.Sprintf("segment %d: %v", q, err), nil
		}
		if (h.Prev == nil) != (prev == "") || (h.Prev != nil && *h.Prev != prev) {
			return reject, fmt.Sprintf("segment %d does not chain to the accepted prefix", q), nil
		}
		prev = f.Hash()
	}
	if c.Hash == nil || prev != *c.Hash {
		return reject, "stream does not end at the cutoff hash", nil
	}
	return accept, "", nil
}

// verify checks one segment against all rules of §13.2.
func (e *Engine) verify(ctx context.Context, chain []*acl.Version, device string, s db.Scope, seq uint64, prevHash string, raw []byte) (*checked, error) {
	head := chain[len(chain)-1]
	no := func(v verdict, format string, args ...any) (*checked, error) {
		return &checked{verdict: v, reason: fmt.Sprintf(format, args...)}, nil
	}
	f, err := segment.Parse(raw)
	if err != nil {
		return no(reject, "%v", err)
	}
	h, err := f.SegmentHeader()
	if err != nil {
		return no(reject, "%v", err)
	}
	// Rule 1.
	if h.DeviceID != device || h.Scope != s || h.Seq != seq {
		return no(reject, "file name does not match the header")
	}
	// Rule 2.
	if h.ClubID != e.Club.Genesis().ClubID {
		return no(reject, "segment of another club")
	}
	// Final acceptance needs the ACL chain the segment names (§13.2, last
	// paragraph): its author may know a version this device has not seen.
	if h.ACLVersion > head.ACL.Version {
		return no(wait, "waiting for ACL version %d", h.ACLVersion)
	}
	// Rule 3: the key of device_id in the accepted chain; revoked devices
	// stay listed. The ACL validator has checked device_id against sign_pub.
	d := head.Device(device)
	if d == nil {
		return no(reject, "device is not in the ACL")
	}
	pub, err := wire.ParsePublicKey(d.SignPub)
	if err != nil {
		return no(reject, "%v", err)
	}
	if err := f.Verify(pub); err != nil {
		return no(reject, "%v", err)
	}
	// Rule 4.
	if (h.Prev == nil) != (prevHash == "") || (h.Prev != nil && *h.Prev != prevHash) {
		return no(reject, "prev does not match the accepted segment %d", seq-1)
	}
	// Rule 5.
	v := versionAt(chain, h.ACLVersion)
	if !v.Perm(device, s).CanWrite() {
		return no(reject, "device has no rw on %s in ACL version %d", s, h.ACLVersion)
	}
	if h.Epoch != v.Epochs[s] {
		return no(reject, "epoch %d is not the epoch of ACL version %d", h.Epoch, h.ACLVersion)
	}
	// Rule 10, before decrypting: a newer format may not parse.
	if h.Schema > segment.SchemaVersion {
		return no(wait, "schema %d is newer than this app (%d): please update the app", h.Schema, segment.SchemaVersion)
	}
	// Rule 7.
	key, err := e.key(ctx, s, h.Epoch)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return no(wait, "key of %s epoch %d not available", s, h.Epoch)
	}
	// Rule 8.
	plain, err := f.Open(key, h.Nonce)
	if err != nil {
		return no(reject, "%v", err)
	}
	ops, err := segment.DecodeOps(plain)
	if err != nil {
		return no(reject, "%v", err)
	}
	// Rule 9.
	if len(ops) == 0 {
		return no(reject, "empty segment")
	}
	ops, err = upcastOps(h.Schema, ops)
	if err != nil {
		return no(reject, "%v", err)
	}
	for i := range ops {
		if err := replica.Validate(ctx, e.DB, s, &ops[i]); err != nil {
			if errors.Is(err, replica.ErrUnsupported) {
				return no(wait, "%v: please update the app", err)
			}
			return no(reject, "%v", err)
		}
	}
	// Rule 8, HLC range; and §12: ops ordered by ascending HLC.
	if ops[0].HLC != h.HLCMin || ops[len(ops)-1].HLC != h.HLCMax {
		return no(reject, "hlc_min/hlc_max do not match the ops")
	}
	for i := 1; i < len(ops); i++ {
		if ops[i].HLC <= ops[i-1].HLC {
			return no(reject, "ops are not in ascending HLC order")
		}
	}
	return &checked{verdict: accept, hdr: h, hash: f.Hash(), ops: ops, clockEpoch: v.ClockEpochs[s]}, nil
}

// apply applies an accepted segment in one transaction (§13.3): every op by
// LWW against the field clocks, pending ops for missing rows, history,
// uniqueness checks, the new stream position, and the HLC receive rule (§12).
func (e *Engine) apply(ctx context.Context, s db.Scope, c *checked) error {
	return e.inTx(ctx, func(tx *sql.Tx) error {
		a := replica.NewApplier(ctx, tx)
		for _, o := range c.ops {
			ch := replica.FromOp(o, c.clockEpoch, c.hdr.DeviceID)
			res, err := a.Apply(ch)
			if err != nil {
				return err
			}
			if res.Pending != "" {
				if err := replica.AddPending(ctx, tx, s, ch, res.Pending); err != nil {
					return err
				}
			}
			if err := addHistory(ctx, tx, s, c.hdr.DeviceID, c.hdr.Seq, c.clockEpoch, o); err != nil {
				return err
			}
		}
		if err := a.Finish(); err != nil {
			return err
		}
		if err := setPosition(ctx, tx, c.hdr.DeviceID, s, c.hdr.Seq, c.hash, StatusOK, ""); err != nil {
			return err
		}
		return e.observe(ctx, tx, c.hdr.DeviceID, c.hdr.HLCMax)
	})
}

// observe applies the receive rule of §12: the local clock follows a
// received HLC unless it lies more than MAX_DRIFT ahead of the wall clock;
// then the author gets a clock warning. Acceptance does not depend on it.
func (e *Engine) observe(ctx context.Context, tx *sql.Tx, device, received string) error {
	r, err := hlc.Parse(received)
	if err != nil {
		return err
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT hlc FROM sync_device WHERE id = 1`).Scan(&stored); err != nil {
		return err
	}
	last, err := hlc.Parse(stored)
	if err != nil {
		return err
	}
	clock := hlc.NewWithSource(last, e.Now, nil)
	if !clock.Observe(r) {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO sync_clock_warnings (device_id, hlc) VALUES (?, ?)
			 ON CONFLICT (device_id) DO UPDATE SET hlc = MAX(hlc, excluded.hlc),
			   seen_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`, device, received)
		return err
	}
	if clock.Last() != last {
		_, err := tx.ExecContext(ctx, `UPDATE sync_device SET hlc = ? WHERE id = 1`, clock.Last().String())
		return err
	}
	return nil
}

// sortedScopes returns scopes in registry order.
func sortedScopes(set []db.Scope) []db.Scope {
	var out []db.Scope
	for _, s := range db.Scopes {
		if slices.Contains(set, s) {
			out = append(out, s)
		}
	}
	return out
}
