package handlers

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/sepa"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// ── Finance Transactions ──────────────────────────────────────────────────────

func (a *API) ListFinanceTransactions(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeAccess(w, r, false) {
		return
	}
	list, err := st.ListTransactions(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list transactions")
		return
	}
	out := make([]openapi.FinanceTransaction, len(list))
	for i, t := range list {
		out[i] = toAPITransaction(t)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateFinanceTransaction(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.FinanceTransactionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Type.Valid() {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	t, err := st.CreateTransaction(r.Context(), store.TransactionInput{
		Date: req.Date, Description: req.Description, Amount: float64(req.Amount), Type: string(req.Type),
		Category: req.Category, Notes: req.Notes,
	})
	if err != nil {
		writeStoreError(w, err, "", "failed to create transaction")
		return
	}
	st.Audit(r.Context(), "CREATE", "FinanceTransaction", t.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPITransaction(t))
}

func (a *API) UpdateFinanceTransaction(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.FinanceTransactionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Type != nil && !req.Type.Valid()) {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := st.GetTransaction(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "transaction not found", "failed to load transaction")
		return
	}

	// Apply partial updates
	in := store.TransactionInput{
		Date: existing.Date, Description: existing.Description, Amount: existing.Amount, Type: existing.Type,
		Category: existing.Category, Notes: existing.Notes,
	}
	if req.Date != nil {
		in.Date = *req.Date
	}
	if req.Description != nil {
		in.Description = *req.Description
	}
	if req.Amount != nil {
		in.Amount = float64(*req.Amount)
	}
	if req.Type != nil {
		in.Type = string(*req.Type)
	}
	if req.Category != nil {
		in.Category = req.Category
	}
	if req.Notes != nil {
		in.Notes = req.Notes
	}

	t, err := st.UpdateTransaction(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "transaction not found", "failed to update transaction")
		return
	}
	st.Audit(r.Context(), "UPDATE", "FinanceTransaction", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPITransaction(t))
}

func (a *API) DeleteFinanceTransaction(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	if err := st.DeleteTransaction(r.Context(), id); err != nil {
		writeStoreError(w, err, "transaction not found", "failed to delete transaction")
		return
	}
	st.Audit(r.Context(), "DELETE", "FinanceTransaction", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── Member Fees ───────────────────────────────────────────────────────────────

func (a *API) ListMemberFees(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.financeAccess(w, r, false) {
		return
	}
	list, err := st.ListFees(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list member fees")
		return
	}
	out := make([]openapi.MemberFee, len(list))
	for i, f := range list {
		out[i] = toAPIFee(f)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateMemberFee(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.MemberFeeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}
	fee, err := st.CreateFee(r.Context(), store.FeeInput{
		MemberID: req.MemberId, Amount: float64(req.Amount), DueDate: req.DueDate, PaidDate: req.PaidDate,
		Description: req.Description, Year: req.Year,
	})
	if err != nil {
		writeStoreError(w, err, "", "failed to create member fee")
		return
	}
	st.Audit(r.Context(), "CREATE", "MemberFee", fee.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPIFee(fee))
}

func (a *API) UpdateMemberFee(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.MemberFeeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	existing, err := st.GetFee(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "member fee not found", "failed to load member fee")
		return
	}

	in := store.FeeInput{
		MemberID: existing.Member.ID, Amount: existing.Amount, DueDate: existing.DueDate, PaidDate: existing.PaidDate,
		Description: existing.Description, Year: existing.Year,
	}
	if req.MemberId != nil {
		in.MemberID = *req.MemberId
	}
	if req.Amount != nil {
		in.Amount = float64(*req.Amount)
	}
	if req.DueDate != nil {
		in.DueDate = *req.DueDate
	}
	if req.Year != nil {
		in.Year = *req.Year
	}
	if req.PaidDate != nil {
		in.PaidDate = req.PaidDate
	}
	if req.Description != nil {
		in.Description = req.Description
	}

	fee, err := st.UpdateFee(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "member fee not found", "failed to update member fee")
		return
	}
	st.Audit(r.Context(), "UPDATE", "MemberFee", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIFee(fee))
}

// DeleteMemberFee (DELETE /finance/reset) removes a single fee by id passed via query param.
func (a *API) DeleteMemberFee(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "id query parameter required"})
		return
	}
	if err := st.DeleteFee(r.Context(), id); err != nil {
		writeStoreError(w, err, "member fee not found", "failed to delete member fee")
		return
	}
	st.Audit(r.Context(), "DELETE", "MemberFee", id, currentUserID(r))
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ResetFinanceLedger deletes all transactions and fees when confirmation is provided.
func (a *API) ResetFinanceLedger(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
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

	fees, transactions, err := st.ResetFinance(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to reset finance ledger")
		return
	}
	writeJSON(w, http.StatusOK, openapi.FinanceResetResponse{
		DeletedMemberFees:   fees,
		DeletedTransactions: transactions,
	})
}

