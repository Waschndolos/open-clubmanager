package db

// Scope names a unit of data that is shared and permission-checked as a whole.
// The future sync layer encrypts change files per scope, so a device only ever
// receives (and can only decrypt) the scopes its user has been granted.
type Scope string

const (
	ScopeMembersCore    Scope = "members.core"
	ScopeMembersContact Scope = "members.contact"
	ScopeMembersBank    Scope = "members.bank"
	ScopeFinance        Scope = "finance"
	ScopeAdmin          Scope = "admin"
)

// Table describes how a table takes part in sync.
type Table struct {
	Name  string
	Scope Scope
	// AppendOnly tables never update or delete rows and therefore carry
	// neither updated_at nor deleted_at.
	AppendOnly bool
}

// SyncedTables lists every table whose rows will be replicated between
// devices, with the scope that governs access to it. Every table in this list
// has a TEXT primary key (or a composite key of TEXT ids) and, unless
// AppendOnly, the columns updated_at and deleted_at (tombstone).
//
// Tables not listed here are local to the device and must never be synced:
// user_preferences, refresh_tokens, password_reset_tokens and goose's
// bookkeeping table.
var SyncedTables = []Table{
	{Name: "members", Scope: ScopeMembersCore},
	{Name: "roles", Scope: ScopeMembersCore},
	{Name: "groups", Scope: ScopeMembersCore},
	{Name: "sections", Scope: ScopeMembersCore},
	{Name: "member_roles", Scope: ScopeMembersCore},
	{Name: "member_groups", Scope: ScopeMembersCore},
	{Name: "member_sections", Scope: ScopeMembersCore},
	{Name: "inventory_items", Scope: ScopeMembersCore},
	{Name: "inventory_loans", Scope: ScopeMembersCore},

	{Name: "member_contacts", Scope: ScopeMembersContact},

	{Name: "member_bank_details", Scope: ScopeMembersBank},

	{Name: "finance_transactions", Scope: ScopeFinance},
	{Name: "member_fees", Scope: ScopeFinance},

	{Name: "users", Scope: ScopeAdmin},
	{Name: "audit_log", Scope: ScopeAdmin, AppendOnly: true},
}

// LocalTables are application tables that stay on this device.
var LocalTables = []string{"user_preferences", "refresh_tokens", "password_reset_tokens"}
