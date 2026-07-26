package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// GetStatistics returns aggregated statistics from the database.
// The statistics IDs follow the convention expected by the frontend.
func (a *API) GetStatistics(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	var memberCount, activeMembers, groups, roles, sections int
	var totalFees, paidFees int

	_ = db.QueryRow(`SELECT COUNT(*) FROM "Member"`).Scan(&memberCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "Member" WHERE "exitDate" IS NULL`).Scan(&activeMembers)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "Group"`).Scan(&groups)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "Role"`).Scan(&roles)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "ClubSection"`).Scan(&sections)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "MemberFee"`).Scan(&totalFees)
	_ = db.QueryRow(`SELECT COUNT(*) FROM "MemberFee" WHERE "paidDate" IS NOT NULL`).Scan(&paidFees)

	stats := []openapi.Statistic{
		{Id: 1, Value: memberCount},
		{Id: 2, Value: activeMembers},
		{Id: 3, Value: groups},
		{Id: 4, Value: roles},
		{Id: 5, Value: sections},
		{Id: 6, Value: totalFees},
		{Id: 7, Value: paidFees},
	}
	writeJSON(w, http.StatusOK, stats)
}

// dashboardCharts is the response type for the chart statistics endpoint.
type dashboardCharts struct {
	MemberGrowth      []memberGrowthPoint      `json:"memberGrowth"`
	FinanceTimeSeries []financeTimeSeriesPoint  `json:"financeTimeSeries"`
	FeeStatus         feeStatus                `json:"feeStatus"`
	MembersBySection  []sectionCount           `json:"membersBySection"`
}

type memberGrowthPoint struct {
	Month   string `json:"month"`
	Entries int    `json:"entries"`
	Exits   int    `json:"exits"`
}

type financeTimeSeriesPoint struct {
	Month    string  `json:"month"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
}

type feeStatus struct {
	Paid int `json:"paid"`
	Open int `json:"open"`
}

type sectionCount struct {
	Section string `json:"section"`
	Count   int    `json:"count"`
}

// isMySQL returns true when the server is configured for MySQL.
func (a *API) isMySQL() bool {
	return a.cfg.DatabaseURL != ""
}

// monthFmt returns the SQL expression to format a datetime column as YYYY-MM,
// adapted for the configured database driver (SQLite vs MySQL).
func (a *API) monthFmt(col string) string {
	if a.isMySQL() {
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m')", col)
	}
	return fmt.Sprintf("strftime('%%Y-%%m', %s)", col)
}

// last12MonthsFilter returns a SQL condition that limits a datetime column to
// the last 12 months, adapted for the configured database driver.
func (a *API) last12MonthsFilter(col string) string {
	if a.isMySQL() {
		return fmt.Sprintf("%s >= DATE_SUB(NOW(), INTERVAL 12 MONTH)", col)
	}
	return fmt.Sprintf("%s >= date('now', '-12 months')", col)
}

// currentYearFilter returns a SQL condition that limits an integer year column
// to the current calendar year.
func (a *API) currentYearFilter(col string) string {
	if a.isMySQL() {
		return fmt.Sprintf("%s = YEAR(NOW())", col)
	}
	return fmt.Sprintf("%s = CAST(strftime('%%Y', 'now') AS INTEGER)", col)
}

// GetStatisticsCharts returns chart-ready statistics for the dashboard.
// It is registered as a custom route alongside the generated OpenAPI routes.
func (a *API) GetStatisticsCharts(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	db := a.getDB()

	charts := dashboardCharts{
		MemberGrowth:      a.queryMemberGrowth(db),
		FinanceTimeSeries: a.queryFinanceTimeSeries(db),
		FeeStatus:         a.queryFeeStatus(db),
		MembersBySection:  a.queryMembersBySection(db),
	}

	writeJSON(w, http.StatusOK, charts)
}

