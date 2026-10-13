package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Item conditions; only ready items may be lent.
const (
	ConditionReady    = "ready"
	ConditionDamaged  = "damaged"
	ConditionInRepair = "in_repair"
	ConditionRetired  = "retired"
)

// Maintenance due states relative to a date.
const (
	DueNone    = "none" // no schedule, or the item is retired
	DueOK      = "ok"
	DueSoon    = "due"
	DueOverdue = "overdue"
)

// ErrItemNotReady is returned when an item that is not ready is lent.
var ErrItemNotReady = errors.New("item is not ready for use")

// ValidCondition reports whether c is a known condition.
func ValidCondition(c string) bool {
	switch c {
	case ConditionReady, ConditionDamaged, ConditionInRepair, ConditionRetired:
		return true
	}
	return false
}

// ItemStatus is the condition and maintenance schedule of an item.
type ItemStatus struct {
	Condition      string
	IntervalMonths *int
	NextDue        *time.Time
}

// ItemMaintenanceState is an item with its status and due state, for the
// overview of due maintenance.
type ItemMaintenanceState struct {
	ItemID   string
	Name     string
	Category string
	Location string
	ItemStatus
	LastMaintenance *time.Time
	Due             string
}

// Maintenance is a performed inspection or maintenance of an item.
type Maintenance struct {
	ID            string
	ItemID        string
	Date          time.Time
	PerformedBy   *string
	Note          *string
	Cost          *float64
	TransactionID *string
}

// MaintenanceInput records a maintenance. Condition, when set, becomes the
// item's condition (e.g. ready after a repair). Expense, when set and the cost
// is positive, books the cost as an expense in the cash book.
type MaintenanceInput struct {
	Date        time.Time
	PerformedBy *string
	Note        *string
	Cost        *float64
	Condition   *string
	Expense     *ExpenseInput
}

// ExpenseInput names the cash book entry for a maintenance cost.
type ExpenseInput struct {
	Description string
	Category    *string
}

// MaintenanceStates returns every live item with its status and due state as
// of now: due when the next date is within warnDays. Overdue and due items
// come first.
func (s *Store) MaintenanceStates(ctx context.Context, now time.Time, warnDays int) ([]ItemMaintenanceState, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT i.id, i.name, i.category, i.location,
		COALESCE(st.condition, 'ready'), st.interval_months, st.next_due_date,
		(SELECT MAX(m.performed_on) FROM inventory_maintenance m WHERE m.item_id = i.id AND m.deleted_at IS NULL)
		FROM inventory_items i
		LEFT JOIN inventory_item_status st ON st.item_id = i.id AND st.deleted_at IS NULL
		WHERE i.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ItemMaintenanceState{}
	for rows.Next() {
		var it ItemMaintenanceState
		var months sql.NullInt64
		var next, last sql.NullString
		if err := rows.Scan(&it.ItemID, &it.Name, &it.Category, &it.Location, &it.Condition, &months, &next, &last); err != nil {
			return nil, err
		}
		if months.Valid {
			v := int(months.Int64)
			it.IntervalMonths = &v
		}
		it.NextDue, it.LastMaintenance = nullTime(next), nullTime(last)
		it.Due = dueState(it.ItemStatus, now, warnDays)
		result = append(result, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortMaintenanceStates(result)
	return result, nil
}

func dueState(st ItemStatus, now time.Time, warnDays int) string {
	if st.NextDue == nil || st.Condition == ConditionRetired {
		return DueNone
	}
	due, today := dateOnly(*st.NextDue), dateOnly(now)
	switch {
	case due.Before(today):
		return DueOverdue
	case !due.After(today.AddDate(0, 0, warnDays)):
		return DueSoon
	}
	return DueOK
}

func sortMaintenanceStates(list []ItemMaintenanceState) {
	rank := map[string]int{DueOverdue: 0, DueSoon: 1, DueOK: 2, DueNone: 3}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if rank[a.Due] != rank[b.Due] {
			return rank[a.Due] < rank[b.Due]
		}
		if a.NextDue != nil && b.NextDue != nil && !a.NextDue.Equal(*b.NextDue) {
			return a.NextDue.Before(*b.NextDue)
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ItemID < b.ItemID
	})
}

// SetItemStatus replaces the condition and schedule of a live item.
func (s *Store) SetItemStatus(ctx context.Context, itemID string, in ItemStatus) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveItem(w, itemID); err != nil {
			return err
		}
		return w.upsert("inventory_item_status", row{"item_id": itemID}, row{
			"condition": in.Condition, "interval_months": intArg(in.IntervalMonths),
			"next_due_date": timeArg(in.NextDue), "deleted_at": nil,
		})
	})
}

