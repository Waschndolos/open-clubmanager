package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/reconcile"
)

var (
	// ErrNotReconcilable is returned for a transaction that was not imported
	// from a bank statement, or an expense that is not a returned debit.
	ErrNotReconcilable = errors.New("transaction cannot be reconciled")
	// ErrAlreadyReconciled is returned when a transaction already has fees
	// assigned.
	ErrAlreadyReconciled = errors.New("transaction already reconciled")
	// ErrNoAllocations is returned when a confirmation assigns no fee.
	ErrNoAllocations = errors.New("no fees assigned")
	// ErrAllocationExceeds is returned when the assigned amounts exceed the
	// amount of the transaction.
	ErrAllocationExceeds = errors.New("assigned amounts exceed the transaction")
)

// AllocationError lists the fees of a confirmation that cannot take the
// assigned amount: not open (for a return: not paid) any more, in a direct
// debit run that has not been confirmed, or with less due than assigned.
type AllocationError struct {
	FeeIDs []string
}

func (e *AllocationError) Error() string {
	return fmt.Sprintf("%d fees cannot take the assigned amount", len(e.FeeIDs))
}

// BankEntry is the bank statement data of an imported transaction.
type BankEntry struct {
	CounterpartyName *string
	CounterpartyIBAN *string
	EndToEndID       *string
	MandateRef       *string
	IsReturn         bool
	ReturnReason     *string
}

// ReconciliationEntry is an imported transaction with the fees assigned to it.
type ReconciliationEntry struct {
	TransactionID string
	Date          time.Time
	Description   string
	AmountCents   int64
	Type          string
	Bank          BankEntry
	DismissedAt   *time.Time
	Payments      []FeePayment
}

// Open reports whether the entry still waits for review: an income or a
// returned debit without assigned fees that has not been dismissed.
func (e ReconciliationEntry) Open() bool {
	return len(e.Payments) == 0 && e.DismissedAt == nil && (e.Type == "income" || e.Bank.IsReturn)
}

// FeePayment is the confirmed assignment of (part of) a transaction to a fee.
// AmountCents is negative for a returned debit.
type FeePayment struct {
	ID            string
	FeeID         string
	TransactionID string
	Member        MemberRef
	FeeYear       int
	FeeDesc       *string
	AmountCents   int64
}

// ReconcileFee is a fee with what has been paid on it.
type ReconcileFee struct {
	ID          string
	Member      MemberRef
	Year        int
	Description *string
	DueDate     time.Time
	AmountCents int64
	// PaidCents is what the fee counts as paid: the full amount once it has
	// a paid date, otherwise the payments assigned since its last returned
	// debit (partial payments).
	PaidCents int64
	PaidDate  *time.Time
	// InCollection is set while the fee is in a direct debit run that has
	// not been confirmed.
	InCollection bool
}

// Open reports whether the fee can take a payment.
func (f ReconcileFee) Open() bool {
	return f.PaidDate == nil && !f.InCollection && f.AmountCents-f.PaidCents > 0
}

// Allocation assigns part of a transaction to a fee in a confirmation; the
// amount is positive, also for returns.
type Allocation struct {
	FeeID       string
	AmountCents int64
}

// ImportTransaction inserts a transaction with its bank statement data (nil
// to import the transaction alone, e.g. on a device without access to bank
// data).
func (s *Store) ImportTransaction(ctx context.Context, in TransactionInput, bank *BankEntry) (Transaction, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		now := db.Now()
		values := transactionColumns(in)
		values["id"], values["created_at"] = id, now
		if err := w.insert("finance_transactions", values); err != nil {
			return err
		}
		if bank == nil {
			return nil
		}
		isReturn := 0
		if bank.IsReturn {
			isReturn = 1
		}
		return w.insert("finance_transaction_bank", row{
			"transaction_id": id, "counterparty_name": strArg(bank.CounterpartyName),
			"counterparty_iban": strArg(bank.CounterpartyIBAN), "end_to_end_id": strArg(bank.EndToEndID),
			"mandate_reference": strArg(bank.MandateRef), "is_return": isReturn,
			"return_reason": strArg(bank.ReturnReason), "created_at": now,
		})
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.GetTransaction(ctx, id)
}

