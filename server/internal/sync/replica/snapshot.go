package replica

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// Cell is one field of a snapshot row: its value and its field clock.
type Cell struct {
	V any `json:"v"`
	Clock
}

// Row is the full state of one row in a snapshot, tombstones included.
type Row struct {
	T     string          `json:"t"`
	PK    map[string]any  `json:"pk"`
	Cells map[string]Cell `json:"cells"`
}

// Snapshot is the plaintext of a snapshot file (§15.1): the full state of
// one scope with field clocks, plus the changes waiting for missing rows
// (they belong to the covered stream prefixes but are not in the tables).
// The spec fixes only what a snapshot contains; this layout is ours.
type Snapshot struct {
	Rows    []Row    `json:"rows"`
	Pending []Change `json:"pending"`
}

// ReadState returns the state of a scope. Columns that have a field clock
// carry it; non-NULL columns without one (data from before sync, which
// club creation normally backfills) carry the zero clock, which is older
// than every write. updated_at is never included: every device derives it.
func ReadState(ctx context.Context, tx *sql.Tx, scope db.Scope) (*Snapshot, error) {
	snap := &Snapshot{Rows: []Row{}, Pending: []Change{}}
	for _, t := range db.SyncedTables {
		if t.Scope != scope {
			continue
		}
		rows, err := tx.QueryContext(ctx, `SELECT * FROM `+t.Name+` ORDER BY `+commaJoin(t.PK))
		if err != nil {
			return nil, err
		}
		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}
		var batch []Row
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return nil, err
			}
			r := Row{T: t.Name, PK: map[string]any{}, Cells: map[string]Cell{}}
			for i, c := range cols {
				v := op.Normalize(vals[i])
				if slices.Contains(t.PK, c) {
					r.PK[c] = v
				}
				if c != "updated_at" {
					r.Cells[c] = Cell{V: v}
				}
			}
			batch = append(batch, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for i := range batch {
			clocks, err := fieldClocks(ctx, tx, t.Name, op.EncodePK(batch[i].PK))
			if err != nil {
				return nil, err
			}
			for c, cell := range batch[i].Cells {
				if clk, ok := clocks[c]; ok {
					cell.Clock = clk
					batch[i].Cells[c] = cell
				} else if cell.V == nil {
					delete(batch[i].Cells, c)
				}
			}
		}
		snap.Rows = append(snap.Rows, batch...)
	}
	rows, err := tx.QueryContext(ctx, `SELECT op FROM sync_pending WHERE scope = ? ORDER BY id`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var ch Change
		if err := decodeChange(data, &ch); err != nil {
			return nil, err
		}
		snap.Pending = append(snap.Pending, ch)
	}
	return snap, rows.Err()
}

