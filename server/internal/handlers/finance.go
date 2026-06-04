package handlers

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// ── Finance Transactions ──────────────────────────────────────────────────────

func (a *API) ListFinanceTransactions(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT "id","date","description","amount","type","category","notes","createdAt","updatedAt" FROM "FinanceTransaction" ORDER BY "date" DESC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list transactions"})
		return
	}
	defer rows.Close()

	result := []openapi.FinanceTransaction{}
	for rows.Next() {
		t, err := scanFinanceTransaction(rows)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan transaction"})
			return
		}
		result = append(result, t)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) CreateFinanceTransaction(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.FinanceTransactionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "FinanceTransaction" ("date","description","amount","type","category","notes","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?)`,
		req.Date.UTC().Format(time.RFC3339), req.Description, req.Amount, string(req.Type),
		optStr(req.Category), optStr(req.Notes), now, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create transaction"})
		return
	}
	id64, _ := res.LastInsertId()
	dbpkg.LogAudit(a.getDB(), "CREATE", "FinanceTransaction", int(id64), currentUserID(r), "")

	t, _ := getFinanceTransactionByID(a.getDB(), int(id64))
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) UpdateFinanceTransaction(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.FinanceTransactionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := getFinanceTransactionByID(a.getDB(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "transaction not found"})
		return
	}

	// Apply partial updates
	date := existing.Date
	if req.Date != nil {
		date = *req.Date
	}
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}
	amount := existing.Amount
	if req.Amount != nil {
		amount = *req.Amount
	}
	txType := string(existing.Type)
	if req.Type != nil {
		txType = string(*req.Type)
	}
	category := existing.Category
	if req.Category != nil {
		category = req.Category
	}
	notes := existing.Notes
	if req.Notes != nil {
		notes = req.Notes
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`UPDATE "FinanceTransaction" SET "date"=?,"description"=?,"amount"=?,"type"=?,"category"=?,"notes"=?,"updatedAt"=? WHERE "id"=?`,
		date.UTC().Format(time.RFC3339), description, amount, txType,
		category, notes, now, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update transaction"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "FinanceTransaction", id, currentUserID(r), "")

	t, _ := getFinanceTransactionByID(a.getDB(), id)
	writeJSON(w, http.StatusOK, t)
}

func (a *API) DeleteFinanceTransaction(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	res, err := a.getDB().Exec(`DELETE FROM "FinanceTransaction" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete transaction"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "transaction not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "FinanceTransaction", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── Member Fees ───────────────────────────────────────────────────────────────

func (a *API) ListMemberFees(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT f."id",f."memberId",f."amount",f."dueDate",f."paidDate",f."description",f."year",f."createdAt",f."updatedAt",
			m."id",m."number",m."firstName",m."lastName"
		FROM "MemberFee" f
		JOIN "Member" m ON m."id"=f."memberId"
		ORDER BY f."year" DESC, f."dueDate" DESC`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list member fees"})
		return
	}
	defer rows.Close()

	result := []openapi.MemberFee{}
	for rows.Next() {
		f, err := scanMemberFee(rows)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to scan fee"})
			return
		}
		result = append(result, f)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) CreateMemberFee(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.MemberFeeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.getDB().Exec(
		`INSERT INTO "MemberFee" ("memberId","amount","dueDate","paidDate","description","year","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?)`,
		req.MemberId, req.Amount, req.DueDate.UTC().Format(time.RFC3339),
		optTime(req.PaidDate), optStr(req.Description), req.Year, now, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to create member fee"})
		return
	}
	id64, _ := res.LastInsertId()
	dbpkg.LogAudit(a.getDB(), "CREATE", "MemberFee", int(id64), currentUserID(r), "")

	fee, _ := getMemberFeeByID(a.getDB(), int(id64))
	writeJSON(w, http.StatusCreated, fee)
}

func (a *API) UpdateMemberFee(w http.ResponseWriter, r *http.Request, id int) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.MemberFeeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := getMemberFeeByID(a.getDB(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "member fee not found"})
		return
	}

	memberID := existing.MemberId
	if req.MemberId != nil {
		memberID = *req.MemberId
	}
	amount := existing.Amount
	if req.Amount != nil {
		amount = *req.Amount
	}
	dueDate := existing.DueDate
	if req.DueDate != nil {
		dueDate = *req.DueDate
	}
	year := existing.Year
	if req.Year != nil {
		year = *req.Year
	}
	paidDate := existing.PaidDate
	if req.PaidDate != nil {
		paidDate = req.PaidDate
	}
	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`UPDATE "MemberFee" SET "memberId"=?,"amount"=?,"dueDate"=?,"paidDate"=?,"description"=?,"year"=?,"updatedAt"=? WHERE "id"=?`,
		memberID, amount, dueDate.UTC().Format(time.RFC3339), optTime(paidDate), optStr(description), year, now, id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update member fee"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "UPDATE", "MemberFee", id, currentUserID(r), "")

	fee, _ := getMemberFeeByID(a.getDB(), id)
	writeJSON(w, http.StatusOK, fee)
}

