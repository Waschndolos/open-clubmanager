// Package db opens the local SQLite database and keeps its schema up to date
// with the embedded goose migrations in migrations/.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// busyTimeout is how long a connection waits for a lock before failing with
// SQLITE_BUSY.
const busyTimeout = 5 * time.Second

// CleanPath turns a configured database path (which may carry a "file:" or
// "file://" prefix) into a plain file-system path.
func CleanPath(p string) string {
	p = strings.TrimPrefix(p, "file:")
	p = strings.TrimPrefix(p, "//")
	return filepath.Clean(p)
}

// Open opens (creating if necessary) the SQLite database at path with WAL
// journaling, enforced foreign keys and a busy timeout, and applies all pending
// migrations. A database still using the pre-UUID schema is backed up next to
// the original file and its data imported into the new schema.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	path = CleanPath(path)
	if path == "" || path == "." {
		return nil, errors.New("database path is empty")
	}

	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeout.Milliseconds()))
	q.Add("_pragma", "synchronous(NORMAL)")
	// Take the write lock at BEGIN so concurrent writers wait on busy_timeout
	// instead of failing when upgrading a read transaction.
	q.Set("_txlock", "immediate")

	// A plain path (no "file:" URI) keeps Windows paths and characters such as
	// '#' working; the driver strips the query part before opening.
	db, err := sql.Open("sqlite", path+"?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// A single connection serialises access within this process; WAL still lets
	// external readers (backups, tooling) proceed.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := migrate(ctx, db, path); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB, path string) error {
	legacy, err := hasLegacySchema(ctx, db)
	if err != nil {
		return err
	}
	if legacy {
		backup := legacyBackupPath(path)
		if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, backup); err != nil {
			return fmt.Errorf("backup legacy database to %s: %w", backup, err)
		}
		log.Printf("db: legacy schema detected, backup written to %s", backup)
	}

	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		log.Printf("db: applied migration %s (%s)", r.Source.Path, r.Duration)
	}

	if legacy {
		if err := importLegacy(ctx, db); err != nil {
			return fmt.Errorf("import legacy data: %w", err)
		}
		log.Printf("db: legacy data imported")
	}
	return nil
}

func legacyBackupPath(path string) string {
	ext := filepath.Ext(path)
	return fmt.Sprintf("%s.pre-uuid-%s%s", strings.TrimSuffix(path, ext), time.Now().UTC().Format("20060102-150405"), ext)
}
