// Package replica applies verified changes to the local SQLite database and
// reads its state for snapshots (docs/sync-protocol.md §13.3, §14, §15). It
// works on a transaction and knows nothing about files, keys or ACLs; the
// caller (package engine, or club creation) has verified what it passes in.
package replica

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

// Clock is the conflict-ordering tuple of a field write (§12, §14.1).
type Clock struct {
	Epoch  uint64 `json:"e"`
	HLC    string `json:"h"`
	Device string `json:"d"`
}

// Less orders clocks by (clock_epoch, hlc, device_id). HLC strings have a
// fixed width of lowercase hex, so string order is time order.
func (c Clock) Less(o Clock) bool {
	if c.Epoch != o.Epoch {
		return c.Epoch < o.Epoch
	}
	if c.HLC != o.HLC {
		return c.HLC < o.HLC
	}
	return c.Device < o.Device
}

// Change writes columns of one row. Every column carries Clock unless
// Clocks names its own (snapshots keep one clock per field).
type Change struct {
	T      string           `json:"t"`
	PK     map[string]any   `json:"pk"`
	Set    map[string]any   `json:"set"`
	Clock  Clock            `json:"clock"`
	Clocks map[string]Clock `json:"clocks,omitempty"`
}

func (ch *Change) clockOf(col string) Clock {
	if c, ok := ch.Clocks[col]; ok {
		return c
	}
	return ch.Clock
}

// FromOp turns a verified op into a change with the clock of its segment.
func FromOp(o op.Op, clockEpoch uint64, device string) Change {
	return Change{T: o.T, PK: o.PK, Set: o.Set, Clock: Clock{Epoch: clockEpoch, HLC: o.HLC, Device: device}}
}

// Queryer is a *sql.DB or *sql.Tx. The database has a single connection, so
// code holding a transaction must pass it rather than the *sql.DB.
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// column describes one column of a synced table.
type column struct {
	name       string
	notNull    bool
	hasDefault bool
	pk         bool
}

// foreignKey is a reference from a column to another table's column.
type foreignKey struct {
	from, table, to string
}

type tableMeta struct {
	db.Table
	columns map[string]column
	fks     []foreignKey
}

var (
	metaMu    sync.Mutex
	metaCache = map[string]*tableMeta{}
)

// meta reads the columns and foreign keys of a synced table. The schema is
// fixed for the life of the process (migrations run at db.Open), so the
// result is cached.
func meta(ctx context.Context, tx Queryer, table string) (*tableMeta, error) {
	metaMu.Lock()
	defer metaMu.Unlock()
	if m, ok := metaCache[table]; ok {
		return m, nil
	}
	t, ok := db.SyncedTable(table)
	if !ok {
		return nil, fmt.Errorf("replica: %s is not a synced table", table)
	}
	m := &tableMeta{Table: t, columns: map[string]column{}}
	rows, err := tx.QueryContext(ctx, `SELECT name, "notnull", dflt_value IS NOT NULL, pk FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c column
		var pk int
		if err := rows.Scan(&c.name, &c.notNull, &c.hasDefault, &pk); err != nil {
			rows.Close()
			return nil, err
		}
		c.pk = pk > 0
		m.columns[c.name] = c
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT "table", "from", "to" FROM pragma_foreign_key_list(?)`, table)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var fk foreignKey
		if err := rows.Scan(&fk.table, &fk.from, &fk.to); err != nil {
			rows.Close()
			return nil, err
		}
		m.fks = append(m.fks, fk)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	metaCache[table] = m
	return m, nil
}

// ErrInvalidOp is returned by Validate for an op that violates the format of
// §10 or targets a table outside its segment's scope (§13.2 rule 9).
var ErrInvalidOp = errors.New("replica: invalid op")

// ErrUnsupported is returned by Validate for valid ops this version cannot
// apply yet (erasure, §15.3). The stream waits, as for a newer schema.
var ErrUnsupported = errors.New("replica: op not supported by this version")

