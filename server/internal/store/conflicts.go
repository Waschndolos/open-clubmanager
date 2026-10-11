package store

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

// Conflict is a row flagged because it holds a unique value that another live
// row holds with an older field clock (docs/sync-protocol.md §14.2).
type Conflict struct {
	Table      string
	PK         string // op.EncodePK form
	Column     string
	Value      any
	KeptPK     string // the row that keeps the value
	DetectedAt time.Time
}

// ListConflicts returns all open uniqueness conflicts.
func (s *Store) ListConflicts(ctx context.Context) ([]Conflict, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tbl, pk, col, value, kept_pk, detected_at FROM sync_conflicts ORDER BY tbl, col, value, pk`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Conflict{}
	for rows.Next() {
		var c Conflict
		var value, detectedAt string
		if err := rows.Scan(&c.Table, &c.PK, &c.Column, &value, &c.KeptPK, &detectedAt); err != nil {
			return nil, err
		}
		if c.Value, err = op.DecodeValue(value); err != nil {
			return nil, err
		}
		c.DetectedAt = parseTime(detectedAt)
		result = append(result, c)
	}
	return result, rows.Err()
}

// DetectUniqueConflicts runs the uniqueness checks of §14.2 for the given rows
// (primary keys in op.EncodePK form) of a synced table after changes were
// applied. For every unique column of the table it looks at each value held by
// one of the rows, or previously flagged for one of them, and among the live
// rows holding that value the one whose value was set with the smallest
// (hlc, device_id) keeps it; every other row is recorded in sync_conflicts.
// Flags that no longer apply are removed. The result depends only on the
// database content, so every device reaches the same decision.
//
// Rows without a field clock (written before sync existed) count as oldest.
// Equal clocks, which only occur for such rows, are ordered by primary key.
func DetectUniqueConflicts(ctx context.Context, tx *sql.Tx, table string, pks []string) error {
	t, ok := db.SyncedTable(table)
	if !ok || len(pks) == 0 {
		return nil
	}
	for _, rule := range db.UniqueColumns {
		if rule.Table != table {
			continue
		}
		values, err := conflictCandidates(ctx, tx, t, rule.Column, pks)
		if err != nil {
			return err
		}
		for _, v := range values {
			if err := resolveUniqueValue(ctx, tx, t, rule.Column, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// conflictCandidates returns the distinct values (EncodeValue form) to
// re-check: the current live values of the rows and the values they were
// flagged for or kept before.
func conflictCandidates(ctx context.Context, tx *sql.Tx, t db.Table, col string, pks []string) ([]string, error) {
	seen := map[string]bool{}
	var values []string
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			values = append(values, v)
		}
	}
	for _, pkText := range pks {
		pk, err := op.DecodePK(pkText)
		if err != nil {
			return nil, err
		}
		var v any
		err = tx.QueryRowContext(ctx,
			`SELECT `+col+` FROM `+t.Name+` WHERE `+pkWhere(t)+` AND deleted_at IS NULL`, pkArgs(t, pk)...).Scan(&v)
		switch {
		case err == sql.ErrNoRows:
		case err != nil:
			return nil, err
		case v != nil:
			add(op.EncodeValue(v))
		}
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT value FROM sync_conflicts WHERE tbl = ? AND col = ?
		   AND (pk IN (`+placeholders(len(pks))+`) OR kept_pk IN (`+placeholders(len(pks))+`))`,
		append([]any{t.Name, col}, append(stringArgs(pks), stringArgs(pks)...)...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		add(v)
	}
	return values, rows.Err()
}

// resolveUniqueValue recomputes the conflicts for one value of a column.
func resolveUniqueValue(ctx context.Context, tx *sql.Tx, t db.Table, col, valueText string) error {
	value, err := op.DecodeValue(valueText)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT `+strings.Join(t.PK, ", ")+` FROM `+t.Name+` WHERE `+col+` = ? AND deleted_at IS NULL`, value)
	if err != nil {
		return err
	}
	type holder struct {
		pk          string
		hlc, device string
	}
	var holders []holder
	for rows.Next() {
		vals := make([]any, len(t.PK))
		ptrs := make([]any, len(t.PK))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return err
		}
		pk := row{}
		for i, c := range t.PK {
			pk[c] = op.Normalize(vals[i])
		}
		holders = append(holders, holder{pk: op.EncodePK(pk)})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range holders {
		var device sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT hlc, device_id FROM sync_field_clock WHERE tbl = ? AND pk = ? AND col = ?`,
			t.Name, holders[i].pk, col).Scan(&holders[i].hlc, &device)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		holders[i].device = device.String
	}
	// HLC strings sort like timestamps; a missing clock ("") sorts first.
	sort.Slice(holders, func(i, j int) bool {
		a, b := holders[i], holders[j]
		if a.hlc != b.hlc {
			return a.hlc < b.hlc
		}
		if a.device != b.device {
			return a.device < b.device
		}
		return a.pk < b.pk
	})

	var flagged []string
	if len(holders) > 1 {
		kept := holders[0].pk
		for _, h := range holders[1:] {
			flagged = append(flagged, h.pk)
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO sync_conflicts (tbl, pk, col, value, kept_pk) VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT (tbl, pk, col) DO UPDATE SET value = excluded.value, kept_pk = excluded.kept_pk,
				   detected_at = CASE WHEN value = excluded.value THEN detected_at ELSE excluded.detected_at END`,
				t.Name, h.pk, col, valueText, kept); err != nil {
				return err
			}
		}
	}
	query := `DELETE FROM sync_conflicts WHERE tbl = ? AND col = ? AND value = ?`
	args := []any{t.Name, col, valueText}
	if len(flagged) > 0 {
		query += ` AND pk NOT IN (` + placeholders(len(flagged)) + `)`
		args = append(args, stringArgs(flagged)...)
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}