// ── CAMT.053 Import ───────────────────────────────────────────────────────────

// ImportFinanceTransactionsCamt053 creates a transaction per statement entry.
// On a device that may write bank data, the payer's name, IBAN and the
// references are kept for the payment reconciliation; matchedMemberCount
// counts the entries for which it has a suggestion. Nothing is assigned to
// fees here, the treasurer confirms that in the reconciliation.
func (a *API) ImportFinanceTransactionsCamt053(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
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

	keepBank := a.collab.CanWrite(r.Context(), db.ScopeMembersBank)
	imported := 0
	skipped := 0
	var ids []string

	for _, e := range entries {
		txType := "income"
		if e.Amount < 0 {
			txType = "expense"
			e.Amount = -e.Amount
		}
		var bank *store.BankEntry
		if keepBank {
			bank = &e.Bank
		}

		t, err := st.ImportTransaction(r.Context(), store.TransactionInput{
			Date: e.Date, Description: e.Description, Amount: e.Amount, Type: txType,
		}, bank)
		if err != nil {
			skipped++
			continue
		}
		st.Audit(r.Context(), "CREATE", "FinanceTransaction", t.ID, currentUserID(r))
		ids = append(ids, t.ID)
		imported++
	}

	matched := 0
	if keepBank && len(ids) > 0 {
		if s, err := reconciliationSuggestions(r.Context(), st); err == nil {
			for _, id := range ids {
				if s[id] != nil {
					matched++
				}
			}
		}
	}

	total := imported + skipped
	writeJSON(w, http.StatusOK, openapi.Camt053ImportResponse{
		TotalCount:           total,
		ImportedCount:        imported,
		SkippedCount:         skipped,
		MatchedMemberCount:   matched,
		MemberFeesCreated:    0,
		MemberFeesMarkedPaid: 0,
	})
}

// camt053Entry is a parsed bank statement entry. Amount is negative for debits.
type camt053Entry struct {
	Date        time.Time
	Amount      float64
	Description string
	Bank        store.BankEntry
}

// German bank transaction code (GVC) of a returned SEPA direct debit, as
// used in the proprietary BkTxCd of German banks (e.g. "NRTI+109+…").
var gvcReturnedDebit = regexp.MustCompile(`(^|\+)109(\+|$)`)

