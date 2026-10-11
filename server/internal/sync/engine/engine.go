// Package engine exchanges data between the devices of a club
// (docs/sync-protocol.md §11–§15): it flushes the local outbox into signed,
// encrypted segments, discovers and verifies the segments of other devices,
// applies them, loads snapshots for joining devices, and rebuilds scopes when
// a revocation invalidates changes that were already applied.
//
// The identity layer (package club) provides the accepted ACL chain and the
// scope keys; package replica does the database work; package segment the
// file format.
package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/replica"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

// DefaultMaxAttempts is how often the next segment of a stream may fail
// verification before the stream is quarantined. Cloud clients deliver
// files piece by piece, so a file that is still arriving fails a few times
// first (§13.2 "Handling failures").
const DefaultMaxAttempts = 3

// maxSegmentPlain bounds the op JSON of one segment, well below the limits
// of §11.2, so that a large outbox is split into several segments.
const maxSegmentPlain = 4 << 20

// Stream statuses in sync_streams.
const (
	StatusOK          = "ok"
	StatusWaiting     = "waiting"
	StatusQuarantined = "quarantined"
)

// Engine syncs one club. It is safe for concurrent use; syncs are
// serialized.
type Engine struct {
	DB   *sql.DB
	Club *club.Club
	Tr   transport.Transport
	// Now is the wall clock, used for the drift check of §12.
	Now func() time.Time
	// MaxAttempts overrides DefaultMaxAttempts.
	MaxAttempts int

	mu   sync.Mutex
	keys map[keyRef][]byte // scope keys fetched during one sync
}

type keyRef struct {
	scope db.Scope
	epoch uint64
}

// New returns an engine for a club and its transport.
func New(conn *sql.DB, c *club.Club, tr transport.Transport) *Engine {
	return &Engine{DB: conn, Club: c, Tr: tr, Now: time.Now}
}

// Report describes what a sync did and what is left.
type Report struct {
	// Written is the number of segments created from the outbox.
	Written int
	// Uploaded is the number of files uploaded (segments and snapshots).
	Uploaded int
	// Applied is the number of segments of other devices applied.
	Applied int
	// Loaded lists scopes whose starting snapshot was loaded.
	Loaded []db.Scope
	// Rebuilt lists scopes rebuilt after a revocation (§17.4).
	Rebuilt []db.Scope
	// Incomplete lists readable scopes that have no verified starting state
	// (no valid snapshot yet, or a rebuild could not be done). They must not
	// be reported as converged.
	Incomplete []db.Scope
	// Unwritten is the number of outbox ops that could not be written:
	// the device lacks rw on their scope, or the app is below min_schema.
	Unwritten int
	// Outdated is true when the ACL's min_schema is above this app's schema
	// version (§16): the device stops writing until it is updated.
	Outdated bool
	// UploadErr reports files that could not be uploaded; they are retried.
	UploadErr error
}

// Sync flushes the outbox, uploads, and pulls and applies the changes of
// other devices. The caller refreshes the club (ACL and keys) first. A
// device that is not an active member does nothing.
func (e *Engine) Sync(ctx context.Context) (*Report, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.keys = map[keyRef][]byte{}
	defer func() {
		for _, k := range e.keys {
			clear(k)
		}
		e.keys = nil
	}()

	rep := &Report{}
	chain := e.Club.Chain()
	if len(chain) == 0 || e.Club.Status() != club.StatusMember {
		return rep, nil
	}
	head := chain[len(chain)-1]
	if err := e.setClockEpochs(ctx, head); err != nil {
		return nil, err
	}
	if err := e.flush(ctx, chain, rep); err != nil {
		return nil, fmt.Errorf("engine: flush: %w", err)
	}
	n, err := replica.Upload(ctx, e.DB, e.Tr)
	rep.Uploaded = n
	if err != nil {
		rep.UploadErr = err
	}
	if err := e.revalidate(ctx, chain, rep); err != nil {
		return nil, fmt.Errorf("engine: revocation check: %w", err)
	}
	if err := e.pull(ctx, chain, rep); err != nil {
		return nil, fmt.Errorf("engine: pull: %w", err)
	}
	return rep, nil
}

// setClockEpochs records the head's clock epochs, which local writes stamp
// on their field clocks and outbox entries.
func (e *Engine) setClockEpochs(ctx context.Context, head *acl.Version) error {
	for _, s := range db.Scopes {
		ce := head.ClockEpochs[s]
		if ce == 0 {
			ce = 1
		}
		if _, err := e.DB.ExecContext(ctx,
			`INSERT INTO sync_clock_epochs (scope, clock_epoch) VALUES (?, ?)
			 ON CONFLICT (scope) DO UPDATE SET clock_epoch = excluded.clock_epoch`, s, ce); err != nil {
			return err
		}
	}
	return nil
}

// key returns K(scope, epoch), verified against the ACL, or nil if this
// device does not have it. Keys are fetched outside transactions: the
// database has a single connection.
func (e *Engine) key(ctx context.Context, s db.Scope, epoch uint64) ([]byte, error) {
	ref := keyRef{s, epoch}
	if k, ok := e.keys[ref]; ok {
		return k, nil
	}
	k, err := e.Club.ScopeKey(ctx, s, epoch)
	if errors.Is(err, club.ErrMissingKey) {
		e.keys[ref] = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.keys[ref] = k
	return k, nil
}

func (e *Engine) self() string { return e.Club.Device().ID }

func (e *Engine) maxAttempts() int {
	if e.MaxAttempts > 0 {
		return e.MaxAttempts
	}
	return DefaultMaxAttempts
}

// inTx runs fn in a transaction with deferred foreign keys (§13.3 step 5).
func (e *Engine) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// versionAt returns ACL version n of the chain, or nil.
func versionAt(chain []*acl.Version, n uint64) *acl.Version {
	if n < 1 || n > uint64(len(chain)) {
		return nil
	}
	return chain[n-1]
}

// readable returns the scopes this device may read in head, in registry
// order (members.core first, which the other scopes reference).
func readable(head *acl.Version, self string) []db.Scope {
	var out []db.Scope
	for _, s := range db.Scopes {
		if head.Perm(self, s).CanRead() {
			out = append(out, s)
		}
	}
	return out
}

// StreamCounts counts the streams of other devices by status.
type StreamCounts struct {
	OK, Waiting, Quarantined int
}

// Counts returns the number of streams per status, the number of changes
// waiting for missing rows, and the devices with a clock warning (§12).
func Counts(ctx context.Context, conn *sql.DB) (StreamCounts, int, []string, error) {
	var sc StreamCounts
	rows, err := conn.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM sync_streams
		 WHERE device_id IS NOT (SELECT device_id FROM sync_device WHERE id = 1) GROUP BY status`)
	if err != nil {
		return sc, 0, nil, err
	}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return sc, 0, nil, err
		}
		switch st {
		case StatusOK:
			sc.OK = n
		case StatusWaiting:
			sc.Waiting = n
		case StatusQuarantined:
			sc.Quarantined = n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sc, 0, nil, err
	}
	var pending int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_pending`).Scan(&pending); err != nil {
		return sc, 0, nil, err
	}
	rows, err = conn.QueryContext(ctx, `SELECT device_id FROM sync_clock_warnings ORDER BY device_id`)
	if err != nil {
		return sc, 0, nil, err
	}
	defer rows.Close()
	var warn []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return sc, 0, nil, err
		}
		warn = append(warn, d)
	}
	return sc, pending, warn, rows.Err()
}

func addScope(list []db.Scope, s db.Scope) []db.Scope {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}
