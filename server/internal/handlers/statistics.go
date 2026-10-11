package handlers

import (
	"log"
	"net/http"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// GetStatistics returns aggregated statistics from the database.
// The statistics IDs follow the convention expected by the frontend.
func (a *API) GetStatistics(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	c, err := st.Counts(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to compute statistics")
		return
	}

	stats := []openapi.Statistic{
		{Id: 1, Value: c.Members},
		{Id: 2, Value: c.ActiveMembers},
		{Id: 3, Value: c.Groups},
		{Id: 4, Value: c.Roles},
		{Id: 5, Value: c.Sections},
		{Id: 6, Value: c.Fees},
		{Id: 7, Value: c.PaidFees},
	}
	writeJSON(w, http.StatusOK, stats)
}

// dashboardCharts is the response type for the chart statistics endpoint.
type dashboardCharts struct {
	MemberGrowth      []memberGrowthPoint      `json:"memberGrowth"`
	FinanceTimeSeries []financeTimeSeriesPoint `json:"financeTimeSeries"`
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

// GetStatisticsCharts returns chart-ready statistics for the dashboard.
// It is registered as a custom route alongside the generated OpenAPI routes.
// A failing series is logged and returned empty so the dashboard still renders.
func (a *API) GetStatisticsCharts(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	ctx := r.Context()
	charts := dashboardCharts{
		MemberGrowth:      []memberGrowthPoint{},
		FinanceTimeSeries: []financeTimeSeriesPoint{},
		MembersBySection:  []sectionCount{},
	}

	if growth, err := st.MemberGrowth(ctx); err != nil {
		log.Printf("statistics: member growth: %v", err)
	} else {
		for _, g := range growth {
			charts.MemberGrowth = append(charts.MemberGrowth, memberGrowthPoint{Month: g.Month, Entries: g.Entries, Exits: g.Exits})
		}
	}

	if series, err := st.FinanceSeries(ctx); err != nil {
		log.Printf("statistics: finance series: %v", err)
	} else {
		for _, s := range series {
			charts.FinanceTimeSeries = append(charts.FinanceTimeSeries, financeTimeSeriesPoint{Month: s.Month, Income: s.Income, Expenses: s.Expenses})
		}
	}

	if paid, open, err := st.FeeStatus(ctx); err != nil {
		log.Printf("statistics: fee status: %v", err)
	} else {
		charts.FeeStatus = feeStatus{Paid: paid, Open: open}
	}

	if sections, err := st.MembersBySection(ctx); err != nil {
		log.Printf("statistics: members by section: %v", err)
	} else {
		for _, s := range sections {
			charts.MembersBySection = append(charts.MembersBySection, sectionCount{Section: s.Section, Count: s.Count})
		}
	}

	writeJSON(w, http.StatusOK, charts)
}
