package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/fees"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

const monthLayout = "2006-01"

type feeModelJSON struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Interval string  `json:"interval"`
	MinAge   *int    `json:"minAge,omitempty"`
	MaxAge   *int    `json:"maxAge,omitempty"`
	ProRata  bool    `json:"proRata"`
}

func toFeeModelJSON(m fees.Model) feeModelJSON {
	return feeModelJSON{ID: m.ID, Name: m.Name, Amount: db.FromCents(m.AmountCents), Interval: string(m.Interval),
		MinAge: m.MinAge, MaxAge: m.MaxAge, ProRata: m.ProRata}
}

type feeAssignmentJSON struct {
	Mode       string  `json:"mode"`
	FeeModelID *string `json:"feeModelId,omitempty"`
}

type memberFeeModelJSON struct {
	Member honorMemberJSON `json:"member"`
	Mode   string          `json:"mode"`
	Model  *feeModelJSON   `json:"model,omitempty"`
	Age    *int            `json:"age,omitempty"`
}

type feePeriodJSON struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Label string `json:"label"`
}

func toFeePeriodJSON(p fees.Period, i fees.Interval) feePeriodJSON {
	return feePeriodJSON{Start: p.Start.Format(dateLayout), End: p.End.Format(dateLayout), Label: p.Label(i)}
}

type feeRunItemJSON struct {
	Member      honorMemberJSON `json:"member"`
	Model       feeModelJSON    `json:"model"`
	Period      feePeriodJSON   `json:"period"`
	Amount      float64         `json:"amount"`
	ProRata     bool            `json:"proRata"`
	DueDate     string          `json:"dueDate"`
	Description string          `json:"description"`
}

type feeRunSkipJSON struct {
	Member honorMemberJSON `json:"member"`
	Reason string          `json:"reason"`
	Model  *feeModelJSON   `json:"model,omitempty"`
	Period *feePeriodJSON  `json:"period,omitempty"`
}

type feeRunPlanJSON struct {
	Count   int              `json:"count"`
	Total   float64          `json:"total"`
	Items   []feeRunItemJSON `json:"items"`
	Skipped []feeRunSkipJSON `json:"skipped"`
}

