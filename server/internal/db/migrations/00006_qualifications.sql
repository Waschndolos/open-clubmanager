-- Qualifications and licences of members with expiry tracking.

-- +goose Up

-- validity_months is the default validity used to compute the expiry date;
-- NULL means the qualification does not expire. inspection_only types (the
-- extended police clearance certificate, § 72a SGB VIII) store only the date
-- the certificate was inspected, never its content or a note.
CREATE TABLE qualification_types (
    id               TEXT NOT NULL PRIMARY KEY,
    name             TEXT NOT NULL,
    validity_months  INTEGER,
    inspection_only  INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT
);
CREATE INDEX qualification_types_name ON qualification_types (name);

-- issued_date is the issue date, or the inspection date for inspection_only types.
CREATE TABLE member_qualifications (
    id           TEXT NOT NULL PRIMARY KEY,
    member_id    TEXT NOT NULL REFERENCES members (id),
    type_id      TEXT NOT NULL REFERENCES qualification_types (id),
    issued_date  TEXT NOT NULL,
    expiry_date  TEXT,
    note         TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at   TEXT
);
CREATE INDEX member_qualifications_member ON member_qualifications (member_id);

-- +goose Down

DROP TABLE member_qualifications;
DROP TABLE qualification_types;