func requireLiveItem(w *writer, itemID string) error {
	var one int
	err := w.QueryRowContext(w.ctx, `SELECT 1 FROM inventory_items WHERE id = ? AND deleted_at IS NULL`, itemID).Scan(&one)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// requireReadyItem returns ErrItemNotReady unless the item may be lent.
func requireReadyItem(w *writer, itemID string) error {
	var condition string
	err := w.QueryRowContext(w.ctx, `SELECT condition FROM inventory_item_status WHERE item_id = ? AND deleted_at IS NULL`,
		itemID).Scan(&condition)
	if err == sql.ErrNoRows || (err == nil && condition == ConditionReady) {
		return nil
	}
	if err != nil {
		return err
	}
	return ErrItemNotReady
}

// ListMaintenance returns the maintenance history of an item, newest first.
func (s *Store) ListMaintenance(ctx context.Context, itemID string) ([]Maintenance, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, item_id, performed_on, performed_by, note, cost_cents, transaction_id
		FROM inventory_maintenance WHERE item_id = ? AND deleted_at IS NULL
		ORDER BY performed_on DESC, created_at DESC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Maintenance{}
	for rows.Next() {
		var m Maintenance
		var date string
		var by, note, tx sql.NullString
		var cost sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ItemID, &date, &by, &note, &cost, &tx); err != nil {
			return nil, err
		}
		m.Date, m.PerformedBy, m.Note, m.Cost, m.TransactionID = parseTime(date), nullStr(by), nullStr(note), nullAmount(cost), nullStr(tx)
		result = append(result, m)
	}
	return result, rows.Err()
}

// RecordMaintenance stores a maintenance of a live item, moves the next due
// date forward by the interval, optionally sets the condition and books the
// cost as an expense, all in one transaction.
func (s *Store) RecordMaintenance(ctx context.Context, itemID string, in MaintenanceInput) (Maintenance, error) {
	id := db.NewID()
	var txID *string
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveItem(w, itemID); err != nil {
			return err
		}
		if in.Expense != nil && in.Cost != nil && *in.Cost > 0 {
			tid := db.NewID()
			values := transactionColumns(TransactionInput{Date: in.Date, Description: in.Expense.Description,
				Amount: *in.Cost, Type: "expense", Category: in.Expense.Category, Notes: in.Note})
			values["id"], values["created_at"] = tid, db.Now()
			if err := w.insert("finance_transactions", values); err != nil {
				return err
			}
			txID = &tid
		}
		if err := w.insert("inventory_maintenance", row{
			"id": id, "item_id": itemID, "performed_on": db.FormatTime(in.Date), "performed_by": strArg(in.PerformedBy),
			"note": strArg(in.Note), "cost_cents": centsArg(in.Cost), "transaction_id": strArg(txID), "created_at": db.Now(),
		}); err != nil {
			return err
		}

		var condition string
		var months sql.NullInt64
		err := w.QueryRowContext(w.ctx, `SELECT condition, interval_months FROM inventory_item_status
			WHERE item_id = ? AND deleted_at IS NULL`, itemID).Scan(&condition, &months)
		switch {
		case err == sql.ErrNoRows:
			condition = ConditionReady
		case err != nil:
			return err
		}
		set := row{"deleted_at": nil}
		if in.Condition != nil {
			set["condition"] = *in.Condition
		} else if err == sql.ErrNoRows {
			set["condition"] = condition
		}
		if months.Valid {
			set["next_due_date"] = db.FormatTime(in.Date.AddDate(0, int(months.Int64), 0))
		}
		if len(set) == 1 && err == nil {
			return nil // nothing to change
		}
		return w.upsert("inventory_item_status", row{"item_id": itemID}, set)
	})
	if err != nil {
		return Maintenance{}, err
	}
	return Maintenance{ID: id, ItemID: itemID, Date: in.Date, PerformedBy: in.PerformedBy, Note: in.Note,
		Cost: in.Cost, TransactionID: txID}, nil
}

// DeleteMaintenance tombstones a maintenance entry. A booked expense stays in
// the cash book; the next due date is not moved back.
func (s *Store) DeleteMaintenance(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("inventory_maintenance", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}
