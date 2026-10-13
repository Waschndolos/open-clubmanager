package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

const defaultMaintenanceWarnDays = 30

type itemStatusJSON struct {
	Condition      string  `json:"condition"`
	IntervalMonths *int    `json:"intervalMonths,omitempty"`
	NextDue        *string `json:"nextDue,omitempty"`
}

type maintenanceStateJSON struct {
	ItemID   string `json:"itemId"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	itemStatusJSON
	LastMaintenance *string `json:"lastMaintenance,omitempty"`
	// Due is none, ok, due or overdue.
	Due string `json:"due"`
}

type maintenanceJSON struct {
	ID            string   `json:"id"`
	ItemID        string   `json:"itemId"`
	Date          string   `json:"date"`
	PerformedBy   *string  `json:"performedBy,omitempty"`
	Note          *string  `json:"note,omitempty"`
	Cost          *float64 `json:"cost,omitempty"`
	TransactionID *string  `json:"transactionId,omitempty"`
}

type maintenanceRequest struct {
	Date        string   `json:"date"`
	PerformedBy *string  `json:"performedBy"`
	Note        *string  `json:"note"`
	Cost        *float64 `json:"cost"`
	Condition   *string  `json:"condition"`
	// BookExpense books the cost as an expense with this description and
	// category in the cash book.
	BookExpense *struct {
		Description string  `json:"description"`
		Category    *string `json:"category"`
	} `json:"bookExpense"`
}

func toMaintenanceJSON(m store.Maintenance) maintenanceJSON {
	return maintenanceJSON{ID: m.ID, ItemID: m.ItemID, Date: m.Date.Format(dateLayout), PerformedBy: m.PerformedBy,
		Note: m.Note, Cost: m.Cost, TransactionID: m.TransactionID}
}

// ListMaintenanceStates returns all items with condition, schedule and due
// state: GET /api/v2/inventory/maintenance?warnDays=30.
func (a *API) ListMaintenanceStates(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "inventory", db.ScopeInventory) {
		return
	}
	warn := defaultMaintenanceWarnDays
	if v := r.URL.Query().Get("warnDays"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 3660 {
			badRequest(w, "warnDays must be a number between 0 and 3660")
			return
		}
		warn = n
	}
	list, err := st.MaintenanceStates(r.Context(), time.Now(), warn)
	if err != nil {
		writeStoreError(w, err, "", "failed to list maintenance")
		return
	}
	out := make([]maintenanceStateJSON, 0, len(list))
	for _, it := range list {
		out = append(out, maintenanceStateJSON{ItemID: it.ItemID, Name: it.Name, Category: it.Category, Location: it.Location,
			itemStatusJSON:  itemStatusJSON{Condition: it.Condition, IntervalMonths: it.IntervalMonths, NextDue: formatDate(it.NextDue)},
			LastMaintenance: formatDate(it.LastMaintenance), Due: it.Due})
	}
	writeJSON(w, http.StatusOK, out)
}

// SetItemStatus: PUT /api/v2/inventory/items/{id}/status.
func (a *API) SetItemStatus(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "inventory", db.ScopeInventory) {
		return
	}
	var req itemStatusJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	if !store.ValidCondition(req.Condition) {
		badRequest(w, "condition must be ready, damaged, in_repair or retired")
		return
	}
	if req.IntervalMonths != nil && (*req.IntervalMonths <= 0 || *req.IntervalMonths > 1200) {
		badRequest(w, "intervalMonths must be between 1 and 1200")
		return
	}
	in := store.ItemStatus{Condition: req.Condition, IntervalMonths: req.IntervalMonths}
	if req.NextDue != nil && *req.NextDue != "" {
		d, err := time.Parse(dateLayout, *req.NextDue)
		if err != nil {
			badRequest(w, "nextDue must be a date (YYYY-MM-DD)")
			return
		}
		in.NextDue = &d
	}
	id := chi.URLParam(r, "id")
	if err := st.SetItemStatus(r.Context(), id, in); err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to update item status")
		return
	}
	st.Audit(r.Context(), "UPDATE", "InventoryItemStatus", id, currentUserID(r))
	writeJSON(w, http.StatusOK, req)
}

// ListItemMaintenance: GET /api/v2/inventory/items/{id}/maintenance.
func (a *API) ListItemMaintenance(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "inventory", db.ScopeInventory) {
		return
	}
	list, err := st.ListMaintenance(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeStoreError(w, err, "", "failed to list maintenance")
		return
	}
	out := make([]maintenanceJSON, 0, len(list))
	for _, m := range list {
		out = append(out, toMaintenanceJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// RecordMaintenance: POST /api/v2/inventory/items/{id}/maintenance. Booking
// the cost as an expense needs the treasurer's rights on the cash book.
func (a *API) RecordMaintenance(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "inventory", db.ScopeInventory) {
		return
	}
	var req maintenanceRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	date, err := time.Parse(dateLayout, req.Date)
	if err != nil {
		badRequest(w, "date must be a date (YYYY-MM-DD)")
		return
	}
	in := store.MaintenanceInput{Date: date, Cost: req.Cost, Condition: req.Condition}
	var valid bool
	if in.PerformedBy, valid = trimmed(req.PerformedBy, 200); !valid {
		badRequest(w, "performedBy is too long")
		return
	}
	if in.Note, valid = trimmed(req.Note, 1000); !valid {
		badRequest(w, "note is too long")
		return
	}
	if in.Cost != nil && (*in.Cost < 0 || *in.Cost > 1e9) {
		badRequest(w, "cost must not be negative")
		return
	}
	if in.Condition != nil && !store.ValidCondition(*in.Condition) {
		badRequest(w, "condition must be ready, damaged, in_repair or retired")
		return
	}
	if req.BookExpense != nil && in.Cost != nil && *in.Cost > 0 {
		claims, _ := auth.ClaimsFromContext(r.Context())
		treasurer := claims != nil && (claims.AppRole == openapi.ADMIN || claims.AppRole == openapi.TREASURER)
		if !treasurer || !a.collab.CanWrite(r.Context(), db.ScopeFinance) {
			code := "FORBIDDEN"
			writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not book expenses"})
			return
		}
		desc, ok := trimmed(&req.BookExpense.Description, 200)
		if !ok || desc == nil {
			badRequest(w, "bookExpense.description is required (at most 200 characters)")
			return
		}
		category, _ := trimmed(req.BookExpense.Category, 100)
		in.Expense = &store.ExpenseInput{Description: *desc, Category: category}
	}
	itemID := chi.URLParam(r, "id")
	m, err := st.RecordMaintenance(r.Context(), itemID, in)
	if err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to record maintenance")
		return
	}
	st.Audit(r.Context(), "CREATE", "InventoryMaintenance", m.ID, currentUserID(r))
	if m.TransactionID != nil {
		st.Audit(r.Context(), "CREATE", "Transaction", *m.TransactionID, currentUserID(r))
	}
	writeJSON(w, http.StatusCreated, toMaintenanceJSON(m))
}

// DeleteMaintenance: DELETE /api/v2/inventory/maintenance/{id}.
func (a *API) DeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "inventory", db.ScopeInventory) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := st.DeleteMaintenance(r.Context(), id); err != nil {
		writeStoreError(w, err, "maintenance not found", "failed to delete maintenance")
		return
	}
	st.Audit(r.Context(), "DELETE", "InventoryMaintenance", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

// writeLoanError reports a loan of an item that is not ready as 409.
func writeLoanError(w http.ResponseWriter, err error, notFound, failure string) {
	if errors.Is(err, store.ErrItemNotReady) {
		code := "ITEM_NOT_READY"
		writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code, Error: "the item is not ready for use and cannot be lent"})
		return
	}
	writeStoreError(w, err, notFound, failure)
}
