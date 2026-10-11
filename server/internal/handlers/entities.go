package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// namedResource binds a named-entity kind to its audit-log entity name and
// the label used in error messages.
type namedResource struct {
	kind   store.NamedKind
	entity string
	label  string
}

var (
	roleResource    = namedResource{store.Roles, "Role", "role"}
	groupResource   = namedResource{store.Groups, "Group", "group"}
	sectionResource = namedResource{store.Sections, "ClubSection", "section"}
)

// ── Roles ────────────────────────────────────────────────────────────────────

func (a *API) ListRoles(w http.ResponseWriter, r *http.Request) { a.listNamed(w, r, roleResource) }
func (a *API) CreateRole(w http.ResponseWriter, r *http.Request) {
	a.createNamed(w, r, roleResource)
}
func (a *API) UpdateRole(w http.ResponseWriter, r *http.Request, id string) {
	a.updateNamed(w, r, roleResource, id)
}
func (a *API) DeleteRole(w http.ResponseWriter, r *http.Request, id string) {
	a.deleteNamed(w, r, roleResource, id)
}
func (a *API) GetRoleMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.namedMembers(w, r, roleResource, id)
}
func (a *API) BulkAssignRoleMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.assignNamedMembers(w, r, roleResource, id)
}

// ── Groups ───────────────────────────────────────────────────────────────────

func (a *API) ListGroups(w http.ResponseWriter, r *http.Request) { a.listNamed(w, r, groupResource) }
func (a *API) CreateGroup(w http.ResponseWriter, r *http.Request) {
	a.createNamed(w, r, groupResource)
}
func (a *API) UpdateGroup(w http.ResponseWriter, r *http.Request, id string) {
	a.updateNamed(w, r, groupResource, id)
}
func (a *API) DeleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	a.deleteNamed(w, r, groupResource, id)
}
func (a *API) GetGroupMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.namedMembers(w, r, groupResource, id)
}
func (a *API) BulkAssignGroupMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.assignNamedMembers(w, r, groupResource, id)
}

// ── Sections ─────────────────────────────────────────────────────────────────

func (a *API) ListSections(w http.ResponseWriter, r *http.Request) {
	a.listNamed(w, r, sectionResource)
}
func (a *API) CreateSection(w http.ResponseWriter, r *http.Request) {
	a.createNamed(w, r, sectionResource)
}
func (a *API) UpdateSection(w http.ResponseWriter, r *http.Request, id string) {
	a.updateNamed(w, r, sectionResource, id)
}
func (a *API) DeleteSection(w http.ResponseWriter, r *http.Request, id string) {
	a.deleteNamed(w, r, sectionResource, id)
}
func (a *API) GetSectionMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.namedMembers(w, r, sectionResource, id)
}
func (a *API) BulkAssignSectionMembers(w http.ResponseWriter, r *http.Request, id string) {
	a.assignNamedMembers(w, r, sectionResource, id)
}

// ── generic named-entity handlers ─────────────────────────────────────────────

func (a *API) listNamed(w http.ResponseWriter, r *http.Request, res namedResource) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	entities, err := st.ListNamed(r.Context(), res.kind)
	if err != nil {
		writeStoreError(w, err, "", "failed to list "+res.label+"s")
		return
	}
	writeJSON(w, http.StatusOK, toAPINamedList(entities))
}

func (a *API) createNamed(w http.ResponseWriter, r *http.Request, res namedResource) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.NamedEntityCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	e, err := st.CreateNamed(r.Context(), res.kind, req.Name)
	if err != nil {
		writeStoreError(w, err, "", "failed to create "+res.label)
		return
	}
	st.Audit(r.Context(), "CREATE", res.entity, e.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, openapi.NamedEntity{Id: e.ID, Name: e.Name})
}

func (a *API) updateNamed(w http.ResponseWriter, r *http.Request, res namedResource, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.NamedEntityUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	e, err := st.RenameNamed(r.Context(), res.kind, id, req.Name)
	if err != nil {
		writeStoreError(w, err, res.label+" not found", "failed to update "+res.label)
		return
	}
	st.Audit(r.Context(), "UPDATE", res.entity, id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.NamedEntity{Id: e.ID, Name: e.Name})
}

func (a *API) deleteNamed(w http.ResponseWriter, r *http.Request, res namedResource, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteNamed(r.Context(), res.kind, id); err != nil {
		writeStoreError(w, err, res.label+" not found", "failed to delete "+res.label)
		return
	}
	st.Audit(r.Context(), "DELETE", res.entity, id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// namedMembers returns the members assigned to a role, group or section.
func (a *API) namedMembers(w http.ResponseWriter, r *http.Request, res namedResource, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	members, err := st.NamedMembers(r.Context(), res.kind, id)
	if err != nil {
		writeStoreError(w, err, res.label+" not found", "failed to list "+res.label+" members")
		return
	}
	out := make([]openapi.MemberSummary, len(members))
	for i, m := range members {
		out[i] = toAPIMemberSummary(m)
	}
	writeJSON(w, http.StatusOK, out)
}

// assignNamedMembers replaces all member assignments of a role, group or
// section.
func (a *API) assignNamedMembers(w http.ResponseWriter, r *http.Request, res namedResource, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.BulkMemberAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if err := st.SetNamedMembers(r.Context(), res.kind, id, req.MemberIds); err != nil {
		writeStoreError(w, err, res.label+" not found", "failed to update "+res.label+" members")
		return
	}
	st.Audit(r.Context(), "UPDATE", res.entity, id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}