// DeleteMemberFee (DELETE /finance/reset) removes a single fee by id passed via query param.
func (a *API) DeleteMemberFee(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "id query parameter required"})
		return
	}
	var id int
	if err := json.Unmarshal([]byte(idStr), &id); err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid id"})
		return
	}
	res, err := a.getDB().Exec(`DELETE FROM "MemberFee" WHERE "id"=?`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to delete member fee"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "member fee not found"})
		return
	}
	dbpkg.LogAudit(a.getDB(), "DELETE", "MemberFee", id, currentUserID(r), "")
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ResetFinanceLedger deletes all transactions and fees when confirmation is provided.
func (a *API) ResetFinanceLedger(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.FinanceResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	if req.Confirmation != "RESET" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "confirmation must be 'RESET'"})
		return
	}

	var deletedFees, deletedTx int
	row := a.getDB().QueryRow(`SELECT COUNT(*) FROM "MemberFee"`)
	_ = row.Scan(&deletedFees)
	row = a.getDB().QueryRow(`SELECT COUNT(*) FROM "FinanceTransaction"`)
	_ = row.Scan(&deletedTx)

	_, _ = a.getDB().Exec(`DELETE FROM "MemberFee"`)
	_, _ = a.getDB().Exec(`DELETE FROM "FinanceTransaction"`)

	writeJSON(w, http.StatusOK, openapi.FinanceResetResponse{
		DeletedMemberFees:   deletedFees,
		DeletedTransactions: deletedTx,
	})
}

// ── CAMT.053 Import ───────────────────────────────────────────────────────────

func (a *API) ImportFinanceTransactionsCamt053(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.Camt053ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	entries, err := parseCamt053(req.Xml)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "failed to parse CAMT.053 XML: " + err.Error()})
		return
	}

	imported := 0
	skipped := 0
	now := time.Now().UTC().Format(time.RFC3339)

	for _, e := range entries {
		txType := "income"
		if e.Amount < 0 {
			txType = "expense"
			e.Amount = -e.Amount
		}

		_, err := a.getDB().Exec(
			`INSERT INTO "FinanceTransaction" ("date","description","amount","type","category","notes","createdAt","updatedAt") VALUES (?,?,?,?,?,?,?,?)`,
			e.Date.UTC().Format(time.RFC3339), e.Description, e.Amount, txType, nil, nil, now, now,
		)
		if err != nil {
			skipped++
			continue
		}
		imported++
	}

	total := imported + skipped
	writeJSON(w, http.StatusOK, openapi.Camt053ImportResponse{
		TotalCount:           total,
		ImportedCount:        imported,
		SkippedCount:         skipped,
		MatchedMemberCount:   0,
		MemberFeesCreated:    0,
		MemberFeesMarkedPaid: 0,
	})
}

// camt053Entry is a minimal parsed bank statement entry.
type camt053Entry struct {
	Date        time.Time
	Amount      float32
	Description string
}

