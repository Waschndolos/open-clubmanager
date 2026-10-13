package dsar

import (
	"context"
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
	Register(Provider{Key: "fees", Title: "Membership fees", Scope: db.ScopeFinance, Collect: collectFees})
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
	return []Record{{
		{"accountHolder", "Account holder", b.AccountHolder},
		{"iban", "IBAN", b.IBAN},
		{"bic", "BIC", b.BIC},
		{"bankName", "Bank", b.BankName},
		{"sepaMandateDate", "SEPA mandate date", date(b.SEPAMandateDate)},
	}}, nil
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
