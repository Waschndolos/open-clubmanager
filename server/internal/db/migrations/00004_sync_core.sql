-- Local state of the sync core (docs/sync-protocol.md §11, §13–§15, §20).
-- None of these tables is ever synced.

-- +goose Up

-- Conflict-ordering generation of the write (§12, §14.1): changes are
-- ordered by (clock_epoch, hlc, device_id). Existing clocks are from the
-- first generation.
ALTER TABLE sync_field_clock ADD COLUMN clock_epoch INTEGER NOT NULL DEFAULT 1;
ALTER TABLE sync_outbox ADD COLUMN clock_epoch INTEGER NOT NULL DEFAULT 1;

-- Current clock epoch per scope from the newest accepted ACL, maintained by
-- the sync core. Local writes stamp their field clocks and outbox entries
-- with it; a scope without a row is in generation 1.
CREATE TABLE sync_clock_epochs (
    scope        TEXT NOT NULL PRIMARY KEY,
    clock_epoch  INTEGER NOT NULL
) WITHOUT ROWID;

-- Stream state (§13.1, §13.2). seq/hash is the last accepted segment; for
-- this device's own streams it is the last segment written, so the next seq
-- is seq + 1 (§11.3, §20 "next seq per scope"). failures counts consecutive
-- rejections of the next segment: a file still being synced by a cloud
-- client is retried before the stream is quarantined. detail explains the
-- status for the UI.
ALTER TABLE sync_streams ADD COLUMN failures INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_streams ADD COLUMN detail TEXT NOT NULL DEFAULT '';

-- Scopes whose starting state is present on this device (§15.1, §15.4): the
-- creating device's initial snapshot, or the snapshot a joining device
-- loaded. Segments of a scope are only applied once it has a base.
CREATE TABLE sync_scopes (
    scope      TEXT NOT NULL PRIMARY KEY,
    -- Path and complete-file hash of the snapshot the state started from.
    snapshot   TEXT NOT NULL,
    hash       TEXT NOT NULL,
    loaded_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) WITHOUT ROWID;

-- Files this device created and must still upload (§11.3): the next seq is
-- persisted together with the exact file before uploading, and a failed
-- upload is retried with the identical bytes.
CREATE TABLE sync_uploads (
    path        TEXT NOT NULL PRIMARY KEY,
    data        BLOB NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) WITHOUT ROWID;

-- Received ops waiting for a row they reference, or for their own row when
-- they do not carry enough columns to insert it (§13.3 step 5). Retried
-- after every apply round.
CREATE TABLE sync_pending (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    scope        TEXT NOT NULL,
    device_id    TEXT NOT NULL,
    clock_epoch  INTEGER NOT NULL,
    hlc          TEXT NOT NULL,
    op           TEXT NOT NULL,
    reason       TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX sync_pending_scope ON sync_pending (scope);

-- Ops of accepted segments (received and own) with their author, for the
-- change history and audit view (§20, §22).
CREATE TABLE sync_history (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    scope        TEXT NOT NULL,
    device_id    TEXT NOT NULL,
    seq          INTEGER NOT NULL,
    tbl          TEXT NOT NULL,
    pk           TEXT NOT NULL,
    clock_epoch  INTEGER NOT NULL,
    hlc          TEXT NOT NULL,
    op           TEXT NOT NULL,
    applied_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX sync_history_row ON sync_history (tbl, pk);
CREATE UNIQUE INDEX sync_history_op ON sync_history (device_id, scope, seq, tbl, pk, hlc);

-- Devices whose ops carried an HLC more than MAX_DRIFT ahead of this
-- device's clock (§12): shown as a clock warning naming the author.
CREATE TABLE sync_clock_warnings (
    device_id  TEXT NOT NULL PRIMARY KEY,
    hlc        TEXT NOT NULL,
    seen_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) WITHOUT ROWID;

-- +goose Down

DROP TABLE sync_clock_warnings;
DROP TABLE sync_clock_epochs;
DROP TABLE sync_history;
DROP TABLE sync_pending;
DROP TABLE sync_uploads;
DROP TABLE sync_scopes;
ALTER TABLE sync_streams DROP COLUMN detail;
ALTER TABLE sync_streams DROP COLUMN failures;
ALTER TABLE sync_outbox DROP COLUMN clock_epoch;
ALTER TABLE sync_field_clock DROP COLUMN clock_epoch;
