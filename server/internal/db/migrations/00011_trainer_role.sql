-- Adds the TRAINER app role. SQLite cannot change a CHECK constraint, so the
-- users table is rebuilt following https://www.sqlite.org/lang_altertable.html
-- ("Making Other Kinds Of Table Schema Changes"): with foreign keys off, the
-- references from refresh_tokens and password_reset_tokens keep pointing to
-- "users" and refer to the new table after the rename. The connection pool
-- has a single connection (db.Open), so the PRAGMAs apply to these
-- statements.

-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
BEGIN;
CREATE TABLE users_new (
    id          TEXT NOT NULL PRIMARY KEY,
    email       TEXT NOT NULL,
    password    TEXT NOT NULL,
    app_role    TEXT NOT NULL DEFAULT 'READONLY' CHECK (app_role IN ('ADMIN', 'SECRETARY', 'TREASURER', 'READONLY', 'TRAINER')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at  TEXT
);
INSERT INTO users_new (id, email, password, app_role, created_at, updated_at, deleted_at)
    SELECT id, email, password, app_role, created_at, updated_at, deleted_at FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE UNIQUE INDEX users_email_live ON users (email) WHERE deleted_at IS NULL;
COMMIT;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
BEGIN;
CREATE TABLE users_new (
    id          TEXT NOT NULL PRIMARY KEY,
    email       TEXT NOT NULL,
    password    TEXT NOT NULL,
    app_role    TEXT NOT NULL DEFAULT 'READONLY' CHECK (app_role IN ('ADMIN', 'SECRETARY', 'TREASURER', 'READONLY')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at  TEXT
);
INSERT INTO users_new (id, email, password, app_role, created_at, updated_at, deleted_at)
    SELECT id, email, password, CASE app_role WHEN 'TRAINER' THEN 'READONLY' ELSE app_role END,
           created_at, updated_at, deleted_at FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE UNIQUE INDEX users_email_live ON users (email) WHERE deleted_at IS NULL;
COMMIT;
PRAGMA foreign_keys = ON;
