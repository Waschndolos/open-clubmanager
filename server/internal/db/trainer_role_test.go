package db

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

// TestTrainerRoleMigration checks the rebuild of the users table: data and
// the cascading references of the token tables survive, foreign keys stay on
// and the new role is accepted.
func TestTrainerRoleMigration(t *testing.T) {
	ctx := context.Background()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "club.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	migrations, _ := fs.Sub(embedMigrations, "migrations")
	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 10); err != nil {
		t.Fatal(err)
	}
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	mustExec(`INSERT INTO users (id, email, password, app_role) VALUES ('u1', 'a@example.org', 'x', 'SECRETARY')`)
	mustExec(`INSERT INTO refresh_tokens (id, token, user_id, expires_at) VALUES ('r1', 't1', 'u1', '2030-01-01')`)
	if _, err := conn.Exec(`INSERT INTO users (id, email, password, app_role) VALUES ('u2', 'b@example.org', 'x', 'TRAINER')`); err == nil {
		t.Fatal("TRAINER accepted before the migration")
	}

	if _, err := provider.UpTo(ctx, 11); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var fk int
	if err := conn.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d (%v) after the migration", fk, err)
	}
	var role string
	if err := conn.QueryRow(`SELECT app_role FROM users WHERE id = 'u1'`).Scan(&role); err != nil || role != "SECRETARY" {
		t.Errorf("existing user = %q, %v", role, err)
	}
	mustExec(`INSERT INTO users (id, email, password, app_role) VALUES ('u2', 'b@example.org', 'x', 'TRAINER')`)
	if _, err := conn.Exec(`INSERT INTO users (id, email, password, app_role) VALUES ('u3', 'c@example.org', 'x', 'GUEST')`); err == nil {
		t.Error("unknown role accepted")
	}
	if _, err := conn.Exec(`INSERT INTO refresh_tokens (id, token, user_id, expires_at) VALUES ('r2', 't2', 'nobody', '2030-01-01')`); err == nil {
		t.Error("token of an unknown user accepted: the foreign key is lost")
	}
	if _, err := conn.Exec(`INSERT INTO users (id, email, password) VALUES ('u4', 'a@example.org', 'x')`); err == nil {
		t.Error("duplicate live e-mail accepted: the unique index is lost")
	}
	mustExec(`DELETE FROM users WHERE id = 'u1'`)
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM refresh_tokens`).Scan(&n); err != nil || n != 0 {
		t.Errorf("refresh tokens after deleting the user = %d (%v), want 0 (cascade)", n, err)
	}

	// Down maps trainers to READONLY.
	if _, err := provider.DownTo(ctx, 10); err != nil {
		t.Fatalf("down: %v", err)
	}
	if err := conn.QueryRow(`SELECT app_role FROM users WHERE id = 'u2'`).Scan(&role); err != nil || role != "READONLY" {
		t.Errorf("trainer after down = %q, %v", role, err)
	}
}