// queryMemberGrowth returns monthly entry and exit counts for the last 12 months.
func (a *API) queryMemberGrowth(db *sql.DB) []memberGrowthPoint {
	monthCol := a.monthFmt(`"entryDate"`)
	filter := a.last12MonthsFilter(`"entryDate"`)
	query := fmt.Sprintf(
		`SELECT %s as month, COUNT(*) as entries FROM "Member" WHERE "entryDate" IS NOT NULL AND %s GROUP BY month ORDER BY month`,
		monthCol, filter,
	)

	entries := map[string]int{}
	if rows, err := db.Query(query); err == nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var count int
			if rows.Scan(&month, &count) == nil {
				entries[month] = count
			}
		}
	}

	monthCol = a.monthFmt(`"exitDate"`)
	filter = a.last12MonthsFilter(`"exitDate"`)
	query = fmt.Sprintf(
		`SELECT %s as month, COUNT(*) as exits FROM "Member" WHERE "exitDate" IS NOT NULL AND %s GROUP BY month ORDER BY month`,
		monthCol, filter,
	)

	exits := map[string]int{}
	if rows, err := db.Query(query); err == nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var count int
			if rows.Scan(&month, &count) == nil {
				exits[month] = count
			}
		}
	}

	// Merge entries and exits by month, preserving order.
	seen := map[string]bool{}
	var months []string
	for m := range entries {
		if !seen[m] {
			seen[m] = true
			months = append(months, m)
		}
	}
	for m := range exits {
		if !seen[m] {
			seen[m] = true
			months = append(months, m)
		}
	}

	// Sort months chronologically.
	sortStrings(months)

	points := make([]memberGrowthPoint, 0, len(months))
	for _, m := range months {
		points = append(points, memberGrowthPoint{
			Month:   m,
			Entries: entries[m],
			Exits:   exits[m],
		})
	}
	return points
}

// queryFinanceTimeSeries returns monthly income and expense totals for the last 12 months.
func (a *API) queryFinanceTimeSeries(db *sql.DB) []financeTimeSeriesPoint {
	monthCol := a.monthFmt(`"date"`)
	filter := a.last12MonthsFilter(`"date"`)
	query := fmt.Sprintf(`
		SELECT %s as month,
		       COALESCE(SUM(CASE WHEN "type" = 'income' THEN "amount" ELSE 0 END), 0) as income,
		       COALESCE(SUM(CASE WHEN "type" = 'expense' THEN "amount" ELSE 0 END), 0) as expenses
		FROM "FinanceTransaction"
		WHERE %s
		GROUP BY month
		ORDER BY month
	`, monthCol, filter)

	rows, err := db.Query(query)
	if err != nil {
		return []financeTimeSeriesPoint{}
	}
	defer rows.Close()

	points := []financeTimeSeriesPoint{}
	for rows.Next() {
		var p financeTimeSeriesPoint
		if err := rows.Scan(&p.Month, &p.Income, &p.Expenses); err == nil {
			points = append(points, p)
		}
	}
	return points
}

// queryFeeStatus returns paid vs. open member fee counts for the current year.
func (a *API) queryFeeStatus(db *sql.DB) feeStatus {
	filter := a.currentYearFilter(`"year"`)
	query := fmt.Sprintf(`
		SELECT
		    COALESCE(SUM(CASE WHEN "paidDate" IS NOT NULL THEN 1 ELSE 0 END), 0) as paid,
		    COALESCE(SUM(CASE WHEN "paidDate" IS NULL THEN 1 ELSE 0 END), 0) as open
		FROM "MemberFee"
		WHERE %s
	`, filter)

	var fs feeStatus
	_ = db.QueryRow(query).Scan(&fs.Paid, &fs.Open)
	return fs
}

// queryMembersBySection returns member counts grouped by club section.
// The _MemberSections join table uses A=ClubSection.id, B=Member.id (Prisma convention).
func (a *API) queryMembersBySection(db *sql.DB) []sectionCount {
	query := `
		SELECT cs."name", COUNT(ms."B") as count
		FROM "ClubSection" cs
		LEFT JOIN "_MemberSections" ms ON ms."A" = cs."id"
		GROUP BY cs."id", cs."name"
		ORDER BY count DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return []sectionCount{}
	}
	defer rows.Close()

	counts := []sectionCount{}
	for rows.Next() {
		var sc sectionCount
		if err := rows.Scan(&sc.Section, &sc.Count); err == nil {
			counts = append(counts, sc)
		}
	}
	return counts
}

// sortStrings sorts a slice of strings in-place.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

