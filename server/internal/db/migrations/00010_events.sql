-- Club events (Termine & Veranstaltungen) with recurrence and audiences.

-- +goose Up

-- starts_at/ends_at are the first occurrence. Recurring events repeat at the
-- same local wall-clock time in time_zone (IANA name), so daylight saving
-- changes do not shift them. recurrence is none, weekly, biweekly or monthly;
-- recurrence_until is the last local date an occurrence may start on.
CREATE TABLE events (
    id                TEXT NOT NULL PRIMARY KEY,
    title             TEXT NOT NULL,
    description       TEXT,
    location          TEXT,
    category          TEXT NOT NULL DEFAULT 'other',
    starts_at         TEXT NOT NULL,
    ends_at           TEXT NOT NULL,
    all_day           INTEGER NOT NULL DEFAULT 0,
    time_zone         TEXT NOT NULL DEFAULT 'UTC',
    recurrence        TEXT NOT NULL DEFAULT 'none',
    recurrence_until  TEXT,
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at        TEXT
);
CREATE INDEX events_starts_at ON events (starts_at);

-- Target audience: kind is group, section or role. An event without
-- audience rows is for the whole club.
CREATE TABLE event_audiences (
    event_id    TEXT NOT NULL REFERENCES events (id),
    kind        TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at  TEXT,
    PRIMARY KEY (event_id, kind, target_id)
);

-- Excluded occurrences of a recurring event, by local start date.
CREATE TABLE event_exceptions (
    event_id         TEXT NOT NULL REFERENCES events (id),
    occurrence_date  TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT,
    PRIMARY KEY (event_id, occurrence_date)
);

-- +goose Down

DROP TABLE event_exceptions;
DROP TABLE event_audiences;
DROP TABLE events;
