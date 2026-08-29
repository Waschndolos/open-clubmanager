package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

func (a *API) ListInventoryItems(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT "id","name","description","serialNumber","category","quantity","location","purchaseDate","purchasePrice","createdAt","updatedAt" FROM "InventoryItem" ORDER BY "name" ASC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list inventory items"})
		return
	}
	defer rows.Close()

	result := []openapi.InventoryItem{}
	for rows.Next() {
		item, scanErr := scanInventoryItem(rows)
		if scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan inventory item"})
			return
		}
		result = append(result, item)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) GetInventoryItem(w http.ResponseWriter, _ *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	item, err := getInventoryItemByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory item not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load inventory item"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) CreateInventoryItem(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.InventoryItemCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "InventoryItem" ("name","description","serialNumber","category","quantity","location","purchaseDate","purchasePrice","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?,?,?)`,
		req.Name, optStr(req.Description), optStr(req.SerialNumber), req.Category, req.Quantity, req.Location,
		optTime(req.PurchaseDate), optFloat(req.PurchasePrice), now, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create inventory item"})
		return
	}
	id64, err := res.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create inventory item"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "CREATE", "InventoryItem", int(id64), currentUserID(r), "")

	item, err := getInventoryItemByID(a.getDB(), int(id64))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load created inventory item"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) UpdateInventoryItem(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.InventoryItemUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := getInventoryItemByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory item not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load inventory item"})
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}
	serialNumber := existing.SerialNumber
	if req.SerialNumber != nil {
		serialNumber = req.SerialNumber
	}
	category := existing.Category
	if req.Category != nil {
		category = *req.Category
	}
	quantity := existing.Quantity
	if req.Quantity != nil {
		quantity = *req.Quantity
	}
	location := existing.Location
	if req.Location != nil {
		location = *req.Location
	}
	purchaseDate := existing.PurchaseDate
	if req.PurchaseDate != nil {
		purchaseDate = req.PurchaseDate
	}
	purchasePrice := existing.PurchasePrice
	if req.PurchasePrice != nil {
		purchasePrice = req.PurchasePrice
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`UPDATE "InventoryItem" SET "name"=?,"description"=?,"serialNumber"=?,"category"=?,"quantity"=?,"location"=?,"purchaseDate"=?,"purchasePrice"=?,"updatedAt"=? WHERE "id"=?`,
		name, optStr(description), optStr(serialNumber), category, quantity, location, optTime(purchaseDate), optFloat(purchasePrice), now, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update inventory item"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "InventoryItem", id, currentUserID(r), "")

	item, err := getInventoryItemByID(a.getDB(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load updated inventory item"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) DeleteInventoryItem(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	res, err := a.getDB().Exec(`DELETE FROM "InventoryItem" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete inventory item"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory item not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "InventoryItem", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func (a *API) ListInventoryLoans(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT "id","itemId","memberId","loanedAt","dueDate","returnedAt","notes","createdAt" FROM "InventoryLoan" ORDER BY "returnedAt" IS NOT NULL, "loanedAt" DESC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list inventory loans"})
		return
	}
	defer rows.Close()

	result := []openapi.InventoryLoan{}
	for rows.Next() {
		loan, scanErr := scanInventoryLoan(rows)
		if scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan inventory loan"})
			return
		}
		result = append(result, loan)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) GetInventoryLoan(w http.ResponseWriter, _ *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	loan, err := getInventoryLoanByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory loan not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load inventory loan"})
		return
	}
	writeJSON(w, http.StatusOK, loan)
}

