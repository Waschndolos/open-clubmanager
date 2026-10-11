package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

// row maps column names to values.
type row = map[string]any

// writer performs writes to synced tables inside one transaction. Next to the
// data change it records, in the same transaction, a field clock per written
// column (sync_field_clock) and the op for the sync layer (sync_outbox), as
// required by docs/sync-protocol.md §20. Every write to a synced table must go
// through insert, update or upsert; local tables may be written via the
// embedded transaction.
type writer struct {
	*sql.Tx
	ctx     context.Context
	clock   *hlc.Clock
	ticked  bool
	touched map[string]bool // synced tables written in this transaction
}

// inWriteTx runs fn in a transaction with a writer, committing when it
// returns nil.
func (s *Store) inWriteTx(ctx context.Context, fn func(w *writer) error) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var stored string
		if err := tx.QueryRowContext(ctx, `SELECT hlc FROM sync_device WHERE id = 1`).Scan(&stored); err != nil {
			return fmt.Errorf("read local hlc: %w", err)
		}
		last, err := hlc.Parse(stored)
		if err != nil {
			return err
		}
		s.clock.Restore(last)

		w := &writer{Tx: tx, ctx: ctx, clock: s.clock, touched: map[string]bool{}}
		if err := fn(w); err != nil {
			return err
		}
		if w.ticked {
			if _, err := tx.ExecContext(ctx, `UPDATE sync_device SET hlc = ? WHERE id = 1`, s.clock.Last().String()); err != nil {
				return err
			}
		}
		return w.refreshConflicts()
	})
}

func syncedTable(name string) db.Table {
	t, ok := db.SyncedTable(name)
	if !ok {
		panic("store: " + name + " is not a synced table")
	}
	return t
}

func sortedKeys(m row) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func pkWhere(t db.Table) string {
	parts := make([]string, len(t.PK))
	for i, c := range t.PK {
		parts[i] = c + ` = ?`
	}
	return strings.Join(parts, ` AND `)
}

func pkArgs(t db.Table, pk row) []any {
	args := make([]any, len(t.PK))
	for i, c := range t.PK {
		args[i] = pk[c]
	}
	return args
}

// record writes the field clocks and the outbox entry of one op and returns
// the updated_at value for the row.
func (w *writer) record(t db.Table, pk row, set row) (string, error) {
	ts := w.clock.Now()
	w.ticked = true
	w.touched[t.Name] = true

	pkText := op.EncodePK(pk)
	for col := range set {
		if _, err := w.ExecContext(w.ctx,
			`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id)
			 VALUES (?, ?, ?, ?, (SELECT device_id FROM sync_device WHERE id = 1))
			 ON CONFLICT (tbl, pk, col) DO UPDATE SET hlc = excluded.hlc, device_id = excluded.device_id`,
			t.Name, pkText, col, ts.String()); err != nil {
			return "", err
		}
	}
	data, err := json.Marshal(op.Op{T: t.Name, PK: pk, HLC: ts.String(), Set: set})
	if err != nil {
		return "", err
	}
	if _, err := w.ExecContext(w.ctx,
		`INSERT INTO sync_outbox (scope, hlc, op) VALUES (?, ?, ?)`, string(t.Scope), ts.String(), string(data)); err != nil {
		return "", err
	}
	return db.FormatTime(ts.Time()), nil
}

// insert adds a row. values must contain the primary key columns. The op
// carries every non-NULL column (§10 "Insert").
func (w *writer) insert(table string, values row) error {
	t := syncedTable(table)
	pk, set := row{}, row{}
	for k, v := range values {
		v = op.Normalize(v)
		if slices.Contains(t.PK, k) {
			pk[k] = v
		}
		if v != nil {
			set[k] = v
		}
	}
	updatedAt, err := w.record(t, pk, set)
	if err != nil {
		return err
	}
	cols := sortedKeys(values)
	args := make([]any, 0, len(cols)+1)
	for _, c := range cols {
		args = append(args, values[c])
	}
	if !t.AppendOnly {
		cols = append(cols, "updated_at")
		args = append(args, updatedAt)
	}
	_, err = w.ExecContext(w.ctx,
		`INSERT INTO `+table+` (`+strings.Join(cols, ", ")+`) VALUES (`+placeholders(len(cols))+`)`, args...)
	return mapErr(err)
}

