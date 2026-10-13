-- Condition, inspection/maintenance schedule and maintenance history of
-- inventory items.

-- +goose Up

-- condition is ready, damaged, in_repair or retired; only ready items may be
-- lent. interval_months is the inspection/maintenance interval; recording a
-- maintenance moves next_due_date forward by it. Items without a row are
-- ready and have no schedule.
CREATE TABLE inventory_item_status (
    item_id          TEXT NOT NULL PRIMARY KEY REFERENCES inventory_items (id),
    condition        TEXT NOT NULL DEFAULT 'ready',
    interval_months  INTEGER,
    next_due_date    TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT
);

-- transaction_id links the expense booked for the cost, if any.
CREATE TABLE inventory_maintenance (
    id              TEXT NOT NULL PRIMARY KEY,
    item_id         TEXT NOT NULL REFERENCES inventory_items (id),
    performed_on    TEXT NOT NULL,
    performed_by    TEXT,
    note            TEXT,
    cost_cents      INTEGER,
    transaction_id  TEXT REFERENCES finance_transactions (id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at      TEXT
);
CREATE INDEX inventory_maintenance_item ON inventory_maintenance (item_id);

-- +goose Down

DROP TABLE inventory_maintenance;
DROP TABLE inventory_item_status;
