package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/replica"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// outboxEntry is one row of sync_outbox.
type outboxEntry struct {
	id         int64
	scope      db.Scope
	clockEpoch uint64
	op         op.Op
	raw        string
}

// batch is a group of outbox ops that goes into segments under one ACL
// version: same scope, same clock epoch.
type batch struct {
	scope   db.Scope
	version *acl.Version
	key     []byte
	entries []outboxEntry
}

// flush writes the outbox into segments (§11.3): one or more segments per
// scope, each persisted with the next seq and queued for upload in the same
// transaction that removes its ops from the outbox, so a crash never reuses a
// seq with different content.
//
// The writer's conflict-ordering generation must be the one its field clocks
// were stamped with, so ops are grouped by clock epoch and each group names
// the newest ACL version with that clock epoch in which this device has rw
// (normally the head).
func (e *Engine) flush(ctx context.Context, chain []*acl.Version, rep *Report) error {
	head := chain[len(chain)-1]
	self := e.self()
	entries, err := e.readOutbox(ctx)
	if err != nil || len(entries) == 0 {
		return err
	}
	if head.MinSchema > segment.SchemaVersion {
		rep.Outdated, rep.Unwritten = true, len(entries)
		return nil
	}

	var batches []*batch
	byKey := map[string]*batch{}
	for _, en := range entries {
		k := fmt.Sprintf("%s/%d", en.scope, en.clockEpoch)
		b := byKey[k]
		if b == nil {
			b = &batch{scope: en.scope, version: writableVersion(chain, self, en.scope, en.clockEpoch)}
			if b.version != nil {
				key, err := e.key(ctx, en.scope, b.version.Epochs[en.scope])
				if err != nil {
					return err
				}
				b.key = key
			}
			byKey[k] = b
			batches = append(batches, b)
		}
		b.entries = append(b.entries, en)
	}

	written := 0
	err = e.inTx(ctx, func(tx *sql.Tx) error {
		written = 0
		for _, s := range db.Scopes {
			for _, b := range batches {
				if b.scope != s {
					continue
				}
				if b.version == nil || b.key == nil || !head.Perm(self, s).CanWrite() {
					rep.Unwritten += len(b.entries)
					continue
				}
				n, err := e.writeBatch(ctx, tx, b)
				if err != nil {
					return err
				}
				written += n
			}
		}
		return nil
	})
	rep.Written += written
	return err
}

// writableVersion returns the newest version of the chain in which self has
// rw on scope and whose clock epoch for scope is clockEpoch.
func writableVersion(chain []*acl.Version, self string, s db.Scope, clockEpoch uint64) *acl.Version {
	for i := len(chain) - 1; i >= 0; i-- {
		v := chain[i]
		if v.ClockEpochs[s] == clockEpoch && v.Perm(self, s).CanWrite() {
			return v
		}
	}
	return nil
}

func (e *Engine) readOutbox(ctx context.Context) ([]outboxEntry, error) {
	rows, err := e.DB.QueryContext(ctx, `SELECT seq, scope, clock_epoch, op FROM sync_outbox ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outboxEntry
	for rows.Next() {
		var en outboxEntry
		if err := rows.Scan(&en.id, &en.scope, &en.clockEpoch, &en.raw); err != nil {
			return nil, err
		}
		o, err := decodeOp(en.raw)
		if err != nil {
			return nil, fmt.Errorf("outbox entry %d: %w", en.id, err)
		}
		en.op = o
		out = append(out, en)
	}
	return out, rows.Err()
}

// decodeOp parses one op as stored in sync_outbox or sync_history.
func decodeOp(raw string) (op.Op, error) {
	ops, err := segment.DecodeOps([]byte("[" + raw + "]"))
	if err != nil {
		return op.Op{}, err
	}
	if len(ops) != 1 {
		return op.Op{}, errors.New("not one op")
	}
	return ops[0], nil
}

// writeBatch writes the entries of b as segments of this device's stream in
// b.scope and returns the number of segments.
func (e *Engine) writeBatch(ctx context.Context, tx *sql.Tx, b *batch) (int, error) {
	self := e.self()
	// §12: ops in a segment are ordered by ascending HLC. The local clock is
	// monotonic, so this is the outbox order.
	sort.SliceStable(b.entries, func(i, j int) bool { return b.entries[i].op.HLC < b.entries[j].op.HLC })

	var seq uint64
	var prev sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT seq, hash FROM sync_streams WHERE device_id = ? AND scope = ?`,
		self, b.scope).Scan(&seq, &prev)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	n := 0
	for start := 0; start < len(b.entries); {
		end, size := start, 2
		for end < len(b.entries) && (end == start || size+len(b.entries[end].raw)+1 <= maxSegmentPlain) {
			size += len(b.entries[end].raw) + 1
			end++
		}
		chunk := b.entries[start:end]
		start = end

		seq++
		ops := make([]op.Op, len(chunk))
		for i, en := range chunk {
			ops[i] = en.op
		}
		plain, err := segment.EncodeOps(ops)
		if err != nil {
			return n, err
		}
		nonce, err := segment.NewNonce()
		if err != nil {
			return n, err
		}
		hdr := segment.Header{
			ClubID: e.Club.Genesis().ClubID, DeviceID: self, Scope: b.scope, Seq: seq,
			Epoch: b.version.Epochs[b.scope], ACLVersion: b.version.ACL.Version, Schema: segment.SchemaVersion,
			Nonce: nonce, HLCMin: ops[0].HLC, HLCMax: ops[len(ops)-1].HLC,
		}
		if prev.Valid {
			p := prev.String
			hdr.Prev = &p
		}
		data, err := segment.Seal(hdr, nonce, plain, b.key, e.Club.Device().Sign)
		if err != nil {
			return n, err
		}
		hash := wire.Hash(data)
		if err := replica.QueueUpload(ctx, tx, layout.SegmentPath(self, b.scope, seq), data); err != nil {
			return n, err
		}
		if err := setPosition(ctx, tx, self, b.scope, seq, hash, StatusOK, ""); err != nil {
			return n, err
		}
		for _, en := range chunk {
			if err := addHistory(ctx, tx, b.scope, self, seq, b.version.ClockEpochs[b.scope], en.op); err != nil {
				return n, err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM sync_outbox WHERE seq = ?`, en.id); err != nil {
				return n, err
			}
		}
		prev = sql.NullString{String: hash, Valid: true}
		n++
	}
	return n, nil
}

// setPosition records the last accepted (or, for this device, written)
// segment of a stream.
func setPosition(ctx context.Context, tx *sql.Tx, device string, s db.Scope, seq uint64, hash, status, detail string) error {
	var h any
	if seq > 0 {
		h = hash
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sync_streams (device_id, scope, seq, hash, status, failures, detail) VALUES (?, ?, ?, ?, ?, 0, ?)
		 ON CONFLICT (device_id, scope) DO UPDATE SET seq = excluded.seq, hash = excluded.hash,
		   status = excluded.status, failures = 0, detail = excluded.detail`,
		device, s, seq, h, status, detail)
	return err
}

// addHistory records an op of an accepted segment (§20 sync_history).
func addHistory(ctx context.Context, tx *sql.Tx, s db.Scope, device string, seq, clockEpoch uint64, o op.Op) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sync_history (scope, device_id, seq, tbl, pk, clock_epoch, hlc, op) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT DO NOTHING`,
		s, device, seq, o.T, op.EncodePK(o.PK), clockEpoch, o.HLC, string(data))
	return err
}
