package dsar

import (
	"context"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

// Attendance and honours are not stored by the application yet; when those
// modules arrive they register their own providers here or in their package.
func init() {
	Register(Provider{Key: "masterData", Title: "Master data", Scope: db.ScopeMembersCore, Collect: collectMaster})
	Register(Provider{Key: "contact", Title: "Contact details", Scope: db.ScopeMembersContact, Collect: collectContact})
	Register(Provider{Key: "bank", Title: "Bank details", Scope: db.ScopeMembersBank, Collect: collectBank})
	Register(Provider{Key: "assignments", Title: "Roles, groups and sections", Scope: db.ScopeMembersCore, Collect: collectAssignments})
	Register(Provider{Key: "household", Title: "Household", Scope: db.ScopeMembersCore, Collect: collectHousehold})
	Register(Provider{Key: "guardians", Title: "Guardians", Scope: db.ScopeMembersContact, Collect: collectGuardians})
	Register(Provider{Key: "directDebits", Title: "SEPA direct debits", Scope: db.ScopeMembersBank, Collect: collectDirectDebits})
	Register(Provider{Key: "feeModel", Title: "Fee model", Scope: db.ScopeFinance, Collect: collectFeeModel})
	Register(Provider{Key: "fees", Title: "Membership fees", Scope: db.ScopeFinance, Collect: collectFees})
	Register(Provider{Key: "feePayments", Title: "Bank payments assigned to fees", Scope: db.ScopeMembersBank, Collect: collectFeePayments})
	Register(Provider{Key: "donations", Title: "Donations and donation receipts", Scope: db.ScopeFinance, Collect: collectDonations})
	Register(Provider{Key: "loans", Title: "Inventory loans", Scope: db.ScopeInventory, Collect: collectLoans})
	Register(Provider{Key: "history", Title: "Change history", Scope: db.ScopeMembersCore, Collect: collectHistory})
}

func date(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func collectMaster(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	m, err := st.GetMember(ctx, id)
	if err != nil {
		return nil, err
	}
	return []Record{{
		{"id", "ID", m.ID},
		{"number", "Member number", m.Number},
		{"firstName", "First name", m.FirstName},
		{"lastName", "Last name", m.LastName},
		{"birthday", "Birthday", date(m.Birthday)},
		{"entryDate", "Entry date", date(m.EntryDate)},
		{"exitDate", "Exit date", date(m.ExitDate)},
		{"comment", "Comment", m.Comment},
		{"createdAt", "Created", ts(m.CreatedAt)},
		{"updatedAt", "Last changed", ts(m.UpdatedAt)},
	}}, nil
}

func collectContact(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	m, err := st.GetMember(ctx, id)
	if err != nil {
		return nil, err
	}
	c := m.Contact
	return []Record{{
		{"email", "E-mail", c.Email},
		{"phone", "Phone", c.Phone},
		{"phoneMobile", "Mobile", c.PhoneMobile},
		{"street", "Street", c.Street},
		{"postalCode", "Postal code", c.PostalCode},
		{"city", "City", c.City},
		{"state", "State", c.State},
	}}, nil
}

func collectBank(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	m, err := st.GetMember(ctx, id)
	if err != nil {
		return nil, err
	}
	b := m.Bank
	ref, err := st.GetMandateReference(ctx, id)
	if err != nil {
		return nil, err
	}
	return []Record{{
		{"accountHolder", "Account holder", b.AccountHolder},
		{"iban", "IBAN", b.IBAN},
		{"bic", "BIC", b.BIC},
		{"bankName", "Bank", b.BankName},
		{"sepaMandateDate", "SEPA mandate date", date(b.SEPAMandateDate)},
		{"sepaMandateReference", "SEPA mandate reference", ref},
	}}, nil
}

func collectDirectDebits(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	runs, err := st.ListSEPARuns(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, run := range runs {
		for _, it := range run.Items {
			if it.Member.ID != id {
				continue
			}
			out = append(out, Record{
				{"collectionDate", "Collection date", date(&run.CollectionDate)},
				{"amount", "Amount", db.FromCents(it.AmountCents)},
				{"mandateReference", "Mandate reference", it.MandateRef},
				{"sequenceType", "Sequence type", it.SequenceType},
				{"debtorName", "Account holder", it.DebtorName},
				{"iban", "IBAN", it.IBAN},
				{"remittance", "Remittance information", it.Remittance},
			})
		}
	}
	return out, nil
}

func collectAssignments(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	m, err := st.GetMember(ctx, id)
	if err != nil {
		return nil, err
	}
	var out []Record
	add := func(kind string, list []store.NamedEntity) {
		for _, e := range list {
			out = append(out, Record{{"type", "Type", kind}, {"id", "ID", e.ID}, {"name", "Name", e.Name}})
		}
	}
	add("role", m.Roles)
	add("group", m.Groups)
	add("section", m.Sections)
	return out, nil
}

func collectHousehold(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	list, err := st.ListHouseholds(ctx)
	if err != nil {
		return nil, err
	}
	for _, h := range list {
		for _, m := range h.Members {
			if m.ID != id {
				continue
			}
			var payer any
			if h.Payer != nil {
				payer = h.Payer.FirstName + " " + h.Payer.LastName
			}
			return []Record{{
				{"id", "ID", h.ID},
				{"name", "Household", h.Name},
				{"payer", "Main payer", payer},
				{"members", "Members", len(h.Members)},
			}}, nil
		}
	}
	return nil, nil
}

func collectGuardians(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	list, err := st.ListGuardians(ctx, id)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, g := range list {
		var name any = g.Name
		if g.Member != nil {
			name = g.Member.FirstName + " " + g.Member.LastName
		}
		out = append(out, Record{
			{"id", "ID", g.ID},
			{"name", "Name", name},
			{"relation", "Relation", g.Relation},
			{"email", "E-mail", g.Email},
			{"phone", "Phone", g.Phone},
		})
	}
	return out, nil
}

func collectFees(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	fees, err := st.ListFees(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, f := range fees {
		if f.Member.ID != id {
			continue
		}
		out = append(out, Record{
			{"id", "ID", f.ID},
			{"year", "Year", f.Year},
			{"amount", "Amount", f.Amount},
			{"dueDate", "Due date", f.DueDate.Format("2006-01-02")},
			{"paidDate", "Paid on", date(f.PaidDate)},
			{"description", "Description", f.Description},
		})
	}
	return out, nil
}

func collectFeeModel(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	a, err := st.GetFeeAssignment(ctx, id)
	if err != nil || a.Mode == store.FeeModeAuto {
		return nil, err
	}
	rec := Record{{"mode", "Assignment", a.Mode}}
	if a.ModelID != nil {
		m, err := st.GetFeeModel(ctx, *a.ModelID)
		if err != nil {
			return nil, err
		}
		rec = append(rec, Field{"model", "Fee model", m.Name})
	}
	return []Record{rec}, nil
}

// collectFeePayments lists the bank statement entries assigned to the
// member's fees, with the payer as it appeared on the statement.
func collectFeePayments(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	entries, err := st.ReconciliationEntries(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		for _, p := range e.Payments {
			if p.Member.ID != id {
				continue
			}
			kind := "payment"
			if e.Bank.IsReturn {
				kind = "returned direct debit"
			}
			out = append(out, Record{
				{"date", "Booking date", date(&e.Date)},
				{"type", "Type", kind},
				{"feeId", "Fee", p.FeeID},
				{"amount", "Assigned amount", db.FromCents(p.AmountCents)},
				{"payerName", "Payer", e.Bank.CounterpartyName},
				{"payerIban", "Payer IBAN", e.Bank.CounterpartyIBAN},
				{"remittance", "Remittance information", e.Description},
			})
		}
	}
	return out, nil
}

func collectDonations(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	entries, err := st.ListDonations(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.Member == nil || e.Member.ID != id {
			continue
		}
		out = append(out, Record{
			{"type", "Type", "donation"},
			{"date", "Date", date(&e.Date)},
			{"amount", "Amount", db.FromCents(e.AmountCents)},
			{"description", "Description", e.Description},
			{"expenseWaiver", "Waiver of expense reimbursement", e.ExpenseWaiver},
			{"receipt", "Receipt", e.ReceiptNumber},
		})
	}
	receipts, err := st.ListReceipts(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, r := range receipts {
		if r.MemberID == nil || *r.MemberID != id {
			continue
		}
		out = append(out, Record{
			{"type", "Type", "donation receipt"},
			{"number", "Number", r.Number},
			{"issueDate", "Issue date", date(&r.IssueDate)},
			{"amount", "Total", db.FromCents(r.TotalCents)},
			{"recipient", "Name and address", r.RecipientName + ", " + strings.ReplaceAll(r.RecipientAddress, "\n", ", ")},
			{"duplicate", "Duplicate", r.DuplicateOf != nil},
			{"voided", "Voided", date(r.VoidedAt)},
		})
	}
	return out, nil
}

func collectLoans(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	loans, err := st.ListLoans(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, l := range loans {
		if l.MemberID != id {
			continue
		}
		var name any
		if item, err := st.GetItem(ctx, l.ItemID); err == nil {
			name = item.Name
		}
		out = append(out, Record{
			{"id", "ID", l.ID},
			{"itemId", "Item ID", l.ItemID},
			{"item", "Item", name},
			{"loanedAt", "Loaned on", l.LoanedAt.Format("2006-01-02")},
			{"dueDate", "Due date", date(l.DueDate)},
			{"returnedAt", "Returned on", date(l.ReturnedAt)},
			{"notes", "Notes", l.Notes},
		})
	}
	return out, nil
}

func collectHistory(ctx context.Context, st *store.Store, id string) ([]Record, error) {
	entries, err := st.ListAudit(ctx)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.EntityID != id {
			continue
		}
		out = append(out, Record{
			{"at", "Time", ts(e.CreatedAt)},
			{"action", "Action", e.Action},
			{"entity", "Entity", e.Entity},
			{"user", "By", e.UserID},
		})
	}
	return out, nil
}
