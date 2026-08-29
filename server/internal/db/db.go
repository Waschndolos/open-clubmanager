// Package db manages the database connection and schema migrations for the
// open-clubmanager server. It supports SQLite (local desktop) and MySQL
// (shared / server deployments).
package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// Open returns a ready-to-use *sql.DB for the given mode and DSN.
// mode is either "sqlite-local" or "mysql-shared".
// For SQLite the DSN may include a "file:" prefix or plain path.
func Open(mode, dsn string) (*sql.DB, error) {
	switch mode {
	case "sqlite-local":
		return openSQLite(dsn)
	case "mysql-shared":
		return openMySQL(dsn)
	default:
		return nil, fmt.Errorf("unsupported database mode: %s", mode)
	}
}

func openSQLite(dsn string) (*sql.DB, error) {
	path := strings.TrimPrefix(dsn, "file:")
	path = strings.TrimPrefix(path, "//")

	// Enable foreign keys and WAL mode for better concurrent read performance.
	sqliteDSN := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", url.PathEscape(path))
	db, err := sql.Open("sqlite", sqliteDSN)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes well
	return db, nil
}

func openMySQL(dsn string) (*sql.DB, error) {
	// Accept both "mysql://user:pass@host:port/dbname" and raw DSN formats.
	if strings.HasPrefix(dsn, "mysql://") || strings.HasPrefix(dsn, "mysqls://") {
		dsn = convertMySQLURL(dsn)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	return db, nil
}

// convertMySQLURL converts a mysql:// URL to the go-sql-driver DSN format.
// mysql://user:pass@host:3306/dbname  →  user:pass@tcp(host:3306)/dbname?parseTime=true
func convertMySQLURL(rawURL string) string {
	rawURL = strings.Replace(rawURL, "mysqls://", "mysql://", 1)
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	host := u.Host
	dbName := strings.TrimPrefix(u.Path, "/")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true", user, pass, host, dbName)
	return dsn
}

// Migrate applies all CREATE TABLE IF NOT EXISTS statements so the schema is
// always up-to-date regardless of whether Prisma migrations have been run.
func Migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS "User" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"email" TEXT NOT NULL UNIQUE,
			"password" TEXT NOT NULL,
			"appRole" TEXT NOT NULL DEFAULT 'READONLY',
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS "RefreshToken" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"token" TEXT NOT NULL UNIQUE,
			"userId" INTEGER NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"expiresAt" DATETIME NOT NULL,
			FOREIGN KEY ("userId") REFERENCES "User" ("id")
		)`,
		`CREATE TABLE IF NOT EXISTS "PasswordResetToken" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"token" TEXT NOT NULL UNIQUE,
			"userId" INTEGER NOT NULL,
			"expiresAt" DATETIME NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY ("userId") REFERENCES "User" ("id") ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS "Member" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"number" INTEGER NOT NULL UNIQUE,
			"firstName" TEXT NOT NULL,
			"lastName" TEXT NOT NULL,
			"email" TEXT NOT NULL UNIQUE,
			"birthday" DATETIME,
			"phone" TEXT,
			"phoneMobile" TEXT,
			"comment" TEXT,
			"entryDate" DATETIME,
			"exitDate" DATETIME,
			"street" TEXT,
			"postalCode" TEXT,
			"city" TEXT,
			"state" TEXT,
			"accountHolder" TEXT,
			"iban" TEXT,
			"bic" TEXT,
			"bankName" TEXT,
			"sepaMandateDate" DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS "Role" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS "Group" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS "ClubSection" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS "UserPreference" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"userId" INTEGER NOT NULL,
			"key" TEXT NOT NULL,
			"value" TEXT NOT NULL,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE("userId", "key")
		)`,
		`CREATE TABLE IF NOT EXISTS "_MemberRoles" (
			"A" INTEGER NOT NULL,
			"B" INTEGER NOT NULL,
			FOREIGN KEY ("A") REFERENCES "Member" ("id") ON DELETE CASCADE,
			FOREIGN KEY ("B") REFERENCES "Role" ("id") ON DELETE CASCADE,
			UNIQUE("A","B")
		)`,
		`CREATE TABLE IF NOT EXISTS "_MemberGroups" (
			"A" INTEGER NOT NULL,
			"B" INTEGER NOT NULL,
			FOREIGN KEY ("A") REFERENCES "Member" ("id") ON DELETE CASCADE,
			FOREIGN KEY ("B") REFERENCES "Group" ("id") ON DELETE CASCADE,
			UNIQUE("A","B")
		)`,
		`CREATE TABLE IF NOT EXISTS "_MemberSections" (
			"A" INTEGER NOT NULL,
			"B" INTEGER NOT NULL,
			FOREIGN KEY ("A") REFERENCES "Member" ("id") ON DELETE CASCADE,
			FOREIGN KEY ("B") REFERENCES "ClubSection" ("id") ON DELETE CASCADE,
			UNIQUE("A","B")
		)`,
		`CREATE TABLE IF NOT EXISTS "FinanceTransaction" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"date" DATETIME NOT NULL,
			"description" TEXT NOT NULL,
			"amount" REAL NOT NULL,
			"type" TEXT NOT NULL,
			"category" TEXT,
			"notes" TEXT,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS "MemberFee" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"memberId" INTEGER NOT NULL,
			"amount" REAL NOT NULL,
			"dueDate" DATETIME NOT NULL,
			"paidDate" DATETIME,
			"description" TEXT,
			"year" INTEGER NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY ("memberId") REFERENCES "Member" ("id") ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS "InventoryItem" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT NOT NULL,
			"description" TEXT,
			"serialNumber" TEXT,
			"category" TEXT NOT NULL,
			"quantity" INTEGER NOT NULL,
			"location" TEXT NOT NULL,
			"purchaseDate" DATETIME,
			"purchasePrice" REAL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS "InventoryLoan" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"itemId" INTEGER NOT NULL,
			"memberId" INTEGER NOT NULL,
			"loanedAt" DATETIME NOT NULL,
			"dueDate" DATETIME,
			"returnedAt" DATETIME,
			"notes" TEXT NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY ("itemId") REFERENCES "InventoryItem" ("id") ON DELETE CASCADE,
			FOREIGN KEY ("memberId") REFERENCES "Member" ("id") ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS "AuditLog" (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"action" TEXT NOT NULL,
			"entity" TEXT NOT NULL,
			"entityId" INTEGER NOT NULL,
			"userId" TEXT NOT NULL,
			"data" TEXT,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Additive column migrations: silently ignore errors when the column already exists.
	additiveMigrations := []string{
		`ALTER TABLE "User" ADD COLUMN "appRole" TEXT NOT NULL DEFAULT 'READONLY'`,
	}
	for _, stmt := range additiveMigrations {
		_, _ = db.Exec(stmt) // intentionally ignore "duplicate column" errors
	}

	return nil
}
