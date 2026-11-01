-- SEPA Core direct debit of member fees: the club's creditor data, mandate
-- references and the collection runs exported as pain.008 files. All tables
-- are in scope members.bank, since they hold IBANs and mandates.

-- +goose Up

-- The club as creditor. A single row with the fixed id 'club', so devices
-- that set it up independently edit the same row instead of creating rivals.
CREATE TABLE sepa_creditor (
    id                   TEXT NOT NULL PRIMARY KEY,
    name                 TEXT NOT NULL,
    creditor_id          TEXT NOT NULL,
    iban                 TEXT NOT NULL,
    bic                  TEXT,
    remittance_template  TEXT,
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at           TEXT
);

-- Mandate reference (MndtId) of a member's SEPA mandate; the date of
-- signature stays in member_bank_details.sepa_mandate_date. References are
-- unique among live rows (db.UniqueColumns, checked by the store).
CREATE TABLE member_sepa_mandates (
    member_id          TEXT NOT NULL PRIMARY KEY REFERENCES members (id),
    mandate_reference  TEXT NOT NULL,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at         TEXT
);
CREATE INDEX member_sepa_mandates_reference ON member_sepa_mandates (mandate_reference) WHERE deleted_at IS NULL;

-- A collection run. The creditor data is copied, so the file can be
-- downloaded again unchanged after the settings changed. paid_date is set
-- once the collection has been confirmed (the fees are then paid);
-- cancelling a run tombstones it and its items, and the fees are open again.
CREATE TABLE sepa_runs (
    id                TEXT NOT NULL PRIMARY KEY,
    collection_date   TEXT NOT NULL,
    creditor_name     TEXT NOT NULL,
    creditor_id       TEXT NOT NULL,
    creditor_iban     TEXT NOT NULL,
    creditor_bic      TEXT,
    paid_date         TEXT,
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at        TEXT
);

-- One debit per fee, with the debtor data as exported. sequence_type is
-- FRST or RCUR.
CREATE TABLE sepa_run_items (
    id                 TEXT NOT NULL PRIMARY KEY,
    run_id             TEXT NOT NULL REFERENCES sepa_runs (id),
    fee_id             TEXT NOT NULL REFERENCES member_fees (id),
    member_id          TEXT NOT NULL REFERENCES members (id),
    amount_cents       INTEGER NOT NULL,
    sequence_type      TEXT NOT NULL,
    mandate_reference  TEXT NOT NULL,
    mandate_date       TEXT NOT NULL,
    debtor_name        TEXT NOT NULL,
    iban               TEXT NOT NULL,
    bic                TEXT,
    remittance         TEXT NOT NULL,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at         TEXT
);
CREATE INDEX sepa_run_items_run ON sepa_run_items (run_id);
CREATE INDEX sepa_run_items_fee ON sepa_run_items (fee_id);
CREATE INDEX sepa_run_items_member ON sepa_run_items (member_id);

-- +goose Down

DROP TABLE sepa_run_items;
DROP TABLE sepa_runs;
DROP TABLE member_sepa_mandates;
DROP TABLE sepa_creditor;