// update sets columns on every row matching where and returns the number of
// matching rows. Only columns whose value actually changes are written, and
// each changed row produces one op; rows without changes keep their
// updated_at.
func (w *writer) update(table, where string, whereArgs []any, set row) (int, error) {
	t := syncedTable(table)
	cols := sortedKeys(set)
	selectCols := append(slices.Clone(t.PK), cols...)
	rows, err := w.QueryContext(w.ctx,
		`SELECT `+strings.Join(selectCols, ", ")+` FROM `+table+` WHERE `+where, whereArgs...)
	if err != nil {
		return 0, err
	}
	type match struct {
		pk      row
		changed row
	}
	var matches []match
	for rows.Next() {
		vals := make([]any, len(selectCols))
		ptrs := make([]any, len(selectCols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return 0, err
		}
		m := match{pk: row{}, changed: row{}}
		for i, c := range t.PK {
			m.pk[c] = op.Normalize(vals[i])
		}
		for i, c := range cols {
			if nv := op.Normalize(set[c]); op.Normalize(vals[len(t.PK)+i]) != nv {
				m.changed[c] = nv
			}
		}
		matches = append(matches, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, m := range matches {
		if len(m.changed) == 0 {
			continue
		}
		updatedAt, err := w.record(t, m.pk, m.changed)
		if err != nil {
			return 0, err
		}
		changedCols := sortedKeys(m.changed)
		assign := make([]string, 0, len(changedCols)+1)
		args := make([]any, 0, len(changedCols)+1+len(t.PK))
		for _, c := range changedCols {
			assign = append(assign, c+` = ?`)
			args = append(args, m.changed[c])
		}
		assign = append(assign, `updated_at = ?`)
		args = append(args, updatedAt)
		args = append(args, pkArgs(t, m.pk)...)
		if _, err := w.ExecContext(w.ctx,
			`UPDATE `+table+` SET `+strings.Join(assign, ", ")+` WHERE `+pkWhere(t), args...); err != nil {
			return 0, mapErr(err)
		}
	}
	return len(matches), nil
}

// upsert updates the row with the given primary key, live or tombstoned, or
// inserts it with pk and set when it does not exist.
func (w *writer) upsert(table string, pk, set row) error {
	t := syncedTable(table)
	n, err := w.update(table, pkWhere(t), pkArgs(t, pk), set)
	if err != nil || n > 0 {
		return err
	}
	values := row{}
	for k, v := range pk {
		values[k] = v
	}
	for k, v := range set {
		values[k] = v
	}
	return w.insert(table, values)
}

// requireUnique returns ErrConflict when a live row other than the one whose
// selfCol equals selfID already holds value in column col (the local pre-save
// check of §14.2). NULL values are never duplicates.
func (w *writer) requireUnique(table, col string, value any, selfCol, selfID string) error {
	if value == nil {
		return nil
	}
	var one int
	err := w.QueryRowContext(w.ctx,
		`SELECT 1 FROM `+table+` WHERE `+col+` = ? AND deleted_at IS NULL AND `+selfCol+` != ? LIMIT 1`,
		value, selfID).Scan(&one)
	switch {
	case err == sql.ErrNoRows:
		return nil
	case err != nil:
		return err
	}
	return ErrConflict
}

// refreshConflicts re-evaluates open uniqueness conflicts of the tables
// written in this transaction, so that a local fix (e.g. a new member
// number) clears its flag. Local writes cannot create new duplicates because
// of requireUnique, so tables without open conflicts are skipped.
func (w *writer) refreshConflicts() error {
	for table := range w.touched {
		var pks []string
		rows, err := w.QueryContext(w.ctx,
			`SELECT pk FROM sync_conflicts WHERE tbl = ? UNION SELECT kept_pk FROM sync_conflicts WHERE tbl = ?`, table, table)
		if err != nil {
			return err
		}
		for rows.Next() {
			var pk string
			if err := rows.Scan(&pk); err != nil {
				rows.Close()
				return err
			}
			pks = append(pks, pk)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(pks) > 0 {
			if err := DetectUniqueConflicts(w.ctx, w.Tx, table, pks); err != nil {
				return err
			}
		}
	}
	return nil
}

// affected turns an update result into ErrNotFound when no row matched.
func affected(n int, err error) error {
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
