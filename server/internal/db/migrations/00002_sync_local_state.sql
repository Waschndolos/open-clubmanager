-- Prepares the local database for sync (docs/sync-protocol.md §20, §22).
--
--   * Uniqueness on synced tables is no longer enforced by SQLite: two valid
--     changes from different devices may collide, and applying them must not
--     fail. The store checks uniqueness before local writes; the sync layer
--     flags collisions in sync_conflicts (§14.2). Plain indexes keep lookups fast.
--   * Local sync state: own device and HLC, per-field clocks, the outbox of
--     local changes not yet written to a segment, and detected conflicts.
--     These tables are never synced.

-- +goose Up

DROP INDEX members_number_live;
CREATE INDEX members_number_live ON members (number) WHERE deleted_at IS NULL;

DROP INDEX member_contacts_email_live;
CREATE INDEX member_contacts_email_live ON member_contacts (email)
    WHERE deleted_at IS NULL AND email IS NOT NULL;

DROP INDEX roles_name_live;
CREATE INDEX roles_name_live ON roles (name) WHERE deleted_at IS NULL;

DROP INDEX groups_name_live;
CREATE INDEX groups_name_live ON groups (name) WHERE deleted_at IS NULL;

DROP INDEX sections_name_live;
CREATE INDEX sections_name_live ON sections (name) WHERE deleted_at IS NULL;

-- This device. Exactly one row. device_id and the key references stay NULL
-- until the device identity is created (§5); field clocks written before that
-- carry device_id NULL, meaning "this device".
CREATE TABLE sync_device (
    id            INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
    device_id     TEXT,
    sign_key_ref  TEXT,
    enc_key_ref   TEXT,
    -- Last HLC handed out or observed (§12), "<ms hex 12>-<counter hex 4>".
    hlc           TEXT NOT NULL DEFAULT '000000000000-0000'
);
INSERT INTO sync_device (id) VALUES (1);

-- Last writer per field (§13.3, §14.1). pk is the JSON object of the row's
-- primary key columns with sorted keys, as in the op's "pk".
CREATE TABLE sync_field_clock (
    tbl        TEXT NOT NULL,
    pk         TEXT NOT NULL,
    col        TEXT NOT NULL,
    hlc        TEXT NOT NULL,
    device_id  TEXT,
    PRIMARY KEY (tbl, pk, col)
) WITHOUT ROWID;

-- Local ops (§10) in the order they were made, not yet written to a segment.
CREATE TABLE sync_outbox (
    seq         INTEGER PRIMARY KEY AUTOINCREMENT,
    scope       TEXT NOT NULL,
    hlc         TEXT NOT NULL,
    op          TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX sync_outbox_scope ON sync_outbox (scope, seq);

-- Rows that hold a unique value another live row holds with an older field
-- clock (§14.2). value is the JSON value, kept_pk the row that keeps it.
CREATE TABLE sync_conflicts (
    tbl          TEXT NOT NULL,
    pk           TEXT NOT NULL,
    col          TEXT NOT NULL,
    value        TEXT NOT NULL,
    kept_pk      TEXT NOT NULL,
    detected_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (tbl, pk, col)
) WITHOUT ROWID;

-- +goose Down

DROP TABLE sync_conflicts;
DROP TABLE sync_outbox;
DROP TABLE sync_field_clock;
DROP TABLE sync_device;

DROP INDEX sections_name_live;
CREATE UNIQUE INDEX sections_name_live ON sections (name) WHERE deleted_at IS NULL;
DROP INDEX groups_name_live;
CREATE UNIQUE INDEX groups_name_live ON groups (name) WHERE deleted_at IS NULL;
DROP INDEX roles_name_live;
CREATE UNIQUE INDEX roles_name_live ON roles (name) WHERE deleted_at IS NULL;
DROP INDEX member_contacts_email_live;
CREATE UNIQUE INDEX member_contacts_email_live ON member_contacts (email)
    WHERE deleted_at IS NULL AND email IS NOT NULL;
DROP INDEX members_number_live;
CREATE UNIQUE INDEX members_number_live ON members (number) WHERE deleted_at IS NULL;
