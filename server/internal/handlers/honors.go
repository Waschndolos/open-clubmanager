package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// Default jubilee stages: membership years and round birthdays.
var (
	defaultMembershipYears = []int{10, 25, 40, 50, 60, 70}
	defaultJubileeAges     = []int{50, 60, 70, 75, 80, 85, 90, 95, 100}
)

type honorMemberJSON struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type honorJSON struct {
	ID        string          `json:"id"`
	Member    honorMemberJSON `json:"member"`
	Kind      string          `json:"kind"`
	Milestone *int            `json:"milestone,omitempty"`
	Title     string          `json:"title"`
	Date      string          `json:"date"`
	Note      *string         `json:"note,omitempty"`
}

type jubileeJSON struct {
	Member    honorMemberJSON `json:"member"`
	Kind      string          `json:"kind"`
	Milestone int             `json:"milestone"`
	Date      string          `json:"date"`
}

type honorRequest struct {
	Kind      string  `json:"kind"`
	Milestone *int    `json:"milestone"`
	Title     string  `json:"title"`
	Date      string  `json:"date"`
	Note      *string `json:"note"`
}

const dateLayout = "2006-01-02"

func memberRefJSON(m store.MemberRef) honorMemberJSON {
	return honorMemberJSON{ID: m.ID, Number: m.Number, FirstName: m.FirstName, LastName: m.LastName}
}

func toHonorJSON(h store.Honor) honorJSON {
	return honorJSON{ID: h.ID, Member: memberRefJSON(h.Member), Kind: h.Kind, Milestone: h.Milestone,
		Title: h.Title, Date: h.Date.Format(dateLayout), Note: h.Note}
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: msg})
}

// honorAccess checks that the caller may read (or, with write, manage) member
// data. These routes are not part of the generated spec, so the role and
// scope checks of the generated routes are repeated here.
func (a *API) honorAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok
	if write {
		allowed = ok && (claims.AppRole == openapi.ADMIN || claims.AppRole == openapi.SECRETARY) &&
			a.collab.CanWrite(r.Context(), db.ScopeMembersCore)
	} else {
		allowed = ok && a.collab.CanRead(r.Context(), db.ScopeMembersCore)
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access honors"})
	}
	return allowed
}

func parseIntList(s string, def []int) ([]int, bool) {
	if s == "" {
		return def, true
	}
	out := []int{}
	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n <= 0 || n > 200 {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

// ListJubilees returns the upcoming anniversaries and round birthdays that
// have not been honored yet: GET /api/v2/jubilees?from=&to=&years=&ages=&format=json|csv.
// from defaults to today, to to 90 days later; years and ages are comma
// separated lists that default to the standard stages.
func (a *API) ListJubilees(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, false) {
		return
	}
	q := r.URL.Query()
	today := time.Now().UTC()
	from := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 90)
	var err error
	if v := q.Get("from"); v != "" {
		if from, err = time.Parse(dateLayout, v); err != nil {
			badRequest(w, "from must be a date (YYYY-MM-DD)")
			return
		}
	}
	if v := q.Get("to"); v != "" {
		if to, err = time.Parse(dateLayout, v); err != nil {
			badRequest(w, "to must be a date (YYYY-MM-DD)")
			return
		}
	}
	if to.Before(from) || to.Sub(from) > 5*366*24*time.Hour {
		badRequest(w, "to must not be before from and at most five years after it")
		return
	}
	years, ok1 := parseIntList(q.Get("years"), defaultMembershipYears)
	ages, ok2 := parseIntList(q.Get("ages"), defaultJubileeAges)
	if !ok1 || !ok2 {
		badRequest(w, "years and ages must be comma separated positive numbers")
		return
	}
	format := q.Get("format")
	if format != "" && format != "json" && format != "csv" {
		badRequest(w, "format must be json or csv")
		return
	}

	list, err := st.Jubilees(r.Context(), store.JubileeQuery{From: from, To: to, MembershipYears: years, Ages: ages})
	if err != nil {
		writeStoreError(w, err, "", "failed to compute jubilees")
		return
	}
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="jubilees.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"date", "kind", "milestone", "number", "firstName", "lastName"})
		for _, j := range list {
			_ = cw.Write([]string{j.Date.Format(dateLayout), j.Kind, strconv.Itoa(j.Milestone),
				strconv.Itoa(j.Member.Number), csvSafe(j.Member.FirstName), csvSafe(j.Member.LastName)})
		}
		cw.Flush()
		return
	}
	out := make([]jubileeJSON, 0, len(list))
	for _, j := range list {
		out = append(out, jubileeJSON{Member: memberRefJSON(j.Member), Kind: j.Kind, Milestone: j.Milestone, Date: j.Date.Format(dateLayout)})
	}
	writeJSON(w, http.StatusOK, out)
}

