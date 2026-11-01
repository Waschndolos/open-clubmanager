-- Fee models (Beitragsmodelle), the model assigned to a member and the
-- billing runs (Sollstellung) that create member fees from them. All tables
-- are in scope finance, like member_fees.

-- +goose Up

-- A fee class such as adults, youth, family or passive. amount_cents is the
-- amount per period of the interval ('yearly', 'half_yearly', 'quarterly',
-- 'monthly'). Models with an age limit (inclusive, age on 1 January of the
-- fee year) are picked automatically for members without an assignment.
-- pro_rata charges only the months of a period in which the member belongs
-- to the club. Names are unique among live rows (db.UniqueColumns).
CREATE TABLE fee_models (
    id            TEXT NOT NULL PRIMARY KEY,
    name          TEXT NOT NULL,
    amount_cents  INTEGER NOT NULL,
    interval      TEXT NOT NULL,
    min_age       INTEGER,
    max_age       INTEGER,
    pro_rata      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);
CREATE INDEX fee_models_name ON fee_models (name);

-- The fee model of a member. fee_model_id NULL means the member pays no fee
-- (e.g. honorary members); members without a row get a model by their age.
CREATE TABLE member_fee_models (
    member_id     TEXT NOT NULL PRIMARY KEY REFERENCES members (id),
    fee_model_id  TEXT REFERENCES fee_models (id),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);

-- A billing run for the periods that start between period_from and
-- period_to (first and last day of a month).
CREATE TABLE fee_runs (
    id            TEXT NOT NULL PRIMARY KEY,
    period_from   TEXT NOT NULL,
    period_to     TEXT NOT NULL,
    due_date      TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);

-- The period a fee of a run was charged for (1:1 with member_fees). A live
-- fee with a charge blocks charging an overlapping period to the member
-- again; deleting the fee frees the period.
CREATE TABLE fee_charges (
    fee_id        TEXT NOT NULL PRIMARY KEY REFERENCES member_fees (id),
    run_id        TEXT NOT NULL REFERENCES fee_runs (id),
    member_id     TEXT NOT NULL REFERENCES members (id),
    fee_model_id  TEXT NOT NULL REFERENCES fee_models (id),
    period_start  TEXT NOT NULL,
    period_end    TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at    TEXT
);
CREATE INDEX fee_charges_run ON fee_charges (run_id);
CREATE INDEX fee_charges_member ON fee_charges (member_id, period_start);

-- +goose Down

DROP TABLE fee_charges;
DROP TABLE fee_runs;
DROP TABLE member_fee_models;
DROP TABLE fee_models;