// parseCamt053 is a best-effort parser for the CAMT.053 bank statement
// format (versions .001.02 to .001.08). Of entries with several transactions
// (batch bookings) the first transaction supplies payer and references.
func parseCamt053(xmlData string) ([]camt053Entry, error) {
	type Party struct {
		Nm  string `xml:"Nm"`
		Pty struct {
			Nm string `xml:"Nm"`
		} `xml:"Pty"`
	}
	type Account struct {
		Id struct {
			IBAN string `xml:"IBAN"`
		} `xml:"Id"`
	}
	type BkTxCd struct {
		Domn struct {
			Fmly struct {
				SubFmlyCd string `xml:"SubFmlyCd"`
			} `xml:"Fmly"`
		} `xml:"Domn"`
		Prtry struct {
			Cd string `xml:"Cd"`
		} `xml:"Prtry"`
	}
	type TxDtls struct {
		Refs struct {
			EndToEndId string `xml:"EndToEndId"`
			MndtId     string `xml:"MndtId"`
		} `xml:"Refs"`
		BkTxCd    BkTxCd `xml:"BkTxCd"`
		RltdPties struct {
			Dbtr     Party   `xml:"Dbtr"`
			DbtrAcct Account `xml:"DbtrAcct"`
			Cdtr     Party   `xml:"Cdtr"`
			CdtrAcct Account `xml:"CdtrAcct"`
		} `xml:"RltdPties"`
		RmtInf struct {
			Ustrd []string `xml:"Ustrd"`
		} `xml:"RmtInf"`
		RtrInf struct {
			Rsn struct {
				Cd string `xml:"Cd"`
			} `xml:"Rsn"`
		} `xml:"RtrInf"`
		AddtlTxInf string `xml:"AddtlTxInf"`
	}
	type Ntry struct {
		Amt       string `xml:"Amt"`
		CdtDbtInd string `xml:"CdtDbtInd"`
		BookgDt   struct {
			Dt   string `xml:"Dt"`
			DtTm string `xml:"DtTm"`
		} `xml:"BookgDt"`
		BkTxCd   BkTxCd `xml:"BkTxCd"`
		NtryDtls struct {
			TxDtls []TxDtls `xml:"TxDtls"`
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

	optional := func(values ...string) *string {
		for _, v := range values {
			if v = strings.Join(strings.Fields(v), " "); v != "" && v != "NOTPROVIDED" {
				return &v
			}
		}
		return nil
	}
	name := func(p Party) string {
		if p.Nm != "" {
			return p.Nm
		}
		return p.Pty.Nm
	}

	var results []camt053Entry
	for _, e := range doc.Entries {
		date, err := time.Parse("2006-01-02", e.BookgDt.Dt)
		if err != nil && len(e.BookgDt.DtTm) >= 10 {
			date, _ = time.Parse("2006-01-02", e.BookgDt.DtTm[:10])
		}
		var tx TxDtls
		if len(e.NtryDtls.TxDtls) > 0 {
			tx = e.NtryDtls.TxDtls[0]
		}
		remittance := strings.TrimSpace(strings.Join(tx.RmtInf.Ustrd, " "))
		description := remittance
		if description == "" {
			description = strings.TrimSpace(tx.AddtlTxInf)
		}
		if description == "" {
			description = strings.TrimSpace(e.AddtlNtryInf)
		}
		amount, _ := strconv.ParseFloat(strings.TrimSpace(e.Amt), 64)
		debit := e.CdtDbtInd == "DBIT"
		if debit {
			amount = -amount
		}

		isReturn := false
		if debit {
			for _, code := range []BkTxCd{e.BkTxCd, tx.BkTxCd} {
				if code.Domn.Fmly.SubFmlyCd == "UPDD" || gvcReturnedDebit.MatchString(code.Prtry.Cd) {
					isReturn = true
				}
			}
			isReturn = isReturn || tx.RtrInf.Rsn.Cd != ""
		}
		// The other party: the payer of a credit or of a returned debit,
		// otherwise the payee.
		party, account := tx.RltdPties.Dbtr, tx.RltdPties.DbtrAcct
		if debit && !(isReturn && (name(party) != "" || account.Id.IBAN != "")) {
			party, account = tx.RltdPties.Cdtr, tx.RltdPties.CdtrAcct
		}
		bank := store.BankEntry{
			CounterpartyName: optional(name(party)), EndToEndID: optional(tx.Refs.EndToEndId),
			MandateRef: optional(tx.Refs.MndtId), IsReturn: isReturn, ReturnReason: optional(tx.RtrInf.Rsn.Cd),
		}
		if iban := sepa.NormalizeIBAN(account.Id.IBAN); iban != "" {
			bank.CounterpartyIBAN = &iban
		}
		results = append(results, camt053Entry{Date: date, Amount: amount, Description: description, Bank: bank})
	}
	return results, nil
}

// ── mapping ───────────────────────────────────────────────────────────────────

func toAPITransaction(t store.Transaction) openapi.FinanceTransaction {
	return openapi.FinanceTransaction{
		Id: t.ID, Date: t.Date, Description: t.Description, Amount: float32(t.Amount),
		Type: openapi.FinanceTransactionType(t.Type), Category: t.Category, Notes: t.Notes,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toAPIFee(f store.Fee) openapi.MemberFee {
	return openapi.MemberFee{
		Id: f.ID, MemberId: f.Member.ID, Member: toAPIMemberSummary(f.Member), Amount: float32(f.Amount),
		DueDate: f.DueDate, PaidDate: f.PaidDate, Description: f.Description, Year: f.Year,
		CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}