// Validate checks an op against the local schema: the table belongs to
// scope, the primary key has exactly the table's key columns, and set names
// only known columns (never updated_at) with scalar values.
func Validate(ctx context.Context, tx Queryer, scope db.Scope, o *op.Op) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidOp}, args...)...)
	}
	t, ok := db.SyncedTable(o.T)
	if !ok || t.Scope != scope {
		return bad("table %q is not in scope %s", o.T, scope)
	}
	if _, err := hlc.Parse(o.HLC); err != nil {
		return bad("%v", err)
	}
	m, err := meta(ctx, tx, o.T)
	if err != nil {
		return err
	}
	if len(o.PK) != len(t.PK) {
		return bad("%s: primary key %v", o.T, o.PK)
	}
	for _, c := range t.PK {
		v, ok := o.PK[c].(string)
		if !ok || v == "" {
			return bad("%s: primary key column %s", o.T, c)
		}
	}
	if len(o.Set) == 0 && len(o.Purge) == 0 {
		return bad("%s: empty op", o.T)
	}
	for c, v := range o.Set {
		col, ok := m.columns[c]
		if !ok || c == "updated_at" {
			return bad("%s: column %q", o.T, c)
		}
		switch v.(type) {
		case nil, string, int64, float64, bool:
		default:
			return bad("%s.%s: value of type %T", o.T, c, v)
		}
		if col.pk && v != o.PK[c] {
			return bad("%s.%s: differs from the primary key", o.T, c)
		}
		if col.pk && v == nil {
			return bad("%s.%s: primary key column set to null", o.T, c)
		}
	}
	if len(o.Purge) > 0 {
		return fmt.Errorf("%w: purge (§15.3)", ErrUnsupported)
	}
	return nil
}

// Applier applies changes inside one transaction.
type Applier struct {
	ctx     context.Context
	tx      *sql.Tx
	touched map[string]map[string]bool // table -> pk text
}

// NewApplier returns an applier on tx. The caller runs the transaction with
// PRAGMA defer_foreign_keys = ON (§13.3 step 5); the applier itself never
// writes a row whose references are missing.
func NewApplier(ctx context.Context, tx *sql.Tx) *Applier {
	return &Applier{ctx: ctx, tx: tx, touched: map[string]map[string]bool{}}
}

// Result of applying one change.
type Result struct {
	// Written lists the columns whose value and clock were written.
	Written []string
	// Pending is non-empty when the change could not be applied because a
	// row it needs is missing; the caller keeps it in sync_pending.
	Pending string
}