type feeRunJSON struct {
	ID        string  `json:"id"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	DueDate   string  `json:"dueDate"`
	CreatedAt string  `json:"createdAt"`
	Count     int     `json:"count"`
	Total     float64 `json:"total"`
	PaidCount int     `json:"paidCount"`
}

func toFeeRunJSON(r store.FeeRun) feeRunJSON {
	return feeRunJSON{ID: r.ID, From: r.From.Format(monthLayout), To: r.To.Format(monthLayout),
		DueDate: r.DueDate.Format(dateLayout), CreatedAt: db.FormatTime(r.CreatedAt), Count: r.Count,
		Total: db.FromCents(r.TotalCents), PaidCount: r.PaidCount}
}

// feeAccess checks that the login may read the finances or, for writes,
// change them (only the roles that manage finances).
func (a *API) feeAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok
	if allowed && write {
		allowed = slices.Contains(financeEditors, claims.AppRole) && a.collab.CanWrite(r.Context(), db.ScopeFinance)
	} else if allowed {
		allowed = a.collab.CanRead(r.Context(), db.ScopeFinance)
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access fee models"})
	}
	return allowed
}

// ListFeeModels: GET /api/v2/fee-models.
func (a *API) ListFeeModels(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, false) {
		return
	}
	list, err := st.ListFeeModels(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list fee models")
		return
	}
	out := make([]feeModelJSON, 0, len(list))
	for _, m := range list {
		out = append(out, toFeeModelJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeFeeModel(w http.ResponseWriter, r *http.Request) (fees.Model, bool) {
	var req feeModelJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return fees.Model{}, false
	}
	m := fees.Model{Name: strings.TrimSpace(req.Name), AmountCents: db.ToCents(req.Amount),
		Interval: fees.Interval(req.Interval), MinAge: req.MinAge, MaxAge: req.MaxAge, ProRata: req.ProRata}
	validAge := func(v *int) bool { return v == nil || (*v >= 0 && *v <= 150) }
	switch {
	case m.Name == "" || len(m.Name) > 100:
		badRequest(w, "name is required (at most 100 characters)")
	case m.AmountCents <= 0 || m.AmountCents > 100_000_00:
		badRequest(w, "amount must be greater than 0 and at most 100000")
	case !m.Interval.Valid():
		badRequest(w, "interval must be yearly, half_yearly, quarterly or monthly")
	case !validAge(m.MinAge) || !validAge(m.MaxAge):
		badRequest(w, "minAge and maxAge must be between 0 and 150")
	case m.MinAge != nil && m.MaxAge != nil && *m.MinAge > *m.MaxAge:
		badRequest(w, "minAge must not be greater than maxAge")
	default:
		return m, true
	}
	return fees.Model{}, false
}

func writeFeeModelError(w http.ResponseWriter, err error, notFound, failure string) {
	if errors.Is(err, store.ErrConflict) {
		code := "FEE_MODEL_NAME_TAKEN"
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code, Error: "a fee model with this name already exists"})
		return
	}
	writeStoreError(w, err, notFound, failure)
}

// CreateFeeModel: POST /api/v2/fee-models.
func (a *API) CreateFeeModel(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	in, ok := decodeFeeModel(w, r)
	if !ok {
		return
	}
	m, err := st.CreateFeeModel(r.Context(), in)
	if err != nil {
		writeFeeModelError(w, err, "", "failed to create fee model")
		return
	}
	st.Audit(r.Context(), "CREATE", "FeeModel", m.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toFeeModelJSON(m))
}

// UpdateFeeModel: PUT /api/v2/fee-models/{id}.
func (a *API) UpdateFeeModel(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	in, ok := decodeFeeModel(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	m, err := st.UpdateFeeModel(r.Context(), id, in)
	if err != nil {
		writeFeeModelError(w, err, "fee model not found", "failed to update fee model")
		return
	}
	st.Audit(r.Context(), "UPDATE", "FeeModel", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toFeeModelJSON(m))
}

// DeleteFeeModel: DELETE /api/v2/fee-models/{id}.
func (a *API) DeleteFeeModel(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteFeeModel(r.Context(), id); err != nil {
		writeStoreError(w, err, "fee model not found", "failed to delete fee model")
		return
	}
	st.Audit(r.Context(), "DELETE", "FeeModel", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

// ListMemberFeeModels: GET /api/v2/fee-models/members?date=YYYY-MM-DD lists
// the fee model of every member on the day (default today).
func (a *API) ListMemberFeeModels(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, false) {
		return
	}
	on := time.Now()
	if v := r.URL.Query().Get("date"); v != "" {
		d, err := time.Parse(dateLayout, v)
		if err != nil {
			badRequest(w, "date must be a date (YYYY-MM-DD)")
			return
		}
		on = d
	}
	list, err := st.FeeModelsOn(r.Context(), on)
	if err != nil {
		writeStoreError(w, err, "", "failed to list member fee models")
		return
	}
	out := make([]memberFeeModelJSON, 0, len(list))
	for _, m := range list {
		j := memberFeeModelJSON{Member: memberRefJSON(m.Member), Mode: m.Mode, Age: m.Age}
		if m.Model != nil {
			model := toFeeModelJSON(*m.Model)
			j.Model = &model
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

// GetMemberFeeModel: GET /api/v2/members/{id}/fee-model.
func (a *API) GetMemberFeeModel(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, false) {
		return
	}
	asg, err := st.GetFeeAssignment(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to load fee model")
		return
	}
	writeJSON(w, http.StatusOK, feeAssignmentJSON{Mode: asg.Mode, FeeModelID: asg.ModelID})
}

// SetMemberFeeModel: PUT /api/v2/members/{id}/fee-model {"mode", "feeModelId"}.
func (a *API) SetMemberFeeModel(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	var req feeAssignmentJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	asg := store.FeeAssignment{Mode: req.Mode}
	switch req.Mode {
	case store.FeeModeAuto, store.FeeModeExempt:
	case store.FeeModeModel:
		if req.FeeModelID == nil || *req.FeeModelID == "" {
			badRequest(w, "feeModelId is required for mode model")
			return
		}
		asg.ModelID = req.FeeModelID
	default:
		badRequest(w, "mode must be auto, model or exempt")
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.SetFeeAssignment(r.Context(), id, asg); err != nil {
		writeStoreError(w, err, "member not found", "failed to save fee model")
		return
	}
	st.Audit(r.Context(), "UPDATE", "MemberFeeModel", id, currentUserID(r))
	writeJSON(w, http.StatusOK, feeAssignmentJSON{Mode: asg.Mode, FeeModelID: asg.ModelID})
}

// decodeFeeRun reads {"from": "YYYY-MM", "to": "YYYY-MM", "dueDate"} (dueDate
// defaults to the first day of from).
func decodeFeeRun(w http.ResponseWriter, r *http.Request) (store.FeeRunInput, bool) {
	var req struct {
		From    string  `json:"from"`
		To      string  `json:"to"`
		DueDate *string `json:"dueDate"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.FeeRunInput{}, false
	}
	from, errFrom := time.Parse(monthLayout, req.From)
	to, errTo := time.Parse(monthLayout, req.To)
	if errFrom != nil || errTo != nil {
		badRequest(w, "from and to must be months (YYYY-MM)")
		return store.FeeRunInput{}, false
	}
	if to.Before(from) || to.After(from.AddDate(2, -1, 0)) {
		badRequest(w, "to must be between from and 24 months after it")
		return store.FeeRunInput{}, false
	}
	in := store.FeeRunInput{From: from, To: to, DueDate: from}
	if req.DueDate != nil && *req.DueDate != "" {
		due, err := time.Parse(dateLayout, *req.DueDate)
		if err != nil {
			badRequest(w, "dueDate must be a date (YYYY-MM-DD)")
			return store.FeeRunInput{}, false
		}
		in.DueDate = due
	}
	return in, true
}

