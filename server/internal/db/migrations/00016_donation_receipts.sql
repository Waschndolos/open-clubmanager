-- Donation receipts (Zuwendungsbestätigungen) after the official German
-- template: the club's tax exemption data, external donors, the assignment of
-- donations in the cash book to a member or donor, and the receipts issued
-- with their items. All tables are in scope finance.

-- +goose Up

-- The club as issuer of receipts. A single row with the fixed id 'club'.
-- exemption_kind is 'notice' (Freistellungsbescheid / Anlage zum
-- Körperschaftsteuerbescheid, valid 5 years) or 'assessment' (Feststellung
-- nach § 60a AO, valid 3 years). assessment_period is the last period
-- (Veranlagungszeitraum) of a notice. fees_deductible is set when membership
-- fees may be certified (not for sports clubs, § 10b Abs. 1 Satz 8 EStG).
CREATE TABLE donation_settings (
    id                 TEXT NOT NULL PRIMARY KEY,
    issuer_name        TEXT NOT NULL,
    issuer_address     TEXT NOT NULL,
    tax_office         TEXT NOT NULL,
    tax_number         TEXT NOT NULL,
    exemption_kind     TEXT NOT NULL,
    exemption_date     TEXT NOT NULL,
    assessment_period  TEXT,
    purposes           TEXT NOT NULL,
    fees_deductible    INTEGER NOT NULL DEFAULT 0,
    signatory          TEXT,
    place              TEXT,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at         TEXT
);

-- A donor who is not a member.
CREATE TABLE donors (
    id           TEXT NOT NULL PRIMARY KEY,
    name         TEXT NOT NULL,
    street       TEXT,
    postal_code  TEXT,
    city         TEXT,
    notes        TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at   TEXT
);

-- Marks an income transaction as a donation by a member or an external donor
-- (exactly one of member_id and donor_id). expense_waiver is set for a waiver
-- of reimbursement of expenses (Aufwandsspende).
CREATE TABLE finance_transaction_donations (
    transaction_id  TEXT NOT NULL PRIMARY KEY REFERENCES finance_transactions (id),
    member_id       TEXT REFERENCES members (id),
    donor_id        TEXT REFERENCES donors (id),
    expense_waiver  INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at      TEXT
);
CREATE INDEX finance_transaction_donations_member ON finance_transaction_donations (member_id);
CREATE INDEX finance_transaction_donations_donor ON finance_transaction_donations (donor_id);

-- An issued receipt; the table is the log of what was certified when for
-- whom. Recipient and issuer data are copied, so the PDF can be produced
-- again unchanged. kind is 'single' or 'collective'. A duplicate
-- (duplicate_of set) repeats an earlier receipt with its number and is
-- marked as such. A voided receipt stays in the log, its items can be
-- certified again.
CREATE TABLE donation_receipts (
    id                   TEXT NOT NULL PRIMARY KEY,
    number               TEXT NOT NULL,
    year                 INTEGER NOT NULL,
    sequence             INTEGER NOT NULL,
    kind                 TEXT NOT NULL,
    member_id            TEXT REFERENCES members (id),
    donor_id             TEXT REFERENCES donors (id),
    recipient_name       TEXT NOT NULL,
    recipient_address    TEXT NOT NULL,
    period_start         TEXT NOT NULL,
    period_end           TEXT NOT NULL,
    total_cents          INTEGER NOT NULL,
    issue_date           TEXT NOT NULL,
    issuer               TEXT NOT NULL,
    duplicate_of         TEXT REFERENCES donation_receipts (id),
    created_by           TEXT,
    voided_at            TEXT,
    void_reason          TEXT,
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at           TEXT
);
CREATE INDEX donation_receipts_year ON donation_receipts (year);

-- One donation or membership fee certified by a receipt. kind is 'donation'
-- (transaction_id set) or 'membership_fee' (fee_id set).
CREATE TABLE donation_receipt_items (
    id              TEXT NOT NULL PRIMARY KEY,
    receipt_id      TEXT NOT NULL REFERENCES donation_receipts (id),
    item_date       TEXT NOT NULL,
    kind            TEXT NOT NULL,
    amount_cents    INTEGER NOT NULL,
    expense_waiver  INTEGER NOT NULL DEFAULT 0,
    transaction_id  TEXT REFERENCES finance_transactions (id),
    fee_id          TEXT REFERENCES member_fees (id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at      TEXT
);
CREATE INDEX donation_receipt_items_receipt ON donation_receipt_items (receipt_id);
CREATE INDEX donation_receipt_items_transaction ON donation_receipt_items (transaction_id);
CREATE INDEX donation_receipt_items_fee ON donation_receipt_items (fee_id);

-- +goose Down

DROP TABLE donation_receipt_items;
DROP TABLE donation_receipts;
DROP TABLE finance_transaction_donations;
DROP TABLE donors;
DROP TABLE donation_settings;