func (a *API) CreateInventoryLoan(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.InventoryLoanCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "InventoryLoan" ("itemId","memberId","loanedAt","dueDate","returnedAt","notes","createdAt") VALUES (?,?,?,?,?,?,?)`,
		req.ItemId, req.MemberId, req.LoanedAt.UTC().Format(time.RFC3339),
		optTime(req.DueDate), nil, req.Notes, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create inventory loan"})
		return
	}
	id64, err := res.LastInsertId()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create inventory loan"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "CREATE", "InventoryLoan", int(id64), currentUserID(r), "")

	loan, err := getInventoryLoanByID(a.getDB(), int(id64))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load created inventory loan"})
		return
	}
	writeJSON(w, http.StatusCreated, loan)
}

func (a *API) UpdateInventoryLoan(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.InventoryLoanUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := getInventoryLoanByID(a.getDB(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory loan not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load inventory loan"})
		return
	}

	itemID := existing.ItemId
	if req.ItemId != nil {
		itemID = *req.ItemId
	}
	memberID := existing.MemberId
	if req.MemberId != nil {
		memberID = *req.MemberId
	}
	loanedAt := existing.LoanedAt
	if req.LoanedAt != nil {
		loanedAt = *req.LoanedAt
	}
	dueDate := existing.DueDate
	if req.DueDate != nil {
		dueDate = req.DueDate
	}
	returnedAt := existing.ReturnedAt
	if req.ReturnedAt != nil {
		returnedAt = req.ReturnedAt
	}
	notes := existing.Notes
	if req.Notes != nil {
		notes = *req.Notes
	}

	_, err = a.getDB().Exec(
		`UPDATE "InventoryLoan" SET "itemId"=?,"memberId"=?,"loanedAt"=?,"dueDate"=?,"returnedAt"=?,"notes"=? WHERE "id"=?`,
		itemID, memberID, loanedAt.UTC().Format(time.RFC3339), optTime(dueDate), optTime(returnedAt), notes, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update inventory loan"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "InventoryLoan", id, currentUserID(r), "")

	loan, err := getInventoryLoanByID(a.getDB(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to load updated inventory loan"})
		return
	}
	writeJSON(w, http.StatusOK, loan)
}

func (a *API) DeleteInventoryLoan(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	res, err := a.getDB().Exec(`DELETE FROM "InventoryLoan" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete inventory loan"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "inventory loan not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "InventoryLoan", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

func getInventoryItemByID(db *sql.DB, id int) (openapi.InventoryItem, error) {
	row := db.QueryRow(
		`SELECT "id","name","description","serialNumber","category","quantity","location","purchaseDate","purchasePrice","createdAt","updatedAt" FROM "InventoryItem" WHERE "id"=?`,
		id,
	)
	return scanInventoryItemRow(row)
}

func scanInventoryItem(rows *sql.Rows) (openapi.InventoryItem, error) {
	var item openapi.InventoryItem
	var description, serialNumber, purchaseDate sql.NullString
	var purchasePrice sql.NullFloat64
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&item.Id, &item.Name, &description, &serialNumber, &item.Category, &item.Quantity, &item.Location,
		&purchaseDate, &purchasePrice, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return item, err
	}
	item.Description = nullStrPtr(description)
	item.SerialNumber = nullStrPtr(serialNumber)
	item.PurchaseDate = parseOptTime(purchaseDate)
	item.PurchasePrice = nullFloat32Ptr(purchasePrice)
	item.CreatedAt, _ = parseTime(createdAtStr)
	item.UpdatedAt, _ = parseTime(updatedAtStr)
	return item, nil
}

func scanInventoryItemRow(row *sql.Row) (openapi.InventoryItem, error) {
	var item openapi.InventoryItem
	var description, serialNumber, purchaseDate sql.NullString
	var purchasePrice sql.NullFloat64
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&item.Id, &item.Name, &description, &serialNumber, &item.Category, &item.Quantity, &item.Location,
		&purchaseDate, &purchasePrice, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return item, err
	}
	item.Description = nullStrPtr(description)
	item.SerialNumber = nullStrPtr(serialNumber)
	item.PurchaseDate = parseOptTime(purchaseDate)
	item.PurchasePrice = nullFloat32Ptr(purchasePrice)
	item.CreatedAt, _ = parseTime(createdAtStr)
	item.UpdatedAt, _ = parseTime(updatedAtStr)
	return item, nil
}

func getInventoryLoanByID(db *sql.DB, id int) (openapi.InventoryLoan, error) {
	row := db.QueryRow(
		`SELECT "id","itemId","memberId","loanedAt","dueDate","returnedAt","notes","createdAt" FROM "InventoryLoan" WHERE "id"=?`,
		id,
	)
	return scanInventoryLoanRow(row)
}

func scanInventoryLoan(rows *sql.Rows) (openapi.InventoryLoan, error) {
	var loan openapi.InventoryLoan
	var loanedAtStr, createdAtStr string
	var dueDate, returnedAt sql.NullString

	err := rows.Scan(
		&loan.Id, &loan.ItemId, &loan.MemberId, &loanedAtStr, &dueDate, &returnedAt, &loan.Notes, &createdAtStr,
	)
	if err != nil {
		return loan, err
	}
	loan.LoanedAt, _ = parseTime(loanedAtStr)
	loan.DueDate = parseOptTime(dueDate)
	loan.ReturnedAt = parseOptTime(returnedAt)
	loan.CreatedAt, _ = parseTime(createdAtStr)
	return loan, nil
}

func scanInventoryLoanRow(row *sql.Row) (openapi.InventoryLoan, error) {
	var loan openapi.InventoryLoan
	var loanedAtStr, createdAtStr string
	var dueDate, returnedAt sql.NullString

	err := row.Scan(
		&loan.Id, &loan.ItemId, &loan.MemberId, &loanedAtStr, &dueDate, &returnedAt, &loan.Notes, &createdAtStr,
	)
	if err != nil {
		return loan, err
	}
	loan.LoanedAt, _ = parseTime(loanedAtStr)
	loan.DueDate = parseOptTime(dueDate)
	loan.ReturnedAt = parseOptTime(returnedAt)
	loan.CreatedAt, _ = parseTime(createdAtStr)
	return loan, nil
}

func nullFloat32Ptr(nf sql.NullFloat64) *float32 {
	if !nf.Valid {
		return nil
	}
	val := float32(nf.Float64)
	return &val
}

func optFloat(v *float32) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
