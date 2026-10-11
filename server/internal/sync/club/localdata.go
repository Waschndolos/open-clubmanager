package club

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

// ErrLocalData is returned when joining a club while synced tables already
// hold data. A joining device MUST start empty (§17.2, §15.4); the UI offers
// an export or backup and then calls ClearSyncedData.
var ErrLocalData = errors.New("club: synced tables are not empty; export and clear them before joining")

// requireEmptySyncedData enforces that a joining device has no synced rows
// and no pending local ops.
func requireEmptySyncedData(ctx context.Context, conn *sql.DB) error {
	tables := []string{"sync_outbox"}
	for _, t := range db.SyncedTables {
		tables = append(tables, t.Name)
	}
	for _, t := range tables {
		var n int
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+t+`)`).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return fmt.Errorf("%w (%s)", ErrLocalData, t)
		}
	}
	return nil
}

// ClearSyncedData deletes all rows of synced tables together with their
// field clocks, detected conflicts and the outbox, so that the device can
// join a club. The caller must have offered an export or backup first.
func ClearSyncedData(ctx context.Context, conn *sql.DB) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
		return err
	}
	for _, t := range slices.Backward(db.SyncedTables) {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+t.Name); err != nil {
			return err
		}
	}
	for _, t := range []string{"sync_field_clock", "sync_conflicts", "sync_outbox"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// adoptLocalData prepares the data of the creating device for the new club
// (§17.1, §15.4): every non-NULL column of a synced row that has no field
// clock yet gets one at the creation HLC, and the outbox is cleared, because
// the initial snapshot carries the full state.
func adoptLocalData(ctx context.Context, tx *sql.Tx, deviceID string, created hlc.Timestamp) error {
	for _, t := range db.SyncedTables {
		if err := backfillClocks(ctx, tx, t, deviceID, created); err != nil {
			return fmt.Errorf("club: backfill field clocks of %s: %w", t.Name, err)
		}
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM sync_outbox`)
	return err
}

func backfillClocks(ctx context.Context, tx *sql.Tx, t db.Table, deviceID string, created hlc.Timestamp) error {
	rows, err := tx.QueryContext(ctx, `SELECT * FROM `+t.Name)
	if err != nil {
		return err
	}
	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return err
	}
	type clock struct{ pk, col string }
	var clocks []clock
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return err
		}
		pk := map[string]any{}
		for i, c := range cols {
			if slices.Contains(t.PK, c) {
				pk[c] = op.Normalize(vals[i])
			}
		}
		pkText := op.EncodePK(pk)
		for i, c := range cols {
			// Ops never carry updated_at (§10), so it has no clock.
			if c != "updated_at" && vals[i] != nil {
				clocks = append(clocks, clock{pkText, c})
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, c := range clocks {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (tbl, pk, col) DO NOTHING`,
			t.Name, c.pk, c.col, created.String(), deviceID); err != nil {
			return err
		}
	}
	return nil
}
