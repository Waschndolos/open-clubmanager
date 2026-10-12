package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

type taskJSON struct {
	ID             string           `json:"id"`
	Title          string           `json:"title"`
	Description    *string          `json:"description,omitempty"`
	DueDate        *string          `json:"dueDate,omitempty"`
	Status         string           `json:"status"`
	Recurrence     string           `json:"recurrence"`
	AssigneeMember *honorMemberJSON `json:"assigneeMember,omitempty"`
	AssigneeRole   *taskRoleJSON    `json:"assigneeRole,omitempty"`
	LinkedMember   *honorMemberJSON `json:"linkedMember,omitempty"`
	Overdue        bool             `json:"overdue"`
}

type taskRoleJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type taskRequest struct {
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	DueDate          *string `json:"dueDate"`
	Status           string  `json:"status"`
	Recurrence       string  `json:"recurrence"`
	AssigneeMemberID *string `json:"assigneeMemberId"`
	AssigneeRoleID   *string `json:"assigneeRoleId"`
	LinkedMemberID   *string `json:"linkedMemberId"`
}

func toTaskJSON(t store.Task, now time.Time) taskJSON {
	j := taskJSON{ID: t.ID, Title: t.Title, Description: t.Description, Status: t.Status, Recurrence: t.Recurrence}
	if t.DueDate != nil {
		d := t.DueDate.Format(dateLayout)
		j.DueDate = &d
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		j.Overdue = t.Status != store.TaskDone && t.DueDate.Before(today)
	}
	if t.AssigneeMember != nil {
		m := memberRefJSON(*t.AssigneeMember)
		j.AssigneeMember = &m
	}
	if t.AssigneeRole != nil {
		j.AssigneeRole = &taskRoleJSON{ID: t.AssigneeRole.ID, Name: t.AssigneeRole.Name}
	}
	if t.LinkedMember != nil {
		m := memberRefJSON(*t.LinkedMember)
		j.LinkedMember = &m
	}
	return j
}

// taskAccess checks read access, or for writes the board roles (everyone but
// read-only users) on a device that may change member data.
func (a *API) taskAccess(w http.ResponseWriter, r *http.Request, write bool) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	allowed := ok
	if write {
		allowed = ok && claims.AppRole != openapi.READONLY && a.collab.CanWrite(r.Context(), db.ScopeMembersCore)
	} else {
		allowed = ok && a.collab.CanRead(r.Context(), db.ScopeMembersCore)
	}
	if !allowed {
		code := "FORBIDDEN"
		writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not access tasks"})
	}
	return allowed
}

// ListTasks: GET /api/v2/tasks?filter=all|mine|overdue|done.
func (a *API) ListTasks(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.taskAccess(w, r, false) {
		return
	}
	filter := r.URL.Query().Get("filter")
	switch filter {
	case "":
		filter = store.TaskFilterAll
	case store.TaskFilterAll, store.TaskFilterMine, store.TaskFilterOverdue, store.TaskFilterDone:
	default:
		badRequest(w, "filter must be all, mine, overdue or done")
		return
	}
	now := time.Now().UTC()
	claims, _ := auth.ClaimsFromContext(r.Context())
	list, err := st.ListTasks(r.Context(), filter, claims.Email, now)
	if err != nil {
		writeStoreError(w, err, "", "failed to list tasks")
		return
	}
	out := make([]taskJSON, 0, len(list))
	for _, t := range list {
		out = append(out, toTaskJSON(t, now))
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeTask(w http.ResponseWriter, r *http.Request) (store.TaskInput, bool) {
	var req taskRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.TaskInput{}, false
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Status == "" {
		req.Status = store.TaskOpen
	}
	if req.Recurrence == "" {
		req.Recurrence = store.RecurNone
	}
	empty := func(p **string) {
		if *p != nil && **p == "" {
			*p = nil
		}
	}
	empty(&req.AssigneeMemberID)
	empty(&req.AssigneeRoleID)
	empty(&req.LinkedMemberID)
	empty(&req.DueDate)
	in := store.TaskInput{
		Title: req.Title, Description: req.Description, Status: req.Status, Recurrence: req.Recurrence,
		AssigneeMemberID: req.AssigneeMemberID, AssigneeRoleID: req.AssigneeRoleID, LinkedMemberID: req.LinkedMemberID,
	}
	switch {
	case req.Title == "" || len(req.Title) > 200:
		badRequest(w, "title is required (at most 200 characters)")
	case req.Description != nil && len(*req.Description) > 4000:
		badRequest(w, "description is too long")
	case !store.ValidTaskStatus(req.Status):
		badRequest(w, "status must be open, in_progress or done")
	case !store.ValidRecurrence(req.Recurrence):
		badRequest(w, "recurrence must be none, weekly, monthly or yearly")
	case req.AssigneeMemberID != nil && req.AssigneeRoleID != nil:
		badRequest(w, "a task is assigned to a member or a role, not both")
	case req.Recurrence != store.RecurNone && req.DueDate == nil:
		badRequest(w, "recurring tasks need a due date")
	default:
		if req.DueDate != nil {
			d, err := time.Parse(dateLayout, *req.DueDate)
			if err != nil {
				badRequest(w, "dueDate must be a date (YYYY-MM-DD)")
				return in, false
			}
			in.DueDate = &d
		}
		return in, true
	}
	return in, false
}

// CreateTask: POST /api/v2/tasks.
func (a *API) CreateTask(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.taskAccess(w, r, true) {
		return
	}
	in, ok := decodeTask(w, r)
	if !ok {
		return
	}
	t, err := st.CreateTask(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to create task")
		return
	}
	st.Audit(r.Context(), "CREATE", "Task", t.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toTaskJSON(t, time.Now().UTC()))
}

// UpdateTask: PUT /api/v2/tasks/{id}.
func (a *API) UpdateTask(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.taskAccess(w, r, true) {
		return
	}
	in, ok := decodeTask(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	t, err := st.UpdateTask(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "task not found", "failed to update task")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Task", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toTaskJSON(t, time.Now().UTC()))
}

// DeleteTask: DELETE /api/v2/tasks/{id}.
func (a *API) DeleteTask(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.taskAccess(w, r, true) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteTask(r.Context(), id); err != nil {
		writeStoreError(w, err, "task not found", "failed to delete task")
		return
	}
	st.Audit(r.Context(), "DELETE", "Task", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}
