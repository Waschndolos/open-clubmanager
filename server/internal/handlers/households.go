package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type householdMemberJSON struct {
	honorMemberJSON
	Birthday *string `json:"birthday,omitempty"`
	ExitDate *string `json:"exitDate,omitempty"`
	Active   bool    `json:"active"`
}

type householdJSON struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Payer        *honorMemberJSON      `json:"payer,omitempty"`
	ShareAddress bool                  `json:"shareAddress"`
	ShareBank    bool                  `json:"shareBank"`
	Members      []householdMemberJSON `json:"members"`
	// PayerMissing is set when active members remain without an active payer.
	PayerMissing bool `json:"payerMissing"`
}

type householdRequest struct {
	Name         string   `json:"name"`
	PayerID      *string  `json:"payerId"`
	ShareAddress bool     `json:"shareAddress"`
	ShareBank    bool     `json:"shareBank"`
	MemberIDs    []string `json:"memberIds"`
}

type guardianJSON struct {
	ID       string           `json:"id"`
	Member   *honorMemberJSON `json:"member,omitempty"`
	Name     *string          `json:"name,omitempty"`
	Relation *string          `json:"relation,omitempty"`
	Email    *string          `json:"email,omitempty"`
	Phone    *string          `json:"phone,omitempty"`
}