func toFeeRunPlanJSON(p store.FeeRunPlan) feeRunPlanJSON {
	out := feeRunPlanJSON{Count: len(p.Items), Total: db.FromCents(p.TotalCents()),
		Items: make([]feeRunItemJSON, 0, len(p.Items)), Skipped: make([]feeRunSkipJSON, 0, len(p.Skipped))}
	for _, it := range p.Items {
		out.Items = append(out.Items, feeRunItemJSON{Member: memberRefJSON(it.Member), Model: toFeeModelJSON(it.Model),
			Period: toFeePeriodJSON(it.Period, it.Model.Interval), Amount: db.FromCents(it.AmountCents),
			ProRata: it.ProRata(), DueDate: it.DueDate.Format(dateLayout), Description: it.Description()})
	}
	for _, s := range p.Skipped {
		j := feeRunSkipJSON{Member: memberRefJSON(s.Member), Reason: s.Reason}
		if s.Model != nil {
			model := toFeeModelJSON(*s.Model)
			j.Model = &model
			if s.Period != nil {
				period := toFeePeriodJSON(*s.Period, s.Model.Interval)
				j.Period = &period
			}
		}
		out.Skipped = append(out.Skipped, j)
	}
	return out
}

// PreviewFeeRun: POST /api/v2/fee-runs/preview shows the fees a billing run
// would create, without saving anything.
func (a *API) PreviewFeeRun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, false) {
		return
	}
	in, ok := decodeFeeRun(w, r)
	if !ok {
		return
	}
	plan, err := st.PlanFeeRun(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to plan billing run")
		return
	}
	writeJSON(w, http.StatusOK, toFeeRunPlanJSON(plan))
}

// ListFeeRuns: GET /api/v2/fee-runs.
func (a *API) ListFeeRuns(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, false) {
		return
	}
	runs, err := st.ListFeeRuns(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list billing runs")
		return
	}
	out := make([]feeRunJSON, 0, len(runs))
	for _, run := range runs {
		out = append(out, toFeeRunJSON(run))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateFeeRun: POST /api/v2/fee-runs creates the fees of a billing run.
// Periods already charged to a member are skipped; a run that would create
// no fee is rejected with FEE_RUN_EMPTY.
func (a *API) CreateFeeRun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	in, ok := decodeFeeRun(w, r)
	if !ok {
		return
	}
	run, err := st.CreateFeeRun(r.Context(), in)
	if errors.Is(err, store.ErrFeeRunEmpty) {
		code := "FEE_RUN_EMPTY"
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code,
			Error: "no fees to create: the periods have already been charged or no member has a fee model"})
		return
	}
	if err != nil {
		writeStoreError(w, err, "", "failed to create billing run")
		return
	}
	st.Audit(r.Context(), "CREATE", "FeeRun", run.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toFeeRunJSON(run))
}

// DeleteFeeRun: DELETE /api/v2/fee-runs/{id} removes the open fees of a run.
func (a *API) DeleteFeeRun(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.feeAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	removed, err := st.DeleteFeeRun(r.Context(), id)
	if errors.Is(err, store.ErrFeeRunInCollection) {
		code := "FEE_RUN_IN_COLLECTION"
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code,
			Error: "some fees of the run are in a direct debit run that has not been confirmed"})
		return
	}
	if err != nil {
		writeStoreError(w, err, "billing run not found", "failed to delete billing run")
		return
	}
	st.Audit(r.Context(), "DELETE", "FeeRun", id, currentUserID(r))
	writeJSON(w, http.StatusOK, map[string]int{"removed": removed})
}
