package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Item is a piece of club inventory.
type Item struct {
	ID            string
	Name          string
	Description   *string
	SerialNumber  *string
	Category      string
	Quantity      int
	Location      string
	PurchaseDate  *time.Time
	PurchasePrice *float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ItemInput holds the writable fields of an item.
type ItemInput struct {
	Name          string
	Description   *string
	SerialNumber  *string
	Category      string
	Quantity      int
	Location      string
	PurchaseDate  *time.Time
	PurchasePrice *float64
}

// Loan records an item lent to a member.
type Loan struct {
	ID         string
	ItemID     string
	MemberID   string
	LoanedAt   time.Time
	DueDate    *time.Time
	ReturnedAt *time.Time
	Notes      string
	CreatedAt  time.Time
}

// LoanInput holds the writable fields of a loan.
type LoanInput struct {
	ItemID     string
	MemberID   string
	LoanedAt   time.Time
	DueDate    *time.Time
	ReturnedAt *time.Time
	Notes      string
}

const itemSelect = `SELECT id, name, description, serial_number, category, quantity, location, purchase_date,
       purchase_price_cents, created_at, updated_at
FROM inventory_items WHERE deleted_at IS NULL`

func scanItem(row scanner) (Item, error) {
	var it Item
	var description, serial, purchaseDate sql.NullString
	var price sql.NullInt64
	var createdAt, updatedAt string
	err := row.Scan(&it.ID, &it.Name, &description, &serial, &it.Category, &it.Quantity, &it.Location, &purchaseDate,
		&price, &createdAt, &updatedAt)
	if err != nil {
		return it, mapErr(err)
	}
	it.Description, it.SerialNumber = nullStr(description), nullStr(serial)
	it.PurchaseDate, it.PurchasePrice = nullTime(purchaseDate), nullAmount(price)
	it.CreatedAt, it.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
	return it, nil
}

// ListItems returns all live items ordered by name.
func (s *Store) ListItems(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, itemSelect+` ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

// GetItem returns a live item.
func (s *Store) GetItem(ctx context.Context, id string) (Item, error) {
	return scanItem(s.db.QueryRowContext(ctx, itemSelect+` AND id = ?`, id))
}

// CreateItem inserts an item.
func (s *Store) CreateItem(ctx context.Context, in ItemInput) (Item, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		values := itemColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("inventory_items", values)
	})
	if err != nil {
		return Item{}, err
	}
	return s.GetItem(ctx, id)
}

// UpdateItem replaces the writable fields of a live item.
func (s *Store) UpdateItem(ctx context.Context, id string, in ItemInput) (Item, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("inventory_items", `id = ? AND deleted_at IS NULL`, []any{id}, itemColumns(in)))
	})
	if err != nil {
		return Item{}, err
	}
	return s.GetItem(ctx, id)
}

func itemColumns(in ItemInput) row {
	return row{
		"name": in.Name, "description": strArg(in.Description), "serial_number": strArg(in.SerialNumber),
		"category": in.Category, "quantity": in.Quantity, "location": in.Location,
		"purchase_date": timeArg(in.PurchaseDate), "purchase_price_cents": centsArg(in.PurchasePrice),
	}
}

// DeleteItem tombstones an item and its loans.
func (s *Store) DeleteItem(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update("inventory_items", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		_, err := w.update("inventory_loans", `item_id = ? AND deleted_at IS NULL`, []any{id}, tombstone)
		return err
	})
}

const loanSelect = `SELECT id, item_id, member_id, loaned_at, due_date, returned_at, notes, created_at
FROM inventory_loans WHERE deleted_at IS NULL`

func scanLoan(row scanner) (Loan, error) {
	var l Loan
	var loanedAt, createdAt string
	var dueDate, returnedAt sql.NullString
	if err := row.Scan(&l.ID, &l.ItemID, &l.MemberID, &loanedAt, &dueDate, &returnedAt, &l.Notes, &createdAt); err != nil {
		return l, mapErr(err)
	}
	l.LoanedAt, l.CreatedAt = parseTime(loanedAt), parseTime(createdAt)
	l.DueDate, l.ReturnedAt = nullTime(dueDate), nullTime(returnedAt)
	return l, nil
}

// ListLoans returns all live loans, open ones first, newest first.
func (s *Store) ListLoans(ctx context.Context) ([]Loan, error) {
	rows, err := s.db.QueryContext(ctx, loanSelect+` ORDER BY returned_at IS NOT NULL, loaned_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Loan{}
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

// GetLoan returns a live loan.
func (s *Store) GetLoan(ctx context.Context, id string) (Loan, error) {
	return scanLoan(s.db.QueryRowContext(ctx, loanSelect+` AND id = ?`, id))
}

// CreateLoan inserts a loan of a live item to a live member.
func (s *Store) CreateLoan(ctx context.Context, in LoanInput) (Loan, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveLoanRefs(ctx, w.Tx, in); err != nil {
			return err
		}
		values := loanColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("inventory_loans", values)
	})
	if err != nil {
		return Loan{}, err
	}
	return s.GetLoan(ctx, id)
}

// UpdateLoan replaces the writable fields of a live loan.
func (s *Store) UpdateLoan(ctx context.Context, id string, in LoanInput) (Loan, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveLoanRefs(ctx, w.Tx, in); err != nil {
			return err
		}
		return affected(w.update("inventory_loans", `id = ? AND deleted_at IS NULL`, []any{id}, loanColumns(in)))
	})
	if err != nil {
		return Loan{}, err
	}
	return s.GetLoan(ctx, id)
}

func loanColumns(in LoanInput) row {
	return row{
		"item_id": in.ItemID, "member_id": in.MemberID, "loaned_at": db.FormatTime(in.LoanedAt),
		"due_date": timeArg(in.DueDate), "returned_at": timeArg(in.ReturnedAt), "notes": in.Notes,
	}
}

// DeleteLoan tombstones a loan.
func (s *Store) DeleteLoan(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("inventory_loans", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

func requireLiveLoanRefs(ctx context.Context, tx *sql.Tx, in LoanInput) error {
	if err := requireLiveMember(ctx, tx, in.MemberID); err != nil {
		return err
	}
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM inventory_items WHERE id = ? AND deleted_at IS NULL`, in.ItemID).Scan(&one)
	if err == sql.ErrNoRows {
		return ErrInvalidReference
	}
	return err
}
