-- Tasks for the board and roles.

-- +goose Up

-- status is open, in_progress or done; recurrence is none, weekly, monthly or
-- yearly (completing a recurring task creates the next one). A task is
-- assigned to a member or to a role, or to nobody.
CREATE TABLE tasks (
    id                  TEXT NOT NULL PRIMARY KEY,
    title               TEXT NOT NULL,
    description         TEXT,
    due_date            TEXT,
    status              TEXT NOT NULL DEFAULT 'open',
    recurrence          TEXT NOT NULL DEFAULT 'none',
    assignee_member_id  TEXT REFERENCES members (id),
    assignee_role_id    TEXT REFERENCES roles (id),
    linked_member_id    TEXT REFERENCES members (id),
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at          TEXT
);
CREATE INDEX tasks_due ON tasks (due_date);

-- +goose Down

DROP TABLE tasks;
