package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func openTemp(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "club.db")
	conn, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, path
}

func TestOpenConfiguresConnection(t *testing.T) {
	conn, _ := openTemp(t)

	var mode string
	if err := conn.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal_mode = %q (%v), want wal", mode, err)
	}
	var fk int
	if err := conn.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d (%v), want 1", fk, err)
	}
	var timeout int
	if err := conn.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil || timeout != int(busyTimeout.Milliseconds()) {
		t.Errorf("busy_timeout = %d (%v), want %d", timeout, err, busyTimeout.Milliseconds())
	}

	// Foreign keys are enforced.
	_, err := conn.Exec(`INSERT INTO member_contacts (member_id) VALUES ('missing')`)
	if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Errorf("insert with dangling member_id: err = %v, want FOREIGN KEY violation", err)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	conn, path := openTemp(t)
	if _, err := conn.Exec(`INSERT INTO roles (id, name) VALUES (?, 'Trainer')`, NewID()); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	again, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer again.Close()
	var n int
	if err := again.QueryRow(`SELECT COUNT(*) FROM roles`).Scan(&n); err != nil || n != 1 {
		t.Errorf("roles after reopen = %d (%v), want 1", n, err)
	}
}

func tableColumns(t *testing.T, conn *sql.DB, table string) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	return cols
}

// Every application table must be classified as synced (with a scope) or
// local, and synced tables must carry the columns the sync layer relies on.
func TestSchemaMatchesScopeRegistry(t *testing.T) {
	conn, _ := openTemp(t)

	rows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type = 'table'
		AND name NOT LIKE 'sqlite_%' AND name != 'goose_db_version'`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	rows.Close()

	known := map[string]Table{}
	for _, st := range SyncedTables {
		known[st.Name] = st
	}
	for _, name := range tables {
		if _, ok := known[name]; !ok && !slices.Contains(LocalTables, name) {
			t.Errorf("table %s is neither in SyncedTables nor in LocalTables", name)
		}
	}

	for _, name := range LocalTables {
		if _, ok := known[name]; ok {
			t.Errorf("table %s is both synced and local", name)
		}
		if !slices.Contains(tables, name) {
			t.Errorf("LocalTables lists %s, which does not exist", name)
		}
	}

	for _, st := range SyncedTables {
		if !slices.Contains(tables, st.Name) {
			t.Errorf("SyncedTables lists %s, which does not exist", st.Name)
			continue
		}
		if !slices.Contains(Scopes, st.Scope) {
			t.Errorf("%s has unknown scope %q", st.Name, st.Scope)
		}
		if pk := primaryKey(t, conn, st.Name); !slices.Equal(pk, st.PK) {
			t.Errorf("%s: registry PK %v, schema PK %v", st.Name, st.PK, pk)
		}
		cols := tableColumns(t, conn, st.Name)
		want := []string{"updated_at", "deleted_at"}
		if st.AppendOnly {
			want = []string{"created_at"}
		}
		for _, c := range want {
			if !slices.Contains(cols, c) {
				t.Errorf("synced table %s lacks column %s", st.Name, c)
			}
		}
	}

	// Banking data must not leak into tables of other scopes.
	for _, st := range SyncedTables {
		if st.Scope == ScopeMembersBank {
			continue
		}
		for _, c := range tableColumns(t, conn, st.Name) {
			if c == "iban" || c == "bic" || c == "sepa_mandate_date" {
				t.Errorf("banking column %s found in %s (scope %s)", c, st.Name, st.Scope)
			}
		}
	}
}

// The scope assignments required by docs/sync-protocol.md §4.1 and §22.
func TestScopeAssignments(t *testing.T) {
	want := map[string]Scope{
		"members": ScopeMembersCore, "member_roles": ScopeMembersCore,
		"member_contacts": ScopeMembersContact, "member_bank_details": ScopeMembersBank,
		"finance_transactions": ScopeFinance, "member_fees": ScopeFinance,
		"inventory_items": ScopeInventory, "inventory_loans": ScopeInventory,
	}
	for name, scope := range want {
		if st, ok := SyncedTable(name); !ok || st.Scope != scope {
			t.Errorf("%s: scope %q (synced %v), want %q", name, st.Scope, ok, scope)
		}
	}
	for _, name := range []string{"users", "audit_log", "user_preferences", "refresh_tokens", "password_reset_tokens"} {
		if _, ok := SyncedTable(name); ok || !slices.Contains(LocalTables, name) {
			t.Errorf("%s must be local only", name)
		}
	}
}

// Synced tables must not carry UNIQUE indexes, since applying valid changes
// from other devices must never fail on them (§14.2). Every uniqueness rule
// still has an index for the pre-save check.
func TestNoUniqueIndexesOnSyncedTables(t *testing.T) {
	conn, _ := openTemp(t)
	for _, st := range SyncedTables {
		rows, err := conn.Query(`SELECT name, "unique", origin FROM pragma_index_list(?)`, st.Name)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var name, origin string
			var unique bool
			if err := rows.Scan(&name, &unique, &origin); err != nil {
				t.Fatal(err)
			}
			if unique && origin != "pk" {
				t.Errorf("synced table %s has unique index %s", st.Name, name)
			}
		}
		rows.Close()
	}
	for _, u := range UniqueColumns {
		if _, ok := SyncedTable(u.Table); !ok {
			t.Errorf("unique rule on %s, which is not synced", u.Table)
		}
		var n int
		err := conn.QueryRow(`SELECT COUNT(*) FROM pragma_index_list(?) l, pragma_index_info(l.name) i
			WHERE i.name = ? AND i.seqno = 0`, u.Table, u.Column).Scan(&n)
		if err != nil || n == 0 {
			t.Errorf("no index on %s.%s (%v)", u.Table, u.Column, err)
		}
	}

	// Duplicates are accepted by SQLite now; the store prevents them locally.
	for _, id := range []string{NewID(), NewID()} {
		if _, err := conn.Exec(`INSERT INTO members (id, number, first_name, last_name) VALUES (?, 1, 'A', 'B')`, id); err != nil {
			t.Errorf("duplicate member number rejected by SQLite: %v", err)
		}
	}
}

func primaryKey(t *testing.T, conn *sql.DB, table string) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT name FROM pragma_table_info(?) WHERE pk > 0 ORDER BY pk`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	return cols
}