// ReconciliationEntries returns all live imported transactions, newest first,
// with their assigned fees.
func (s *Store) ReconciliationEntries(ctx context.Context) ([]ReconciliationEntry, error) {
	return reconciliationEntries(ctx, s.db, "")
}

func reconciliationEntries(ctx context.Context, q queryer, id string) ([]ReconciliationEntry, error) {
	query := `SELECT t.id, t.date, t.description, t.amount_cents, t.type,
		b.counterparty_name, b.counterparty_iban, b.end_to_end_id, b.mandate_reference, b.is_return, b.return_reason,
		b.dismissed_at
	FROM finance_transactions t JOIN finance_transaction_bank b ON b.transaction_id = t.id AND b.deleted_at IS NULL
	WHERE t.deleted_at IS NULL`
	var args []any
	if id != "" {
		query += ` AND t.id = ?`
		args = append(args, id)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY t.date DESC, t.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	result := []ReconciliationEntry{}
	index := map[string]int{}
	for rows.Next() {
		var e ReconciliationEntry
		var date string
		var name, iban, e2e, mandate, reason, dismissed sql.NullString
		if err := rows.Scan(&e.TransactionID, &date, &e.Description, &e.AmountCents, &e.Type,
			&name, &iban, &e2e, &mandate, &e.Bank.IsReturn, &reason, &dismissed); err != nil {
			rows.Close()
			return nil, err
		}
		e.Date, e.DismissedAt = parseTime(date), nullTime(dismissed)
		e.Bank.CounterpartyName, e.Bank.CounterpartyIBAN, e.Bank.EndToEndID = nullStr(name), nullStr(iban), nullStr(e2e)
		e.Bank.MandateRef, e.Bank.ReturnReason = nullStr(mandate), nullStr(reason)
		e.Payments = []FeePayment{}
		index[e.TransactionID] = len(result)
		result = append(result, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(result) == 0 {
		return result, err
	}

	payments, err := feePayments(ctx, q, "p.transaction_id", id)
	if err != nil {
		return nil, err
	}
	for _, p := range payments {
		if i, ok := index[p.TransactionID]; ok {
			result[i].Payments = append(result[i].Payments, p)
		}
	}
	return result, nil
}

// feePayments returns the live assignments of live fees and transactions,
// optionally only those whose column col equals id.
func feePayments(ctx context.Context, q queryer, col, id string) ([]FeePayment, error) {
	query := `SELECT p.id, p.fee_id, p.transaction_id, p.amount_cents, f.year, f.description,
		m.id, m.number, m.first_name, m.last_name
	FROM member_fee_payments p
	JOIN member_fees f ON f.id = p.fee_id AND f.deleted_at IS NULL
	JOIN members m ON m.id = f.member_id
	JOIN finance_transactions t ON t.id = p.transaction_id AND t.deleted_at IS NULL
	WHERE p.deleted_at IS NULL`
	var args []any
	if id != "" {
		query += ` AND ` + col + ` = ?`
		args = append(args, id)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY m.last_name, m.first_name, m.number, f.due_date, p.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []FeePayment{}
	for rows.Next() {
		var p FeePayment
		var desc sql.NullString
		if err := rows.Scan(&p.ID, &p.FeeID, &p.TransactionID, &p.AmountCents, &p.FeeYear, &desc,
			&p.Member.ID, &p.Member.Number, &p.Member.FirstName, &p.Member.LastName); err != nil {
			return nil, err
		}
		p.FeeDesc = nullStr(desc)
		result = append(result, p)
	}
	return result, rows.Err()
}

// MemberFeePayments returns the assignments to the fees of a member.
func (s *Store) MemberFeePayments(ctx context.Context, memberID string) ([]FeePayment, error) {
	return feePayments(ctx, s.db, "m.id", memberID)
}

// ReconcileFees returns the live fees of live members with what has been
// paid on them, ordered by member and due date.
func (s *Store) ReconcileFees(ctx context.Context) ([]ReconcileFee, error) {
	return reconcileFees(ctx, s.db, nil)
}

func reconcileFees(ctx context.Context, q queryer, ids []string) ([]ReconcileFee, error) {
	query := `SELECT f.id, f.year, f.description, f.due_date, f.amount_cents, f.paid_date,
		m.id, m.number, m.first_name, m.last_name,
		EXISTS (SELECT 1 FROM sepa_run_items i JOIN sepa_runs r ON r.id = i.run_id AND r.deleted_at IS NULL
			AND r.paid_date IS NULL WHERE i.deleted_at IS NULL AND i.fee_id = f.id),
		` + paidSinceReturnSQL + `
	FROM member_fees f JOIN members m ON m.id = f.member_id AND m.deleted_at IS NULL
	WHERE f.deleted_at IS NULL`
	var args []any
	if ids != nil {
		query += ` AND f.id IN (` + placeholders(len(ids)) + `)`
		args = stringArgs(ids)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY m.last_name, m.first_name, m.number, f.due_date, f.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReconcileFee{}
	for rows.Next() {
		var f ReconcileFee
		var desc, paid sql.NullString
		var due string
		if err := rows.Scan(&f.ID, &f.Year, &desc, &due, &f.AmountCents, &paid,
			&f.Member.ID, &f.Member.Number, &f.Member.FirstName, &f.Member.LastName,
			&f.InCollection, &f.PaidCents); err != nil {
			return nil, err
		}
		f.Description, f.DueDate, f.PaidDate = nullStr(desc), parseTime(due), nullTime(paid)
		if f.PaidDate != nil {
			f.PaidCents = f.AmountCents
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// paidSinceReturnSQL selects the sum of the payments assigned to the fee f
// since its last returned debit: a return reverses everything paid before,
// however the fee had been paid (by hand, direct debit or transfer). Link
// ids are UUIDv7 and therefore ordered by creation.
const paidSinceReturnSQL = `COALESCE((SELECT SUM(p.amount_cents) FROM member_fee_payments p
			JOIN finance_transactions t ON t.id = p.transaction_id AND t.deleted_at IS NULL
			WHERE p.fee_id = f.id AND p.deleted_at IS NULL AND p.amount_cents > 0
			AND p.id > COALESCE((SELECT MAX(r.id) FROM member_fee_payments r
				JOIN finance_transactions rt ON rt.id = r.transaction_id AND rt.deleted_at IS NULL
				WHERE r.fee_id = f.id AND r.deleted_at IS NULL AND r.amount_cents < 0), '')), 0)`

// ReconcileMembers returns the live members with what identifies them on a
// bank statement: account holder, IBAN and household.
func (s *Store) ReconcileMembers(ctx context.Context) ([]reconcile.Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, m.number, m.first_name, m.last_name,
		b.account_holder, b.iban, h.id
	FROM members m
	LEFT JOIN member_bank_details b ON b.member_id = m.id AND b.deleted_at IS NULL
	LEFT JOIN household_members hm ON hm.member_id = m.id AND hm.deleted_at IS NULL
	LEFT JOIN households h ON h.id = hm.household_id AND h.deleted_at IS NULL
	WHERE m.deleted_at IS NULL ORDER BY m.number, m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []reconcile.Member{}
	for rows.Next() {
		var m reconcile.Member
		var holder, iban, household sql.NullString
		if err := rows.Scan(&m.ID, &m.Number, &m.FirstName, &m.LastName, &holder, &iban, &household); err != nil {
			return nil, err
		}
		m.AccountHolder, m.IBAN, m.HouseholdID = holder.String, iban.String, household.String
		result = append(result, m)
	}
	return result, rows.Err()
}

// ConfirmReconciliation assigns an imported transaction to fees. For an
// income the fees must be open and the amounts are payments; for a returned
// debit the fees must be paid and the amounts are reversed. A fee whose
// payments reach its amount gets the date of the transaction as paid date, a
// fee that falls below is open again.
func (s *Store) ConfirmReconciliation(ctx context.Context, transactionID string, allocations []Allocation) (ReconciliationEntry, error) {
	if len(allocations) == 0 {
		return ReconciliationEntry{}, ErrNoAllocations
	}
	err := s.inWriteTx(ctx, func(w *writer) error {
		entry, err := loadEntry(ctx, w.Tx, transactionID)
		if err != nil {
			return err
		}
		if len(entry.Payments) > 0 {
			return ErrAlreadyReconciled
		}
		if entry.Type != "income" && !entry.Bank.IsReturn {
			return ErrNotReconcilable
		}
		ids := make([]string, 0, len(allocations))
		var total int64
		seen := map[string]bool{}
		for _, a := range allocations {
			if a.AmountCents <= 0 || seen[a.FeeID] {
				return &AllocationError{FeeIDs: []string{a.FeeID}}
			}
			seen[a.FeeID] = true
			ids = append(ids, a.FeeID)
			total += a.AmountCents
		}
		if total > entry.AmountCents {
			return ErrAllocationExceeds
		}
		fees, err := reconcileFees(ctx, w.Tx, ids)
		if err != nil {
			return err
		}
		byID := map[string]ReconcileFee{}
		for _, f := range fees {
			byID[f.ID] = f
		}
		invalid := []string{}
		for _, a := range allocations {
			f, ok := byID[a.FeeID]
			switch {
			case !ok:
				invalid = append(invalid, a.FeeID)
			case entry.Bank.IsReturn && (f.PaidCents <= 0 || a.AmountCents > f.PaidCents):
				invalid = append(invalid, a.FeeID)
			case !entry.Bank.IsReturn && (!f.Open() || a.AmountCents > f.AmountCents-f.PaidCents):
				invalid = append(invalid, a.FeeID)
			}
		}
		if len(invalid) > 0 {
			sort.Strings(invalid)
			return &AllocationError{FeeIDs: invalid}
		}

		now := db.Now()
		for _, a := range allocations {
			amount := a.AmountCents
			if entry.Bank.IsReturn {
				amount = -amount
			}
			if err := w.insert("member_fee_payments", row{
				"id": db.NewID(), "fee_id": a.FeeID, "transaction_id": transactionID, "amount_cents": amount,
				"prior_paid_date": timeArg(byID[a.FeeID].PaidDate), "created_at": now,
			}); err != nil {
				return err
			}
		}
		if entry.DismissedAt != nil {
			if _, err := w.update("finance_transaction_bank", `transaction_id = ?`, []any{transactionID},
				row{"dismissed_at": nil}); err != nil {
				return err
			}
		}
		for _, id := range ids {
			if err := settleFee(w, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ReconciliationEntry{}, err
	}
	return s.reconciliationEntry(ctx, transactionID)
}

// DismissReconciliation marks an open imported transaction as reviewed
// without a fee to assign (e.g. a donation).
func (s *Store) DismissReconciliation(ctx context.Context, transactionID string) (ReconciliationEntry, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		entry, err := loadEntry(ctx, w.Tx, transactionID)
		if err != nil {
			return err
		}
		if len(entry.Payments) > 0 {
			return ErrAlreadyReconciled
		}
		_, err = w.update("finance_transaction_bank", `transaction_id = ? AND dismissed_at IS NULL`, []any{transactionID},
			row{"dismissed_at": db.Now()})
		return err
	})
	if err != nil {
		return ReconciliationEntry{}, err
	}
	return s.reconciliationEntry(ctx, transactionID)
}

// ReopenReconciliation undoes the assignments (or the dismissal) of an
// imported transaction. Fees without other assignments get back the paid
// date they had before.
func (s *Store) ReopenReconciliation(ctx context.Context, transactionID string) (ReconciliationEntry, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if _, err := loadEntry(ctx, w.Tx, transactionID); err != nil {
			return err
		}
		if _, err := w.update("finance_transaction_bank", `transaction_id = ? AND dismissed_at IS NOT NULL`,
			[]any{transactionID}, row{"dismissed_at": nil}); err != nil {
			return err
		}
		return removePayments(w, `transaction_id = ?`, transactionID)
	})
	if err != nil {
		return ReconciliationEntry{}, err
	}
	return s.reconciliationEntry(ctx, transactionID)
}

// removePayments tombstones the live assignments matching where and settles
// their fees: a fee whose removed assignments were its latest gets back the
// paid date it had before them, otherwise it is settled from the remaining
// ones.
func removePayments(w *writer, where, arg string) error {
	rows, err := w.QueryContext(w.ctx, `SELECT id, fee_id, prior_paid_date FROM member_fee_payments
		WHERE deleted_at IS NULL AND `+where+` ORDER BY id`, arg)
	if err != nil {
		return err
	}
	type removed struct {
		firstID string
		prior   *string
	}
	byFee := map[string]removed{}
	var order []string
	for rows.Next() {
		var id, fee string
		var prior sql.NullString
		if err := rows.Scan(&id, &fee, &prior); err != nil {
			rows.Close()
			return err
		}
		if _, ok := byFee[fee]; !ok {
			byFee[fee] = removed{firstID: id, prior: nullStr(prior)}
			order = append(order, fee)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(order) == 0 {
		return err
	}
	if _, err := w.update("member_fee_payments", `deleted_at IS NULL AND `+where, []any{arg},
		row{"deleted_at": db.Now()}); err != nil {
		return err
	}
	for _, fee := range order {
		var later int
		if err := w.QueryRowContext(w.ctx, `SELECT COUNT(*) FROM member_fee_payments p
			JOIN finance_transactions t ON t.id = p.transaction_id AND t.deleted_at IS NULL
			WHERE p.fee_id = ? AND p.deleted_at IS NULL AND p.id > ?`, fee, byFee[fee].firstID).Scan(&later); err != nil {
			return err
		}
		if later > 0 {
			if err := settleFee(w, fee); err != nil {
				return err
			}
			continue
		}
		if _, err := w.update("member_fees", `id = ? AND deleted_at IS NULL`, []any{fee},
			row{"paid_date": strArg(byFee[fee].prior)}); err != nil {
			return err
		}
	}
	return nil
}

// settleFee sets the paid date of a fee from its assignments: open after a
// returned debit, paid (on the date of the last payment) once the payments
// since then reach the amount. A fee without assignments is left as it is.
func settleFee(w *writer, feeID string) error {
	var amount, linked int64
	var paidDate, lastPayment, lastLink sql.NullString
	err := w.QueryRowContext(w.ctx, `SELECT f.amount_cents, f.paid_date, `+paidSinceReturnSQL+`,
		(SELECT MAX(t.date) FROM member_fee_payments p
			JOIN finance_transactions t ON t.id = p.transaction_id AND t.deleted_at IS NULL
			WHERE p.fee_id = f.id AND p.deleted_at IS NULL AND p.amount_cents > 0),
		(SELECT CASE WHEN p.amount_cents < 0 THEN 'return' ELSE 'payment' END FROM member_fee_payments p
			JOIN finance_transactions t ON t.id = p.transaction_id AND t.deleted_at IS NULL
			WHERE p.fee_id = f.id AND p.deleted_at IS NULL ORDER BY p.id DESC LIMIT 1)
		FROM member_fees f WHERE f.id = ? AND f.deleted_at IS NULL`, feeID).
		Scan(&amount, &paidDate, &linked, &lastPayment, &lastLink)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var paid any
	switch {
	case !lastLink.Valid:
		return nil
	case lastLink.String == "return" || linked < amount:
		paid = nil
	case paidDate.Valid:
		return nil
	case lastPayment.Valid:
		paid = db.FormatTime(dateOnly(parseTime(lastPayment.String)))
	default:
		paid = db.Now()
	}
	_, err = w.update("member_fees", `id = ? AND deleted_at IS NULL`, []any{feeID}, row{"paid_date": paid})
	return err
}

func loadEntry(ctx context.Context, q queryer, transactionID string) (ReconciliationEntry, error) {
	entries, err := reconciliationEntries(ctx, q, transactionID)
	if err != nil {
		return ReconciliationEntry{}, err
	}
	if len(entries) == 0 {
		return ReconciliationEntry{}, ErrNotFound
	}
	return entries[0], nil
}

func (s *Store) reconciliationEntry(ctx context.Context, transactionID string) (ReconciliationEntry, error) {
	return loadEntry(ctx, s.db, transactionID)
}
