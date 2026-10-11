package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Transaction is a cash book entry (scope finance).
type Transaction struct {
	ID          string
	Date        time.Time
	Description string
	Amount      float64
	Type        string // "income" | "expense"
	Category    *string
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TransactionInput holds the writable fields of a transaction.
type TransactionInput struct {
	Date        time.Time
	Description string
	Amount      float64
	Type        string
	Category    *string
	Notes       *string
}

// Fee is a membership fee owed by a member (scope finance).
type Fee struct {
	ID          string
	Member      MemberSummary
	Amount      float64
	DueDate     time.Time
	PaidDate    *time.Time
	Description *string
	Year        int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FeeInput holds the writable fields of a fee.
type FeeInput struct {
	MemberID    string
	Amount      float64
	DueDate     time.Time
	PaidDate    *time.Time
	Description *string
	Year        int
}

const transactionSelect = `SELECT id, date, description, amount_cents, type, category, notes, created_at, updated_at
FROM finance_transactions WHERE deleted_at IS NULL`

func scanTransaction(row scanner) (Transaction, error) {
	var t Transaction
	var date, createdAt, updatedAt string
	var cents int64
	var category, notes sql.NullString
	if err := row.Scan(&t.ID, &date, &t.Description, &cents, &t.Type, &category, &notes, &createdAt, &updatedAt); err != nil {
		return t, mapErr(err)
	}
	t.Date, t.CreatedAt, t.UpdatedAt = parseTime(date), parseTime(createdAt), parseTime(updatedAt)
	t.Amount = db.FromCents(cents)
	t.Category, t.Notes = nullStr(category), nullStr(notes)
	return t, nil
}

// ListTransactions returns all live transactions, newest first.
func (s *Store) ListTransactions(ctx context.Context) ([]Transaction, error) {
	rows, err := s.db.QueryContext(ctx, transactionSelect+` ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// GetTransaction returns a live transaction.
func (s *Store) GetTransaction(ctx context.Context, id string) (Transaction, error) {
	return scanTransaction(s.db.QueryRowContext(ctx, transactionSelect+` AND id = ?`, id))
}

// CreateTransaction inserts a transaction.
func (s *Store) CreateTransaction(ctx context.Context, in TransactionInput) (Transaction, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		values := transactionColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("finance_transactions", values)
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.GetTransaction(ctx, id)
}

// UpdateTransaction replaces the writable fields of a live transaction.
func (s *Store) UpdateTransaction(ctx context.Context, id string, in TransactionInput) (Transaction, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("finance_transactions", `id = ? AND deleted_at IS NULL`, []any{id}, transactionColumns(in)))
	})
	if err != nil {
		return Transaction{}, err
	}
	return s.GetTransaction(ctx, id)
}

func transactionColumns(in TransactionInput) row {
	return row{
		"date": db.FormatTime(in.Date), "description": in.Description, "amount_cents": db.ToCents(in.Amount),
		"type": in.Type, "category": strArg(in.Category), "notes": strArg(in.Notes),
	}
}

// DeleteTransaction tombstones a transaction.
func (s *Store) DeleteTransaction(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("finance_transactions", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

const feeSelect = `SELECT f.id, f.amount_cents, f.due_date, f.paid_date, f.description, f.year, f.created_at, f.updated_at,
       m.id, m.number, m.first_name, m.last_name
FROM member_fees f
JOIN members m ON m.id = f.member_id
WHERE f.deleted_at IS NULL`

func scanFee(row scanner) (Fee, error) {
	var f Fee
	var cents int64
	var dueDate, createdAt, updatedAt string
	var paidDate, description sql.NullString
	err := row.Scan(&f.ID, &cents, &dueDate, &paidDate, &description, &f.Year, &createdAt, &updatedAt,
		&f.Member.ID, &f.Member.Number, &f.Member.FirstName, &f.Member.LastName)
	if err != nil {
		return f, mapErr(err)
	}
	f.Amount = db.FromCents(cents)
	f.DueDate, f.CreatedAt, f.UpdatedAt = parseTime(dueDate), parseTime(createdAt), parseTime(updatedAt)
	f.PaidDate, f.Description = nullTime(paidDate), nullStr(description)
	return f, nil
}

// ListFees returns all live fees, newest year first.
func (s *Store) ListFees(ctx context.Context) ([]Fee, error) {
	rows, err := s.db.QueryContext(ctx, feeSelect+` ORDER BY f.year DESC, f.due_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Fee{}
	for rows.Next() {
		f, err := scanFee(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// GetFee returns a live fee.
func (s *Store) GetFee(ctx context.Context, id string) (Fee, error) {
	return scanFee(s.db.QueryRowContext(ctx, feeSelect+` AND f.id = ?`, id))
}

// CreateFee inserts a fee for a live member.
func (s *Store) CreateFee(ctx context.Context, in FeeInput) (Fee, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, in.MemberID); err != nil {
			return err
		}
		values := feeColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("member_fees", values)
	})
	if err != nil {
		return Fee{}, err
	}
	return s.GetFee(ctx, id)
}

// UpdateFee replaces the writable fields of a live fee.
func (s *Store) UpdateFee(ctx context.Context, id string, in FeeInput) (Fee, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, in.MemberID); err != nil {
			return err
		}
		return affected(w.update("member_fees", `id = ? AND deleted_at IS NULL`, []any{id}, feeColumns(in)))
	})
	if err != nil {
		return Fee{}, err
	}
	return s.GetFee(ctx, id)
}

func feeColumns(in FeeInput) row {
	return row{
		"member_id": in.MemberID, "amount_cents": db.ToCents(in.Amount), "due_date": db.FormatTime(in.DueDate),
		"paid_date": timeArg(in.PaidDate), "description": strArg(in.Description), "year": in.Year,
	}
}

// DeleteFee tombstones a fee.
func (s *Store) DeleteFee(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("member_fees", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

// ResetFinance tombstones all live fees and transactions and reports how many
// of each were removed.
func (s *Store) ResetFinance(ctx context.Context) (fees, transactions int, err error) {
	err = s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		var err error
		if fees, err = w.update("member_fees", `deleted_at IS NULL`, nil, tombstone); err != nil {
			return err
		}
		transactions, err = w.update("finance_transactions", `deleted_at IS NULL`, nil, tombstone)
		return err
	})
	return fees, transactions, err
}

// requireLiveMember returns ErrInvalidReference unless id names a live member.
func requireLiveMember(ctx context.Context, tx *sql.Tx, id string) error {
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM members WHERE id = ? AND deleted_at IS NULL`, id).Scan(&one)
	if err == sql.ErrNoRows {
		return ErrInvalidReference
	}
	return err
}
