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
	now := db.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO inventory_items (id, name, description, serial_number, category, quantity, location, purchase_date,
		   purchase_price_cents, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.Name, strArg(in.Description), strArg(in.SerialNumber), in.Category, in.Quantity, in.Location,
		timeArg(in.PurchaseDate), centsArg(in.PurchasePrice), now, now)
	if err != nil {
		return Item{}, mapErr(err)
	}
	return s.GetItem(ctx, id)
}

// UpdateItem replaces the writable fields of a live item.
func (s *Store) UpdateItem(ctx context.Context, id string, in ItemInput) (Item, error) {
	err := requireAffected(s.db.ExecContext(ctx,
		`UPDATE inventory_items SET name = ?, description = ?, serial_number = ?, category = ?, quantity = ?, location = ?,
		   purchase_date = ?, purchase_price_cents = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		in.Name, strArg(in.Description), strArg(in.SerialNumber), in.Category, in.Quantity, in.Location,
		timeArg(in.PurchaseDate), centsArg(in.PurchasePrice), db.Now(), id))
	if err != nil {
		return Item{}, err
	}
	return s.GetItem(ctx, id)
}

// DeleteItem tombstones an item and its loans.
func (s *Store) DeleteItem(ctx context.Context, id string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE inventory_items SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE inventory_loans SET deleted_at = ?, updated_at = ? WHERE item_id = ? AND deleted_at IS NULL`, now, now, id)
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
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := requireLiveLoanRefs(ctx, tx, in); err != nil {
			return err
		}
		now := db.Now()
		_, err := tx.ExecContext(ctx,
			`INSERT INTO inventory_loans (id, item_id, member_id, loaned_at, due_date, returned_at, notes, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, in.ItemID, in.MemberID, db.FormatTime(in.LoanedAt), timeArg(in.DueDate), timeArg(in.ReturnedAt), in.Notes, now, now)
		return mapErr(err)
	})
	if err != nil {
		return Loan{}, err
	}
	return s.GetLoan(ctx, id)
}

// UpdateLoan replaces the writable fields of a live loan.
func (s *Store) UpdateLoan(ctx context.Context, id string, in LoanInput) (Loan, error) {
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := requireLiveLoanRefs(ctx, tx, in); err != nil {
			return err
		}
		return requireAffected(tx.ExecContext(ctx,
			`UPDATE inventory_loans SET item_id = ?, member_id = ?, loaned_at = ?, due_date = ?, returned_at = ?, notes = ?, updated_at = ?
			 WHERE id = ? AND deleted_at IS NULL`,
			in.ItemID, in.MemberID, db.FormatTime(in.LoanedAt), timeArg(in.DueDate), timeArg(in.ReturnedAt), in.Notes, db.Now(), id))
	})
	if err != nil {
		return Loan{}, err
	}
	return s.GetLoan(ctx, id)
}

// DeleteLoan tombstones a loan.
func (s *Store) DeleteLoan(ctx context.Context, id string) error {
	now := db.Now()
	return requireAffected(s.db.ExecContext(ctx,
		`UPDATE inventory_loans SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id))
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
