package handlers

import (
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