type guardianRequest struct {
	MemberID *string `json:"memberId"`
	Name     *string `json:"name"`
	Relation *string `json:"relation"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

func toHouseholdJSON(h store.Household, now time.Time) householdJSON {
	j := householdJSON{ID: h.ID, Name: h.Name, ShareAddress: h.ShareAddress, ShareBank: h.ShareBank,
		Members: make([]householdMemberJSON, 0, len(h.Members)), PayerMissing: h.PayerMissing(now)}
	if h.Payer != nil {
		p := memberRefJSON(*h.Payer)
		j.Payer = &p
	}
	for _, m := range h.Members {
		j.Members = append(j.Members, householdMemberJSON{honorMemberJSON: memberRefJSON(m.MemberRef),
			Birthday: formatDate(m.Birthday), ExitDate: formatDate(m.ExitDate), Active: m.Active(now)})
	}
	return j
}

// scopeAccess checks that the login may read (or, for writes, change) the
// given scopes; read-only and trainer logins never write.
func (a *API) scopeAccess(w http.ResponseWriter, r *http.Request, write bool, what string, scopes ...db.Scope) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok && (!write || canEditData(claims.AppRole))
	for _, s := range scopes {
		if !allowed {
			break
		}
		if write {
			allowed = a.collab.CanWrite(r.Context(), s)
		} else {
			allowed = a.collab.CanRead(r.Context(), s)
		}
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access " + what})
	}
	return allowed
}

// canEditData reports whether an app role may change club data in general.
// Trainers only record attendance (see attendanceAccess).
func canEditData(role openapi.AppRole) bool {
	return role != openapi.READONLY && role != openapi.TRAINER
}

func writeHouseholdError(w http.ResponseWriter, err error, failure string) {
	if errors.Is(err, store.ErrMemberInOtherHousehold) {
		code := "MEMBER_IN_OTHER_HOUSEHOLD"
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code, Error: "a member already belongs to another household"})
		return
	}
	writeStoreError(w, err, "household not found", failure)
}

// ListHouseholds: GET /api/v2/households.
func (a *API) ListHouseholds(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "households", db.ScopeMembersCore) {
		return
	}
	list, err := st.ListHouseholds(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list households")
		return
	}
	now := time.Now()
	out := make([]householdJSON, 0, len(list))
	for _, h := range list {
		out = append(out, toHouseholdJSON(h, now))
	}
	writeJSON(w, http.StatusOK, out)
}

// decodeHousehold reads the request and checks access: sharing the address or
// bank details writes those scopes of every member of the household.
func (a *API) decodeHousehold(w http.ResponseWriter, r *http.Request) (store.HouseholdInput, bool) {
	var req householdRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.HouseholdInput{}, false
	}
	scopes := []db.Scope{db.ScopeMembersCore}
	if req.ShareAddress {
		scopes = append(scopes, db.ScopeMembersContact)
	}
	if req.ShareBank {
		scopes = append(scopes, db.ScopeMembersBank)
	}
	if !a.scopeAccess(w, r, true, "households", scopes...) {
		return store.HouseholdInput{}, false
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		badRequest(w, "name is required (at most 100 characters)")
		return store.HouseholdInput{}, false
	}
	if len(req.MemberIDs) == 0 || len(req.MemberIDs) > 50 {
		badRequest(w, "a household has between 1 and 50 members")
		return store.HouseholdInput{}, false
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(req.MemberIDs))
	for _, id := range req.MemberIDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if req.PayerID != nil && *req.PayerID == "" {
		req.PayerID = nil
	}
	if req.PayerID != nil && !seen[*req.PayerID] {
		badRequest(w, "the payer must be a member of the household")
		return store.HouseholdInput{}, false
	}
	return store.HouseholdInput{Name: name, PayerID: req.PayerID, ShareAddress: req.ShareAddress,
		ShareBank: req.ShareBank, MemberIDs: ids}, true
}

// CreateHousehold: POST /api/v2/households.
func (a *API) CreateHousehold(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	in, ok := a.decodeHousehold(w, r)
	if !ok {
		return
	}
	h, err := st.CreateHousehold(r.Context(), in)
	if err != nil {
		writeHouseholdError(w, err, "failed to create household")
		return
	}
	st.Audit(r.Context(), "CREATE", "Household", h.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toHouseholdJSON(h, time.Now()))
}

// UpdateHousehold: PUT /api/v2/households/{id}.
func (a *API) UpdateHousehold(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	in, ok := a.decodeHousehold(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	h, err := st.UpdateHousehold(r.Context(), id, in)
	if err != nil {
		writeHouseholdError(w, err, "failed to update household")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Household", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toHouseholdJSON(h, time.Now()))
}

// DeleteHousehold: DELETE /api/v2/households/{id}.
func (a *API) DeleteHousehold(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "households", db.ScopeMembersCore) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteHousehold(r.Context(), id); err != nil {
		writeStoreError(w, err, "household not found", "failed to delete household")
		return
	}
	st.Audit(r.Context(), "DELETE", "Household", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

// ListGuardians: GET /api/v2/members/{id}/guardians.
func (a *API) ListGuardians(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "guardians", db.ScopeMembersContact) {
		return
	}
	list, err := st.ListGuardians(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "", "failed to list guardians")
		return
	}
	out := make([]guardianJSON, 0, len(list))
	for _, g := range list {
		j := guardianJSON{ID: g.ID, Name: g.Name, Relation: g.Relation, Email: g.Email, Phone: g.Phone}
		if g.Member != nil {
			m := memberRefJSON(*g.Member)
			j.Member = &m
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

func trimmed(s *string, max int) (*string, bool) {
	if s == nil {
		return nil, true
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, true
	}
	return &v, len(v) <= max
}

// CreateGuardian: POST /api/v2/members/{id}/guardians.
func (a *API) CreateGuardian(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "guardians", db.ScopeMembersContact) {
		return
	}
	var req guardianRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	var in store.GuardianInput
	ok = true
	for _, f := range []struct {
		dst **string
		src *string
		max int
	}{{&in.GuardianMemberID, req.MemberID, 64}, {&in.Name, req.Name, 200}, {&in.Relation, req.Relation, 100},
		{&in.Email, req.Email, 254}, {&in.Phone, req.Phone, 50}} {
		var valid bool
		*f.dst, valid = trimmed(f.src, f.max)
		ok = ok && valid
	}
	if !ok {
		badRequest(w, "a field is too long")
		return
	}
	if (in.GuardianMemberID == nil) == (in.Name == nil) {
		badRequest(w, "either memberId or name is required")
		return
	}
	id, err := st.CreateGuardian(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to create guardian")
		return
	}
	st.Audit(r.Context(), "CREATE", "MemberGuardian", id, currentUserID(r))
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// DeleteGuardian: DELETE /api/v2/guardians/{id}.
func (a *API) DeleteGuardian(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "guardians", db.ScopeMembersContact) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteGuardian(r.Context(), id); err != nil {
		writeStoreError(w, err, "guardian not found", "failed to delete guardian")
		return
	}
	st.Audit(r.Context(), "DELETE", "MemberGuardian", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}
