package store

import (
	"context"
	"sort"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Counts are the headline numbers of the dashboard.
type Counts struct {
	Members       int
	ActiveMembers int
	Groups        int
	Roles         int
	Sections      int
	Fees          int
	PaidFees      int
}

// Counts returns the number of live rows of the main entities.
func (s *Store) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM members WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM members WHERE deleted_at IS NULL AND exit_date IS NULL),
		(SELECT COUNT(*) FROM groups WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM sections WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM member_fees WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM member_fees WHERE deleted_at IS NULL AND paid_date IS NOT NULL)`,
	).Scan(&c.Members, &c.ActiveMembers, &c.Groups, &c.Roles, &c.Sections, &c.Fees, &c.PaidFees)
	return c, err
}

// MonthCount is the number of member entries and exits in a month (YYYY-MM).
type MonthCount struct {
	Month   string
	Entries int
	Exits   int
}

// MemberGrowth returns monthly entries and exits of the last 12 months.
func (s *Store) MemberGrowth(ctx context.Context) ([]MonthCount, error) {
	byMonth := map[string]*MonthCount{}
	for _, col := range []string{"entry_date", "exit_date"} {
		rows, err := s.db.QueryContext(ctx,
			`SELECT strftime('%Y-%m', `+col+`) AS month, COUNT(*) FROM members
			 WHERE deleted_at IS NULL AND `+col+` IS NOT NULL AND `+col+` >= date('now', '-12 months')
			 GROUP BY month`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var month string
			var n int
			if err := rows.Scan(&month, &n); err != nil {
				rows.Close()
				return nil, err
			}
			mc := byMonth[month]
			if mc == nil {
				mc = &MonthCount{Month: month}
				byMonth[month] = mc
			}
			if col == "entry_date" {
				mc.Entries = n
			} else {
				mc.Exits = n
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	result := make([]MonthCount, 0, len(byMonth))
	for _, mc := range byMonth {
		result = append(result, *mc)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Month < result[j].Month })
	return result, nil
}

// MonthTotals are the income and expense totals of a month (YYYY-MM).
type MonthTotals struct {
	Month    string
	Income   float64
	Expenses float64
}

// FinanceSeries returns monthly income and expense totals of the last 12
// months.
func (s *Store) FinanceSeries(ctx context.Context) ([]MonthTotals, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT strftime('%Y-%m', date) AS month,
		        COALESCE(SUM(CASE WHEN type = 'income' THEN amount_cents ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END), 0)
		 FROM finance_transactions
		 WHERE deleted_at IS NULL AND date >= date('now', '-12 months')
		 GROUP BY month ORDER BY month`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []MonthTotals{}
	for rows.Next() {
		var t MonthTotals
		var income, expenses int64
		if err := rows.Scan(&t.Month, &income, &expenses); err != nil {
			return nil, err
		}
		t.Income, t.Expenses = db.FromCents(income), db.FromCents(expenses)
		result = append(result, t)
	}
	return result, rows.Err()
}

// FeeStatus returns the number of paid and open fees of the current year.
func (s *Store) FeeStatus(ctx context.Context) (paid, open int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(paid_date IS NOT NULL), 0), COALESCE(SUM(paid_date IS NULL), 0)
		 FROM member_fees
		 WHERE deleted_at IS NULL AND year = CAST(strftime('%Y', 'now') AS INTEGER)`,
	).Scan(&paid, &open)
	return paid, open, err
}

// SectionCount is the number of live members in a section.
type SectionCount struct {
	Section string
	Count   int
}

// MembersBySection returns member counts per live section, largest first.
func (s *Store) MembersBySection(ctx context.Context) ([]SectionCount, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sec.name, COUNT(m.id) AS n
		 FROM sections sec
		 LEFT JOIN member_sections l ON l.section_id = sec.id AND l.deleted_at IS NULL
		 LEFT JOIN members m ON m.id = l.member_id AND m.deleted_at IS NULL
		 WHERE sec.deleted_at IS NULL
		 GROUP BY sec.id, sec.name
		 ORDER BY n DESC, sec.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SectionCount{}
	for rows.Next() {
		var c SectionCount
		if err := rows.Scan(&c.Section, &c.Count); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
