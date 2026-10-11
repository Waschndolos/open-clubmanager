-- Local state of the sync identity layer (docs/sync-protocol.md §5, §7–§9,
-- §17, §20). None of these tables is ever synced.

-- +goose Up

-- Public halves of the device keys, so the device can describe itself
-- (join request, ACL entry) without unlocking the private keys.
ALTER TABLE sync_device ADD COLUMN sign_pub TEXT;
ALTER TABLE sync_device ADD COLUMN age_recipient TEXT;

-- The club this device belongs to. Exactly one row once the device created
-- or joined a club. club_id and root_sign_pub are pinned (§7): a later
-- club.json with other values is refused. acl_version/acl_hash are the head
-- of the highest accepted ACL chain (§8.4); 0/NULL while a join request is
-- pending.
CREATE TABLE sync_club (
    id                  INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
    club_id             TEXT NOT NULL,
    name                TEXT NOT NULL,
    root_sign_pub       TEXT NOT NULL,
    root_age_recipient  TEXT NOT NULL,
    genesis             BLOB NOT NULL,
    genesis_sig         BLOB NOT NULL,
    acl_version         INTEGER NOT NULL DEFAULT 0,
    acl_hash            TEXT,
    -- joining: request written, not yet in the ACL; member: active in the
    -- ACL; revoked: revoked in the ACL.
    status              TEXT NOT NULL CHECK (status IN ('joining', 'member', 'revoked')),
    -- Storage location chosen by the user (JSON), e.g. {"kind":"folder"}.
    transport           TEXT NOT NULL DEFAULT '{}',
    join_invite_id      TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- Accepted ACL chain (§8). Kept so that deleting ACL files from the shared
-- folder can neither roll the chain back nor make it unverifiable.
CREATE TABLE sync_acl (
    version  INTEGER NOT NULL PRIMARY KEY,
    hash     TEXT NOT NULL,
    body     BLOB NOT NULL,
    sig      BLOB NOT NULL
);

-- Scope keys wrapped for this device (§9), as fetched from
-- keys/<scope>/<epoch>/<device_id>.age. They stay age-encrypted at rest.
CREATE TABLE sync_keys (
    scope    TEXT NOT NULL,
    epoch    INTEGER NOT NULL,
    wrapped  BLOB NOT NULL,
    PRIMARY KEY (scope, epoch)
) WITHOUT ROWID;

-- Keys of new epochs this (admin) device generated for an ACL version it has
-- not yet seen accepted, wrapped for this device. A retry after a failed
-- publish reuses them, so key files already written stay consistent with the
-- key check in the ACL. Removed once the epoch is decided.
CREATE TABLE sync_key_drafts (
    scope    TEXT NOT NULL,
    epoch    INTEGER NOT NULL,
    wrapped  BLOB NOT NULL,
    PRIMARY KEY (scope, epoch)
) WITHOUT ROWID;

-- Invites created by this (admin) device (§17.2 step 1). Only stored here,
-- never in the shared folder.
CREATE TABLE sync_invites (
    invite_id   TEXT NOT NULL PRIMARY KEY,
    secret      BLOB NOT NULL,
    role        TEXT NOT NULL,
    person      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    used_at     TEXT,
    used_by     TEXT
);

-- Position per stream (§13, §20). Written by the sync core; the identity
-- layer reads it to compute revocation cutoffs (§17.4).
CREATE TABLE sync_streams (
    device_id  TEXT NOT NULL,
    scope      TEXT NOT NULL,
    seq        INTEGER NOT NULL DEFAULT 0,
    hash       TEXT,
    status     TEXT NOT NULL DEFAULT 'ok' CHECK (status IN ('ok', 'waiting', 'quarantined')),
    PRIMARY KEY (device_id, scope)
) WITHOUT ROWID;

-- +goose Down

DROP TABLE sync_streams;
DROP TABLE sync_invites;
DROP TABLE sync_key_drafts;
DROP TABLE sync_keys;
DROP TABLE sync_acl;
DROP TABLE sync_club;
ALTER TABLE sync_device DROP COLUMN age_recipient;
ALTER TABLE sync_device DROP COLUMN sign_pub;