// Apply writes the columns of ch whose clock is newer than the stored field
// clock (§13.3 step 2), inserting the row if it does not exist (step 3),
// and sets updated_at from the newest field clock of the row (step 4). If
// the row does not exist and ch lacks a required column, or a written
// column references a missing row, nothing is written and Result.Pending
// says why. Applying the same change twice has no effect.
func (a *Applier) Apply(ch Change) (Result, error) {
	m, err := meta(a.ctx, a.tx, ch.T)
	if err != nil {
		return Result{}, err
	}
	pkText := op.EncodePK(ch.PK)
	where, whereArgs := pkWhere(m.Table, ch.PK)

	var one int
	err = a.tx.QueryRowContext(a.ctx, `SELECT 1 FROM `+ch.T+` WHERE `+where, whereArgs...).Scan(&one)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Result{}, err
	}

	stored, err := fieldClocks(a.ctx, a.tx, ch.T, pkText)
	if err != nil {
		return Result{}, err
	}
	win := map[string]any{}
	for col, v := range ch.Set {
		c := ch.clockOf(col)
		if cur, ok := stored[col]; ok && !cur.Less(c) {
			continue
		}
		win[col] = v
	}

	if !exists {
		for name, col := range m.columns {
			if col.pk || !col.notNull || col.hasDefault {
				continue
			}
			if win[name] == nil {
				return Result{Pending: fmt.Sprintf("row %s %s missing", ch.T, pkText)}, nil
			}
		}
	}
	for _, fk := range m.fks {
		v, written := win[fk.from]
		if !written && !exists {
			v = ch.PK[fk.from]
		}
		if v == nil {
			continue
		}
		err := a.tx.QueryRowContext(a.ctx, `SELECT 1 FROM `+fk.table+` WHERE `+fk.to+` = ?`, v).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return Result{Pending: fmt.Sprintf("%s %v missing", fk.table, v)}, nil
		}
		if err != nil {
			return Result{}, err
		}
	}
	if len(win) == 0 {
		return Result{}, nil
	}

	cols := slices.Sorted(maps.Keys(win))
	if exists {
		var assign []string
		var args []any
		for _, c := range cols {
			if m.columns[c].pk {
				continue
			}
			assign = append(assign, c+` = ?`)
			args = append(args, win[c])
		}
		if len(assign) > 0 {
			if _, err := a.tx.ExecContext(a.ctx,
				`UPDATE `+ch.T+` SET `+strings.Join(assign, ", ")+` WHERE `+where, append(args, whereArgs...)...); err != nil {
				return Result{}, fmt.Errorf("replica: update %s %s: %w", ch.T, pkText, err)
			}
		}
	} else {
		values := map[string]any{}
		for k, v := range ch.PK {
			values[k] = v
		}
		for k, v := range win {
			values[k] = v
		}
		ins := slices.Sorted(maps.Keys(values))
		args := make([]any, len(ins))
		for i, c := range ins {
			args[i] = values[c]
		}
		if _, err := a.tx.ExecContext(a.ctx,
			`INSERT INTO `+ch.T+` (`+strings.Join(ins, ", ")+`) VALUES (`+placeholders(len(ins))+`)`, args...); err != nil {
			return Result{}, fmt.Errorf("replica: insert %s %s: %w", ch.T, pkText, err)
		}
	}
	for _, c := range cols {
		clk := ch.clockOf(c)
		if _, err := a.tx.ExecContext(a.ctx,
			`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id, clock_epoch) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (tbl, pk, col) DO UPDATE SET hlc = excluded.hlc, device_id = excluded.device_id,
			   clock_epoch = excluded.clock_epoch`,
			ch.T, pkText, c, clk.HLC, clk.Device, clk.Epoch); err != nil {
			return Result{}, err
		}
	}
	if !m.AppendOnly {
		if err := setUpdatedAt(a.ctx, a.tx, m.Table, ch.PK, pkText); err != nil {
			return Result{}, err
		}
	}
	if a.touched[ch.T] == nil {
		a.touched[ch.T] = map[string]bool{}
	}
	a.touched[ch.T][pkText] = true
	return Result{Written: cols}, nil
}

// Finish runs the uniqueness checks of §14.2 for every row written through
// the applier (§13.3 step 6).
func (a *Applier) Finish() error {
	for _, table := range slices.Sorted(maps.Keys(a.touched)) {
		pks := slices.Sorted(maps.Keys(a.touched[table]))
		if err := store.DetectUniqueConflicts(a.ctx, a.tx, table, pks); err != nil {
			return err
		}
	}
	a.touched = map[string]map[string]bool{}
	return nil
}

