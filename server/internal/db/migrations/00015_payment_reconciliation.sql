-- Payment reconciliation: bank statement data of imported transactions and
-- the links between transactions and the member fees they pay (or, for a
-- returned direct debit, reverse).

-- +goose Up

-- Bank data of a transaction imported from a CAMT.053 statement (scope
-- members.bank, since it holds the payer's IBAN). is_return marks a returned
-- direct debit (Rücklastschrift). dismissed_at is set when the treasurer
-- reviewed the entry and found no fee to assign.
CREATE TABLE finance_transaction_bank (
    transaction_id     TEXT NOT NULL PRIMARY KEY REFERENCES finance_transactions (id),
    counterparty_name  TEXT,
    counterparty_iban  TEXT,
    end_to_end_id      TEXT,
    mandate_reference  TEXT,
    is_return          INTEGER NOT NULL DEFAULT 0,
    return_reason      TEXT,
    dismissed_at       TEXT,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at         TEXT
);

-- A confirmed assignment of (part of) a transaction to a fee (scope finance).
-- amount_cents is positive for a payment and negative for a returned direct
-- debit. A transaction may pay several fees (e.g. a family's transfer), a fee
-- may be paid by several transactions (partial payments). prior_paid_date is
-- the fee's paid date before the link was made, so that undoing the link
-- restores it.
CREATE TABLE member_fee_payments (
    id               TEXT NOT NULL PRIMARY KEY,
    fee_id           TEXT NOT NULL REFERENCES member_fees (id),
    transaction_id   TEXT NOT NULL REFERENCES finance_transactions (id),
    amount_cents     INTEGER NOT NULL,
    prior_paid_date  TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deleted_at       TEXT
);
CREATE INDEX member_fee_payments_fee ON member_fee_payments (fee_id);
CREATE INDEX member_fee_payments_transaction ON member_fee_payments (transaction_id);

-- +goose Down

DROP TABLE member_fee_payments;
DROP TABLE finance_transaction_bank;
