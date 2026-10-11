package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

func (a *API) ListInventoryItems(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	items, err := st.ListItems(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list inventory items")
		return
	}
	out := make([]openapi.InventoryItem, len(items))
	for i, it := range items {
		out[i] = toAPIItem(it)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) GetInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	item, err := st.GetItem(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to load inventory item")
		return
	}
	writeJSON(w, http.StatusOK, toAPIItem(item))
}

func (a *API) CreateInventoryItem(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.InventoryItemCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	item, err := st.CreateItem(r.Context(), store.ItemInput{
		Name: req.Name, Description: req.Description, SerialNumber: req.SerialNumber, Category: req.Category,
		Quantity: req.Quantity, Location: req.Location, PurchaseDate: req.PurchaseDate,
		PurchasePrice: float64Ptr(req.PurchasePrice),
	})
	if err != nil {
		writeStoreError(w, err, "", "failed to create inventory item")
		return
	}
	st.Audit(r.Context(), "CREATE", "InventoryItem", item.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPIItem(item))
}

func (a *API) UpdateInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.InventoryItemUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := st.GetItem(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to load inventory item")
		return
	}

	in := store.ItemInput{
		Name: existing.Name, Description: existing.Description, SerialNumber: existing.SerialNumber,
		Category: existing.Category, Quantity: existing.Quantity, Location: existing.Location,
		PurchaseDate: existing.PurchaseDate, PurchasePrice: existing.PurchasePrice,
	}
	if req.Name != nil {
		in.Name = *req.Name
	}
	if req.Description != nil {
		in.Description = req.Description
	}
	if req.SerialNumber != nil {
		in.SerialNumber = req.SerialNumber
	}
	if req.Category != nil {
		in.Category = *req.Category
	}
	if req.Quantity != nil {
		in.Quantity = *req.Quantity
	}
	if req.Location != nil {
		in.Location = *req.Location
	}
	if req.PurchaseDate != nil {
		in.PurchaseDate = req.PurchaseDate
	}
	if req.PurchasePrice != nil {
		in.PurchasePrice = float64Ptr(req.PurchasePrice)
	}

	item, err := st.UpdateItem(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to update inventory item")
		return
	}
	st.Audit(r.Context(), "UPDATE", "InventoryItem", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIItem(item))
}

func (a *API) DeleteInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteItem(r.Context(), id); err != nil {
		writeStoreError(w, err, "inventory item not found", "failed to delete inventory item")
		return
	}
	st.Audit(r.Context(), "DELETE", "InventoryItem", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func (a *API) ListInventoryLoans(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	loans, err := st.ListLoans(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list inventory loans")
		return
	}
	out := make([]openapi.InventoryLoan, len(loans))
	for i, l := range loans {
		out[i] = toAPILoan(l)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) GetInventoryLoan(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	loan, err := st.GetLoan(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "inventory loan not found", "failed to load inventory loan")
		return
	}
	writeJSON(w, http.StatusOK, toAPILoan(loan))
}

func (a *API) CreateInventoryLoan(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.InventoryLoanCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	loan, err := st.CreateLoan(r.Context(), store.LoanInput{
		ItemID: req.ItemId, MemberID: req.MemberId, LoanedAt: req.LoanedAt, DueDate: req.DueDate, Notes: req.Notes,
	})
	if err != nil {
		writeStoreError(w, err, "", "failed to create inventory loan")
		return
	}
	st.Audit(r.Context(), "CREATE", "InventoryLoan", loan.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPILoan(loan))
}

func (a *API) UpdateInventoryLoan(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.InventoryLoanUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := st.GetLoan(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "inventory loan not found", "failed to load inventory loan")
		return
	}

	in := store.LoanInput{
		ItemID: existing.ItemID, MemberID: existing.MemberID, LoanedAt: existing.LoanedAt,
		DueDate: existing.DueDate, ReturnedAt: existing.ReturnedAt, Notes: existing.Notes,
	}
	if req.ItemId != nil {
		in.ItemID = *req.ItemId
	}
	if req.MemberId != nil {
		in.MemberID = *req.MemberId
	}
	if req.LoanedAt != nil {
		in.LoanedAt = *req.LoanedAt
	}
	if req.DueDate != nil {
		in.DueDate = req.DueDate
	}
	if req.ReturnedAt != nil {
		in.ReturnedAt = req.ReturnedAt
	}
	if req.Notes != nil {
		in.Notes = *req.Notes
	}

	loan, err := st.UpdateLoan(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "inventory loan not found", "failed to update inventory loan")
		return
	}
	st.Audit(r.Context(), "UPDATE", "InventoryLoan", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPILoan(loan))
}

func (a *API) DeleteInventoryLoan(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteLoan(r.Context(), id); err != nil {
		writeStoreError(w, err, "inventory loan not found", "failed to delete inventory loan")
		return
	}
	st.Audit(r.Context(), "DELETE", "InventoryLoan", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── mapping ───────────────────────────────────────────────────────────────────

func toAPIItem(it store.Item) openapi.InventoryItem {
	return openapi.InventoryItem{
		Id: it.ID, Name: it.Name, Description: it.Description, SerialNumber: it.SerialNumber,
		Category: it.Category, Quantity: it.Quantity, Location: it.Location, PurchaseDate: it.PurchaseDate,
		PurchasePrice: float32Ptr(it.PurchasePrice), CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
}

func toAPILoan(l store.Loan) openapi.InventoryLoan {
	return openapi.InventoryLoan{
		Id: l.ID, ItemId: l.ItemID, MemberId: l.MemberID, LoanedAt: l.LoanedAt, DueDate: l.DueDate,
		ReturnedAt: l.ReturnedAt, Notes: l.Notes, CreatedAt: l.CreatedAt,
	}
}

func float64Ptr(v *float32) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}

func float32Ptr(v *float64) *float32 {
	if v == nil {
		return nil
	}
	f := float32(*v)
	return &f
}