// fieldClocks returns the stored clocks of a row by column. A clock written
// before the device identity existed has no device id; it belongs to this
// device.
func fieldClocks(ctx context.Context, tx *sql.Tx, table, pkText string) (map[string]Clock, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT col, clock_epoch, hlc, COALESCE(device_id, (SELECT device_id FROM sync_device WHERE id = 1), '')
		 FROM sync_field_clock WHERE tbl = ? AND pk = ?`, table, pkText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Clock{}
	for rows.Next() {
		var col string
		var c Clock
		if err := rows.Scan(&col, &c.Epoch, &c.HLC, &c.Device); err != nil {
			return nil, err
		}
		out[col] = c
	}
	return out, rows.Err()
}

// setUpdatedAt derives updated_at from the newest HLC among the row's field
// clocks, so that every device that holds the same clocks shows the same
// value (§10: ops never carry updated_at).
func setUpdatedAt(ctx context.Context, tx *sql.Tx, t db.Table, pk map[string]any, pkText string) error {
	var newest sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(hlc) FROM sync_field_clock WHERE tbl = ? AND pk = ?`, t.Name, pkText).Scan(&newest); err != nil {
		return err
	}
	if !newest.Valid {
		return nil
	}
	ts, err := hlc.Parse(newest.String)
	if err != nil {
		return err
	}
	where, args := pkWhere(t, pk)
	_, err = tx.ExecContext(ctx, `UPDATE `+t.Name+` SET updated_at = ? WHERE `+where,
		append([]any{db.FormatTime(ts.Time())}, args...)...)
	return err
}

// AddPending stores a change that is waiting for a missing row.
func AddPending(ctx context.Context, tx *sql.Tx, scope db.Scope, ch Change, reason string) error {
	data, err := json.Marshal(ch)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sync_pending (scope, device_id, clock_epoch, hlc, op, reason) VALUES (?, ?, ?, ?, ?, ?)`,
		scope, ch.Clock.Device, ch.Clock.Epoch, ch.Clock.HLC, string(data), reason)
	return err
}

// RetryPending applies waiting changes until no more of them can be applied
// (§13.3 step 5: retried after each apply round). It returns the number of
// changes still waiting.
func (a *Applier) RetryPending(scopes []db.Scope) (int, error) {
	type entry struct {
		id int64
		ch Change
	}
	for {
		query := `SELECT id, op FROM sync_pending`
		var args []any
		if scopes != nil {
			if len(scopes) == 0 {
				return 0, nil
			}
			query += ` WHERE scope IN (` + placeholders(len(scopes)) + `)`
			for _, s := range scopes {
				args = append(args, s)
			}
		}
		rows, err := a.tx.QueryContext(a.ctx, query+` ORDER BY id`, args...)
		if err != nil {
			return 0, err
		}
		var entries []entry
		for rows.Next() {
			var e entry
			var data string
			if err := rows.Scan(&e.id, &data); err != nil {
				rows.Close()
				return 0, err
			}
			if err := decodeChange(data, &e.ch); err != nil {
				rows.Close()
				return 0, err
			}
			entries = append(entries, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return 0, err
		}
		progress := false
		for _, e := range entries {
			res, err := a.Apply(e.ch)
			if err != nil {
				return 0, err
			}
			if res.Pending != "" {
				continue
			}
			progress = true
			if _, err := a.tx.ExecContext(a.ctx, `DELETE FROM sync_pending WHERE id = ?`, e.id); err != nil {
				return 0, err
			}
		}
		if !progress {
			return len(entries), nil
		}
	}
}

func decodeChange(data string, ch *Change) error {
	if err := json.Unmarshal([]byte(data), ch); err != nil {
		return err
	}
	// Re-read values with integral numbers as int64, as ops carry them.
	var raw struct {
		PK  json.RawMessage `json:"pk"`
		Set json.RawMessage `json:"set"`
	}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return err
	}
	pk, err := op.DecodePK(string(raw.PK))
	if err != nil {
		return err
	}
	set, err := op.DecodePK(string(raw.Set))
	if err != nil {
		return err
	}
	ch.PK, ch.Set = pk, set
	return nil
}

func pkWhere(t db.Table, pk map[string]any) (string, []any) {
	parts := make([]string, len(t.PK))
	args := make([]any, len(t.PK))
	for i, c := range t.PK {
		parts[i] = c + ` = ?`
		args[i] = pk[c]
	}
	return strings.Join(parts, ` AND `), args
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}