// parseCamt053 is a best-effort parser for the CAMT.053 bank statement format.
func parseCamt053(xmlData string) ([]camt053Entry, error) {
	type Amount struct {
		Value    float32 `xml:",chardata"`
		Currency string  `xml:"Ccy,attr"`
		CdtDbtInd string
	}
	type Ntry struct {
		Amt       Amount `xml:"Amt"`
		CdtDbtInd string `xml:"CdtDbtInd"`
		BookgDt   struct {
			Dt string `xml:"Dt"`
		} `xml:"BookgDt"`
		NtryDtls struct {
			TxDtls struct {
				RmtInf struct {
					Ustrd string `xml:"Ustrd"`
				} `xml:"RmtInf"`
				AddtlTxInf string `xml:"AddtlTxInf"`
			} `xml:"TxDtls"`
		} `xml:"NtryDtls"`
		AddtlNtryInf string `xml:"AddtlNtryInf"`
	}
	type Document struct {
		Entries []Ntry `xml:"BkToCstmrStmt>Stmt>Ntry"`
	}

	var doc Document
	if err := xml.Unmarshal([]byte(xmlData), &doc); err != nil {
		return nil, err
	}

	var results []camt053Entry
	for _, e := range doc.Entries {
		date, _ := time.Parse("2006-01-02", e.BookgDt.Dt)
		description := strings.TrimSpace(e.NtryDtls.TxDtls.RmtInf.Ustrd)
		if description == "" {
			description = strings.TrimSpace(e.NtryDtls.TxDtls.AddtlTxInf)
		}
		if description == "" {
			description = strings.TrimSpace(e.AddtlNtryInf)
		}
		amount := e.Amt.Value
		if e.CdtDbtInd == "DBIT" {
			amount = -amount
		}
		results = append(results, camt053Entry{Date: date, Amount: amount, Description: description})
	}
	return results, nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

func getFinanceTransactionByID(db *sql.DB, id int) (openapi.FinanceTransaction, error) {
	row := db.QueryRow(
		`SELECT "id","date","description","amount","type","category","notes","createdAt","updatedAt" FROM "FinanceTransaction" WHERE "id"=?`, id,
	)
	return scanFinanceTransactionRow(row)
}

type financeRows interface {
	Scan(dest ...interface{}) error
}

func scanFinanceTransaction(rows *sql.Rows) (openapi.FinanceTransaction, error) {
	var t openapi.FinanceTransaction
	var dateStr, createdAtStr, updatedAtStr string
	var category, notes sql.NullString

	err := rows.Scan(&t.Id, &dateStr, &t.Description, &t.Amount, (*string)(&t.Type), &category, &notes, &createdAtStr, &updatedAtStr)
	if err != nil {
		return t, err
	}
	t.Category = nullStrPtr(category)
	t.Notes = nullStrPtr(notes)
	t.Date, _ = parseTime(dateStr)
	t.CreatedAt, _ = parseTime(createdAtStr)
	t.UpdatedAt, _ = parseTime(updatedAtStr)
	return t, nil
}

func scanFinanceTransactionRow(row *sql.Row) (openapi.FinanceTransaction, error) {
	var t openapi.FinanceTransaction
	var dateStr, createdAtStr, updatedAtStr string
	var category, notes sql.NullString

	err := row.Scan(&t.Id, &dateStr, &t.Description, &t.Amount, (*string)(&t.Type), &category, &notes, &createdAtStr, &updatedAtStr)
	if err != nil {
		return t, err
	}
	t.Category = nullStrPtr(category)
	t.Notes = nullStrPtr(notes)
	t.Date, _ = parseTime(dateStr)
	t.CreatedAt, _ = parseTime(createdAtStr)
	t.UpdatedAt, _ = parseTime(updatedAtStr)
	return t, nil
}

func getMemberFeeByID(db *sql.DB, id int) (openapi.MemberFee, error) {
	row := db.QueryRow(
		`SELECT f."id",f."memberId",f."amount",f."dueDate",f."paidDate",f."description",f."year",f."createdAt",f."updatedAt",
			m."id",m."number",m."firstName",m."lastName"
		FROM "MemberFee" f
		JOIN "Member" m ON m."id"=f."memberId"
		WHERE f."id"=?`, id,
	)
	return scanMemberFeeRow(row)
}

func scanMemberFee(rows *sql.Rows) (openapi.MemberFee, error) {
	var f openapi.MemberFee
	var dueDateStr, createdAtStr, updatedAtStr string
	var paidDate, description sql.NullString

	err := rows.Scan(
		&f.Id, &f.MemberId, &f.Amount, &dueDateStr, &paidDate, &description, &f.Year, &createdAtStr, &updatedAtStr,
		&f.Member.Id, &f.Member.Number, &f.Member.FirstName, &f.Member.LastName,
	)
	if err != nil {
		return f, err
	}
	f.Description = nullStrPtr(description)
	f.PaidDate = parseOptTime(paidDate)
	f.DueDate, _ = parseTime(dueDateStr)
	f.CreatedAt, _ = parseTime(createdAtStr)
	f.UpdatedAt, _ = parseTime(updatedAtStr)
	return f, nil
}

func scanMemberFeeRow(row *sql.Row) (openapi.MemberFee, error) {
	var f openapi.MemberFee
	var dueDateStr, createdAtStr, updatedAtStr string
	var paidDate, description sql.NullString

	err := row.Scan(
		&f.Id, &f.MemberId, &f.Amount, &dueDateStr, &paidDate, &description, &f.Year, &createdAtStr, &updatedAtStr,
		&f.Member.Id, &f.Member.Number, &f.Member.FirstName, &f.Member.LastName,
	)
	if err != nil {
		return f, err
	}
	f.Description = nullStrPtr(description)
	f.PaidDate = parseOptTime(paidDate)
	f.DueDate, _ = parseTime(dueDateStr)
	f.CreatedAt, _ = parseTime(createdAtStr)
	f.UpdatedAt, _ = parseTime(updatedAtStr)
	return f, nil
}

func parseTime(s string) (time.Time, error) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}

