-- Households (families) with a main payer, and guardians of members.

-- +goose Up

-- payer_member_id is the main payer (Hauptzahler) of the household. With
-- share_address or share_bank set, a change of the address or bank details of
-- one member is copied to all members of the household.
CREATE TABLE households (
    id               TEXT NOT NULL PRIMARY KEY,
    name             TEXT NOT NULL,
    payer_member_id  TEXT REFERENCES members (id),
    share_address    INTEGER NOT NULL DEFAULT 0,
    share_bank       INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT
);

-- Keyed by member, so a member belongs to at most one household.
CREATE TABLE household_members (
    member_id     TEXT NOT NULL PRIMARY KEY REFERENCES members (id),
    household_id  TEXT NOT NULL REFERENCES households (id),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);
CREATE INDEX household_members_household ON household_members (household_id);

-- A guardian (Erziehungsberechtigte/r) is either another member
-- (guardian_member_id) or a person given by name and contact details.
CREATE TABLE member_guardians (
    id                  TEXT NOT NULL PRIMARY KEY,
    member_id           TEXT NOT NULL REFERENCES members (id),
    guardian_member_id  TEXT REFERENCES members (id),
    name                TEXT,
    relation            TEXT,
    email               TEXT,
    phone               TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at          TEXT
);
CREATE INDEX member_guardians_member ON member_guardians (member_id);

-- +goose Down

DROP TABLE member_guardians;
DROP TABLE household_members;
DROP TABLE households;
