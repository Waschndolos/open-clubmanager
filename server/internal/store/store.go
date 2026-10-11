// Package store is the repository layer on top of the local SQLite database.
// It owns all SQL; handlers translate between its types and the API types.
//
// Rows of synced tables are never deleted: Delete methods set deleted_at
// (a tombstone the future sync layer replicates) and every read filters on
// deleted_at IS NULL. All writes bump updated_at.
package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

var (
	// ErrNotFound is returned when the requested (live) row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a write violates a uniqueness constraint.
	ErrConflict = errors.New("conflict")
	// ErrInvalidReference is returned when a write references a row that does
	// not exist.
	ErrInvalidReference = errors.New("invalid reference")
)

// Store provides typed access to the database.
type Store struct {
	db *sql.DB
}

// New wraps an open database (see db.Open).
func New(conn *sql.DB) *Store {
	return &Store{db: conn}
}

// inTx runs fn in a transaction, committing when it returns nil.
func (s *Store) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// mapErr translates SQLite constraint violations into the package errors.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return ErrConflict
		case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return ErrInvalidReference
		}
	}
	return err
}

// requireAffected returns ErrNotFound when res touched no rows.
func requireAffected(res sql.Result, err error) error {
	if err != nil {
		return mapErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func timeArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return db.FormatTime(*t)
}

func strArg(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func centsArg(v *float64) any {
	if v == nil {
		return nil
	}
	return db.ToCents(*v)
}

func parseTime(s string) time.Time {
	t, _ := db.ParseTime(s)
	return t
}

func nullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, ok := db.ParseTime(ns.String)
	if !ok {
		return nil
	}
	return &t
}

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func nullAmount(n sql.NullInt64) *float64 {
	if !n.Valid {
		return nil
	}
	v := db.FromCents(n.Int64)
	return &v
}

// placeholders returns "?, ?, ..." for n arguments.
func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}

func stringArgs(ids []string) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// setLinks makes the live links in a member_* link table from fixedCol=fixedID
// exactly the given otherIDs: missing links are tombstoned, new ones inserted
// and previously tombstoned ones revived.
func setLinks(ctx context.Context, tx execer, table, fixedCol, fixedID, otherCol string, otherIDs []string, now string) error {
	args := append([]any{now, now, fixedID}, stringArgs(otherIDs)...)
	query := `UPDATE ` + table + ` SET deleted_at = ?, updated_at = ? WHERE ` + fixedCol + ` = ? AND deleted_at IS NULL`
	if len(otherIDs) > 0 {
		query += ` AND ` + otherCol + ` NOT IN (` + placeholders(len(otherIDs)) + `)`
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return mapErr(err)
	}
	for _, id := range otherIDs {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO `+table+` (`+fixedCol+`, `+otherCol+`, updated_at) VALUES (?, ?, ?)
			 ON CONFLICT DO UPDATE SET deleted_at = NULL, updated_at = excluded.updated_at WHERE deleted_at IS NOT NULL`,
			fixedID, id, now)
		if err != nil {
			return mapErr(err)
		}
	}
	return nil
}