func commaJoin(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

// Encode returns the snapshot plaintext.
func (s *Snapshot) Encode() ([]byte, error) { return json.Marshal(s) }

// DecodeSnapshot parses a snapshot plaintext strictly and checks it against
// the local schema: every row and pending change belongs to scope, has the
// table's primary key and names only known columns.
func DecodeSnapshot(ctx context.Context, tx Queryer, scope db.Scope, plain []byte) (*Snapshot, error) {
	var s Snapshot
	if err := wire.DecodeStrict(plain, &s); err != nil {
		return nil, fmt.Errorf("%w: snapshot: %v", ErrInvalidOp, err)
	}
	for i := range s.Rows {
		r := &s.Rows[i]
		normalize(r.PK)
		set := map[string]any{}
		for c, cell := range r.Cells {
			if err := checkClock(cell.Clock, true); err != nil {
				return nil, fmt.Errorf("%w: snapshot %s.%s: %v", ErrInvalidOp, r.T, c, err)
			}
			cell.V = op.Normalize(cell.V)
			r.Cells[c] = cell
			set[c] = cell.V
		}
		o := op.Op{T: r.T, PK: r.PK, HLC: "000000000000-0000", Set: set}
		if err := Validate(ctx, tx, scope, &o); err != nil {
			return nil, err
		}
	}
	for i := range s.Pending {
		ch := &s.Pending[i]
		normalize(ch.PK)
		normalize(ch.Set)
		o := op.Op{T: ch.T, PK: ch.PK, HLC: ch.Clock.HLC, Set: ch.Set}
		if err := Validate(ctx, tx, scope, &o); err != nil {
			return nil, err
		}
		for _, clk := range append(slices.Collect(maps.Values(ch.Clocks)), ch.Clock) {
			if err := checkClock(clk, false); err != nil {
				return nil, fmt.Errorf("%w: snapshot pending %s: %v", ErrInvalidOp, ch.T, err)
			}
		}
	}
	return &s, nil
}

// checkClock validates a clock read from a snapshot. The zero clock stands
// for data from before sync and is allowed for rows only.
func checkClock(c Clock, zeroOK bool) error {
	if zeroOK && c == (Clock{}) {
		return nil
	}
	if c.Epoch < 1 || !layout.ValidDeviceID(c.Device) {
		return errors.New("bad field clock")
	}
	_, err := hlc.Parse(c.HLC)
	return err
}

func normalize(m map[string]any) {
	for k, v := range m {
		m[k] = op.Normalize(v)
	}
}

// Load applies a snapshot through the applier: rows with their per-field
// clocks, and the pending changes it carries. Rows whose references are
// still missing go to sync_pending like any other change.
func (a *Applier) Load(scope db.Scope, s *Snapshot) error {
	for _, r := range s.Rows {
		ch := Change{T: r.T, PK: r.PK, Set: map[string]any{}, Clocks: map[string]Clock{}}
		for c, cell := range r.Cells {
			ch.Set[c] = cell.V
			ch.Clocks[c] = cell.Clock
			if ch.Clock.Less(cell.Clock) {
				ch.Clock = cell.Clock
			}
		}
		res, err := a.Apply(ch)
		if err != nil {
			return err
		}
		if res.Pending != "" {
			if err := AddPending(a.ctx, a.tx, scope, ch, res.Pending); err != nil {
				return err
			}
		}
	}
	for _, ch := range s.Pending {
		if err := AddPending(a.ctx, a.tx, scope, ch, "from snapshot"); err != nil {
			return err
		}
	}
	return nil
}

// QueueUpload stores a file this device created, to be uploaded by Upload.
// It runs in the transaction that persists the state the file stands for
// (§11.3: persist before uploading, retry with the identical file).
func QueueUpload(ctx context.Context, tx *sql.Tx, path string, data []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sync_uploads (path, data) VALUES (?, ?)`, path, data)
	return err
}

// ErrBlocked is returned by Upload when a file this device must create
// already exists with different content: the path is taken, and the stream
// cannot continue (§17.4 "persistent invalid file").
var ErrBlocked = errors.New("replica: upload path is taken by a different file")

// Upload creates the queued files, in path order (so ascending seq per
// stream), and forgets each one once it exists with the exact bytes. It
// returns the number of files uploaded.
func Upload(ctx context.Context, conn *sql.DB, tr transport.Transport) (int, error) {
	rows, err := conn.QueryContext(ctx, `SELECT path, data FROM sync_uploads ORDER BY path`)
	if err != nil {
		return 0, err
	}
	type file struct {
		path string
		data []byte
	}
	var files []file
	for rows.Next() {
		var f file
		if err := rows.Scan(&f.path, &f.data); err != nil {
			rows.Close()
			return 0, err
		}
		files = append(files, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	n := 0
	var blocked error
	stopped := map[string]bool{} // directories (streams) behind a blocked file
	for _, f := range files {
		dir := path.Dir(f.path)
		if stopped[dir] {
			continue
		}
		err := tr.Create(ctx, f.path, f.data)
		if errors.Is(err, transport.ErrExists) {
			// A previous attempt may have succeeded before the crash.
			got, gerr := tr.Get(ctx, f.path)
			if gerr != nil {
				return n, gerr
			}
			if !slices.Equal(got, f.data) {
				blocked = errors.Join(blocked, fmt.Errorf("%w: %s", ErrBlocked, f.path))
				stopped[dir] = true
				continue
			}
			err = nil
		}
		if err != nil {
			return n, err
		}
		if _, err := conn.ExecContext(ctx, `DELETE FROM sync_uploads WHERE path = ?`, f.path); err != nil {
			return n, err
		}
		n++
	}
	return n, blocked
}
