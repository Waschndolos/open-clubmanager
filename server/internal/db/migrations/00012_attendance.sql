-- Attendance of members at event occurrences (training sessions etc.).

-- +goose Up

-- One row per member and occurrence (event, local start date). status is
-- present, excused or unexcused; removing an entry tombstones the row.
CREATE TABLE event_attendance (
    event_id         TEXT NOT NULL REFERENCES events (id),
    occurrence_date  TEXT NOT NULL,
    member_id        TEXT NOT NULL REFERENCES members (id),
    status           TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT,
    PRIMARY KEY (event_id, occurrence_date, member_id)
);
CREATE INDEX event_attendance_member ON event_attendance (member_id);

-- +goose Down

DROP TABLE event_attendance;