// csvSafe neutralises spreadsheet formulas in exported text.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// ListHonors returns all honors, or those of one member with ?memberId=.
func (a *API) ListHonors(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, false) {
		return
	}
	list, err := st.ListHonors(r.Context(), r.URL.Query().Get("memberId"))
	if err != nil {
		writeStoreError(w, err, "", "failed to list honors")
		return
	}
	out := make([]honorJSON, 0, len(list))
	for _, h := range list {
		out = append(out, toHonorJSON(h))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateHonor records an honor: POST /api/v2/members/{id}/honors.
func (a *API) CreateHonor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, true) {
		return
	}
	var req honorRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	date, err := time.Parse(dateLayout, req.Date)
	req.Title = strings.TrimSpace(req.Title)
	switch {
	case err != nil:
		badRequest(w, "date must be a date (YYYY-MM-DD)")
		return
	case req.Title == "" || len(req.Title) > 200:
		badRequest(w, "title is required (at most 200 characters)")
		return
	case req.Kind != store.HonorMembership && req.Kind != store.HonorBirthday && req.Kind != store.HonorOther:
		badRequest(w, "kind must be membership, birthday or other")
		return
	case req.Kind != store.HonorOther && (req.Milestone == nil || *req.Milestone <= 0):
		badRequest(w, "milestone is required for membership and birthday honors")
		return
	}
	if req.Kind == store.HonorOther {
		req.Milestone = nil
	}
	id := chi.URLParam(r, "id")
	h, err := st.CreateHonor(r.Context(), id, store.HonorInput{
		Kind: req.Kind, Milestone: req.Milestone, Title: req.Title, Date: date, Note: req.Note,
	})
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to create honor")
		return
	}
	st.Audit(r.Context(), "CREATE", "MemberHonor", h.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toHonorJSON(h))
}

// DeleteHonor removes an honor: DELETE /api/v2/honors/{id}.
func (a *API) DeleteHonor(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteHonor(r.Context(), id); err != nil {
		writeStoreError(w, err, "honor not found", "failed to delete honor")
		return
	}
	st.Audit(r.Context(), "DELETE", "MemberHonor", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

type membershipAdjust struct {
	Years int `json:"years"`
}

// GetMembershipAdjust returns the manual correction of a member's membership years.
func (a *API) GetMembershipAdjust(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, false) {
		return
	}
	years, err := st.MembershipAdjust(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to read membership correction")
		return
	}
	writeJSON(w, http.StatusOK, membershipAdjust{Years: years})
}

// SetMembershipAdjust sets it: PUT /api/v2/members/{id}/membership-adjustment.
func (a *API) SetMembershipAdjust(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.honorAccess(w, r, true) {
		return
	}
	var req membershipAdjust
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil || req.Years < -100 || req.Years > 100 {
		badRequest(w, "years must be a number between -100 and 100")
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.SetMembershipAdjust(r.Context(), id, req.Years); err != nil {
		writeStoreError(w, err, "member not found", "failed to set membership correction")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Member", id, currentUserID(r))
	writeJSON(w, http.StatusOK, req)
}