// legacySchema is the schema the server created before the goose migrations
// (abridged to the tables exercised below, column names unchanged).
var legacySchema = []string{
	`CREATE TABLE "User" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "email" TEXT NOT NULL UNIQUE,
		"password" TEXT NOT NULL, "appRole" TEXT NOT NULL DEFAULT 'READONLY',
		"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
	`CREATE TABLE "Member" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "number" INTEGER NOT NULL UNIQUE,
		"firstName" TEXT NOT NULL, "lastName" TEXT NOT NULL, "email" TEXT NOT NULL UNIQUE, "birthday" DATETIME,
		"phone" TEXT, "phoneMobile" TEXT, "comment" TEXT, "entryDate" DATETIME, "exitDate" DATETIME, "street" TEXT,
		"postalCode" TEXT, "city" TEXT, "state" TEXT, "accountHolder" TEXT, "iban" TEXT, "bic" TEXT,
		"bankName" TEXT, "sepaMandateDate" DATETIME)`,
	`CREATE TABLE "Role" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL UNIQUE)`,
	`CREATE TABLE "Group" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL UNIQUE)`,
	`CREATE TABLE "_MemberRoles" ("A" INTEGER NOT NULL, "B" INTEGER NOT NULL, UNIQUE("A","B"))`,
	`CREATE TABLE "_MemberGroups" ("A" INTEGER NOT NULL, "B" INTEGER NOT NULL, UNIQUE("A","B"))`,
	`CREATE TABLE "FinanceTransaction" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "date" DATETIME NOT NULL,
		"description" TEXT NOT NULL, "amount" REAL NOT NULL, "type" TEXT NOT NULL, "category" TEXT, "notes" TEXT,
		"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
	`CREATE TABLE "MemberFee" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "memberId" INTEGER NOT NULL,
		"amount" REAL NOT NULL, "dueDate" DATETIME NOT NULL, "paidDate" DATETIME, "description" TEXT,
		"year" INTEGER NOT NULL, "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
	`CREATE TABLE "UserPreference" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "userId" INTEGER NOT NULL,
		"key" TEXT NOT NULL, "value" TEXT NOT NULL, "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE("userId", "key"))`,
	`CREATE TABLE "AuditLog" ("id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "action" TEXT NOT NULL,
		"entity" TEXT NOT NULL, "entityId" INTEGER NOT NULL, "userId" TEXT NOT NULL, "data" TEXT,
		"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
}

func TestOpenImportsLegacySchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "club.db")

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	stmts := append(legacySchema,
		`INSERT INTO "User" ("id","email","password","appRole") VALUES (7,'kassierer@example.org','hash','TREASURER')`,
		`INSERT INTO "Member" ("id","number","firstName","lastName","email","birthday","city","iban","sepaMandateDate")
			VALUES (3,101,'Ada','Lovelace','ada@example.org','1815-12-10T00:00:00Z','London','DE02120300000000202051','2024-01-15T00:00:00Z')`,
		`INSERT INTO "Member" ("id","number","firstName","lastName","email") VALUES (4,102,'Alan','Turing','alan@example.org')`,
		`INSERT INTO "Role" ("id","name") VALUES (1,'Vorstand')`,
		`INSERT INTO "Group" ("id","name") VALUES (5,'Jugend')`,
		`INSERT INTO "_MemberRoles" ("A","B") VALUES (3,1)`,  // A = Member, B = Role
		`INSERT INTO "_MemberGroups" ("A","B") VALUES (5,4)`, // A = Group, B = Member
		`INSERT INTO "_MemberGroups" ("A","B") VALUES (5,99)`,
		`INSERT INTO "FinanceTransaction" ("id","date","description","amount","type") VALUES (1,'2025-03-01T00:00:00Z','Spende',12.345,'income')`,
		// Prisma stored DateTime as Unix epoch milliseconds.
		`INSERT INTO "MemberFee" ("id","memberId","amount","dueDate","year") VALUES (1,3,60.1,1735689600000,2025)`,
		`INSERT INTO "UserPreference" ("userId","key","value") VALUES (1,'theme','"dark"')`,
		`INSERT INTO "AuditLog" ("action","entity","entityId","userId") VALUES ('CREATE','Member',3,'admin@example.org')`,
	)
	for _, s := range stmts {
		if _, err := legacy.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	legacy.Close()

	conn, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	backups, _ := filepath.Glob(filepath.Join(dir, "club.pre-uuid-*.db"))
	if len(backups) != 1 {
		t.Errorf("backups = %v, want exactly one", backups)
	} else if fi, err := os.Stat(backups[0]); err != nil || fi.Size() == 0 {
		t.Errorf("backup %s unusable: %v", backups[0], err)
	}

	if ok, _ := tableExists(context.Background(), conn, "Member"); ok {
		t.Error("legacy table Member still exists")
	}

	var adaID, city, iban, mandate, birthday string
	err = conn.QueryRow(`SELECT m.id, c.city, b.iban, b.sepa_mandate_date, m.birthday FROM members m
		JOIN member_contacts c ON c.member_id = m.id
		JOIN member_bank_details b ON b.member_id = m.id
		WHERE m.number = 101`).Scan(&adaID, &city, &iban, &mandate, &birthday)
	if err != nil {
		t.Fatalf("query imported member: %v", err)
	}
	if city != "London" || iban != "DE02120300000000202051" || mandate != "2024-01-15T00:00:00.000Z" || birthday != "1815-12-10T00:00:00.000Z" {
		t.Errorf("imported member = %q %q %q %q", city, iban, mandate, birthday)
	}

	var bankRows int
	conn.QueryRow(`SELECT COUNT(*) FROM member_bank_details`).Scan(&bankRows)
	if bankRows != 1 {
		t.Errorf("member_bank_details rows = %d, want 1 (only members with bank data)", bankRows)
	}

	var role, group string
	conn.QueryRow(`SELECT r.name FROM member_roles l JOIN roles r ON r.id = l.role_id WHERE l.member_id = ?`, adaID).Scan(&role)
	conn.QueryRow(`SELECT g.name FROM member_groups l JOIN groups g ON g.id = l.group_id
		JOIN members m ON m.id = l.member_id WHERE m.number = 102`).Scan(&group)
	if role != "Vorstand" || group != "Jugend" {
		t.Errorf("links: role %q group %q", role, group)
	}
	var links int
	conn.QueryRow(`SELECT COUNT(*) FROM member_groups`).Scan(&links)
	if links != 1 {
		t.Errorf("member_groups rows = %d, want 1 (dangling link dropped)", links)
	}

	var cents int64
	conn.QueryRow(`SELECT amount_cents FROM finance_transactions`).Scan(&cents)
	if cents != 1235 {
		t.Errorf("amount_cents = %d, want 1235", cents)
	}

	var feeMember, dueDate string
	var feeCents int64
	conn.QueryRow(`SELECT member_id, amount_cents, due_date FROM member_fees`).Scan(&feeMember, &feeCents, &dueDate)
	if feeMember != adaID || feeCents != 6010 || dueDate != "2025-01-01T00:00:00.000Z" {
		t.Errorf("fee = %s %d %s", feeMember, feeCents, dueDate)
	}

	var role7 string
	conn.QueryRow(`SELECT app_role FROM users WHERE email = 'kassierer@example.org'`).Scan(&role7)
	if role7 != "TREASURER" {
		t.Errorf("user role = %q", role7)
	}

	var prefUser string
	conn.QueryRow(`SELECT user_id FROM user_preferences WHERE key = 'theme'`).Scan(&prefUser)
	if prefUser != AdminUserID {
		t.Errorf("preference user = %q, want admin", prefUser)
	}

	var auditEntity string
	conn.QueryRow(`SELECT entity_id FROM audit_log`).Scan(&auditEntity)
	if auditEntity != adaID {
		t.Errorf("audit entity_id = %q, want %q", auditEntity, adaID)
	}
}
