package handlers

import (
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

const defaultQualificationWarnDays = 60

type qualificationTypeJSON struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ValidityMonths *int   `json:"validityMonths,omitempty"`
	InspectionOnly bool   `json:"inspectionOnly"`
}

type qualificationJSON struct {
	ID         string                `json:"id"`
	Member     honorMemberJSON       `json:"member"`
	Type       qualificationTypeJSON `json:"type"`
	IssuedDate string                `json:"issuedDate"`
	ExpiryDate *string               `json:"expiryDate,omitempty"`
	Note       *string               `json:"note,omitempty"`
	Status     string                `json:"status"`
}

type qualificationRequest struct {
	TypeID     string  `json:"typeId"`
	IssuedDate string  `json:"issuedDate"`
	ExpiryDate *string `json:"expiryDate"`
	Note       *string `json:"note"`
}

func toQualTypeJSON(t store.QualificationType) qualificationTypeJSON {
	return qualificationTypeJSON{ID: t.ID, Name: t.Name, ValidityMonths: t.ValidityMonths, InspectionOnly: t.InspectionOnly}
}

// qualificationAccess limits qualification data to administrators and
// secretaries (and, for writes, to devices that may change member data).
func (a *API) qualificationAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok && (claims.AppRole == openapi.ADMIN || claims.AppRole == openapi.SECRETARY)
	if allowed {
		if write {
			allowed = a.collab.CanWrite(r.Context(), db.ScopeMembersCore)
		} else {
			allowed = a.collab.CanRead(r.Context(), db.ScopeMembersCore)
		}
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access qualifications"})
	}
	return allowed
}

// ListQualificationTypes: GET /api/v2/qualification-types.
func (a *API) ListQualificationTypes(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, false) {
		return
	}
	list, err := st.ListQualificationTypes(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list qualification types")
		return
	}
	out := make([]qualificationTypeJSON, 0, len(list))
	for _, t := range list {
		out = append(out, toQualTypeJSON(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeQualType(w http.ResponseWriter, r *http.Request) (store.QualificationTypeInput, bool) {
	var req qualificationTypeJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.QualificationTypeInput{}, false
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		badRequest(w, "name is required (at most 100 characters)")
		return store.QualificationTypeInput{}, false
	}
	if req.ValidityMonths != nil && (*req.ValidityMonths <= 0 || *req.ValidityMonths > 1200) {
		badRequest(w, "validityMonths must be between 1 and 1200")
		return store.QualificationTypeInput{}, false
	}
	return store.QualificationTypeInput{Name: name, ValidityMonths: req.ValidityMonths, InspectionOnly: req.InspectionOnly}, true
}

// CreateQualificationType: POST /api/v2/qualification-types.
func (a *API) CreateQualificationType(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, true) {
		return
	}
	in, ok := decodeQualType(w, r)
	if !ok {
		return
	}
	t, err := st.CreateQualificationType(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to create qualification type")
		return
	}
	st.Audit(r.Context(), "CREATE", "QualificationType", t.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toQualTypeJSON(t))
}

// UpdateQualificationType: PUT /api/v2/qualification-types/{id}.
func (a *API) UpdateQualificationType(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, true) {
		return
	}
	in, ok := decodeQualType(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	t, err := st.UpdateQualificationType(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "qualification type not found", "failed to update qualification type")
		return
	}
	st.Audit(r.Context(), "UPDATE", "QualificationType", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toQualTypeJSON(t))
}

// DeleteQualificationType: DELETE /api/v2/qualification-types/{id}.
func (a *API) DeleteQualificationType(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteQualificationType(r.Context(), id); err != nil {
		writeStoreError(w, err, "qualification type not found", "failed to delete qualification type")
		return
	}
	st.Audit(r.Context(), "DELETE", "QualificationType", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

// ListQualifications returns the current qualification per member and type
// with its traffic-light status: GET /api/v2/qualifications?warnDays=60.
func (a *API) ListQualifications(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, false) {
		return
	}
	warn := defaultQualificationWarnDays
	if v := r.URL.Query().Get("warnDays"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 3660 {
			badRequest(w, "warnDays must be a number between 0 and 3660")
			return
		}
		warn = n
	}
	list, err := st.Qualifications(r.Context(), time.Now(), warn)
	if err != nil {
		writeStoreError(w, err, "", "failed to list qualifications")
		return
	}
	out := make([]qualificationJSON, 0, len(list))
	for _, q := range list {
		j := qualificationJSON{ID: q.ID, Member: memberRefJSON(q.Member), Type: toQualTypeJSON(q.Type),
			IssuedDate: q.IssuedDate.Format(dateLayout), Note: q.Note, Status: q.Status}
		if q.ExpiryDate != nil {
			e := q.ExpiryDate.Format(dateLayout)
			j.ExpiryDate = &e
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateQualification: POST /api/v2/members/{id}/qualifications. A renewal is
// a new entry; the list shows the one valid longest.
func (a *API) CreateQualification(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, true) {
		return
	}
	var req qualificationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	issued, err := time.Parse(dateLayout, req.IssuedDate)
	if err != nil {
		badRequest(w, "issuedDate must be a date (YYYY-MM-DD)")
		return
	}
	in := store.QualificationInput{TypeID: req.TypeID, IssuedDate: issued, Note: req.Note}
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		exp, err := time.Parse(dateLayout, *req.ExpiryDate)
		if err != nil || exp.Before(issued) {
			badRequest(w, "expiryDate must be a date (YYYY-MM-DD) not before issuedDate")
			return
		}
		in.ExpiryDate = &exp
	}
	if in.Note != nil && len(*in.Note) > 500 {
		badRequest(w, "note is too long")
		return
	}
	id, err := st.CreateQualification(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeStoreError(w, err, "member not found", "failed to create qualification")
		return
	}
	st.Audit(r.Context(), "CREATE", "MemberQualification", id, currentUserID(r))
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// DeleteQualification: DELETE /api/v2/qualifications/{id}.
func (a *API) DeleteQualification(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.qualificationAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteQualification(r.Context(), id); err != nil {
		writeStoreError(w, err, "qualification not found", "failed to delete qualification")
		return
	}
	st.Audit(r.Context(), "DELETE", "MemberQualification", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}
