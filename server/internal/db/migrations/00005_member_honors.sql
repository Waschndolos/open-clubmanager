-- Honors (Ehrungen) of members and a manual correction of the membership
-- years used for anniversaries (e.g. after a break in the membership).

-- +goose Up

-- Manual correction of the membership years (may be negative), one row per
-- member that has one.
CREATE TABLE member_membership (
    member_id     TEXT NOT NULL PRIMARY KEY REFERENCES members (id),
    years_adjust  INTEGER NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);

-- kind is 'membership' (anniversary), 'birthday' or 'other'. milestone is the
-- honored number of membership years or the age; NULL for 'other'. Rows with
-- a milestone mark that stage as done, so it is not suggested again.
CREATE TABLE member_honors (
    id          TEXT NOT NULL PRIMARY KEY,
    member_id   TEXT NOT NULL REFERENCES members (id),
    kind        TEXT NOT NULL,
    milestone   INTEGER,
    title       TEXT NOT NULL,
    honor_date  TEXT NOT NULL,
    note        TEXT,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at  TEXT
);
CREATE INDEX member_honors_member ON member_honors (member_id);

-- +goose Down

DROP TABLE member_honors;
DROP TABLE member_membership;
