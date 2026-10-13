package db

// Scope names a unit of data that is shared and permission-checked as a whole.
// The sync layer encrypts change files per scope, so a device only ever
// receives (and can only decrypt) the scopes its user has been granted. See
// docs/sync-protocol.md §4.
type Scope string

const (
	ScopeMembersCore    Scope = "members.core"
	ScopeMembersContact Scope = "members.contact"
	ScopeMembersBank    Scope = "members.bank"
	ScopeFinance        Scope = "finance"
	ScopeInventory      Scope = "inventory"
	// ScopeAdmin carries the ACL and club settings, which are not SQL tables
	// in protocol version 1.
	ScopeAdmin Scope = "admin"
)

// Scopes lists every scope.
var Scopes = []Scope{ScopeMembersCore, ScopeMembersContact, ScopeMembersBank, ScopeFinance, ScopeInventory, ScopeAdmin}

// Table describes how a table takes part in sync.
type Table struct {
	Name  string
	Scope Scope
	// PK lists the primary key columns in the order they are declared.
	PK []string
	// AppendOnly tables never update or delete rows and therefore carry
	// neither updated_at nor deleted_at.
	AppendOnly bool
}

var idPK = []string{"id"}

// SyncedTables lists every table whose rows will be replicated between
// devices, with the scope that governs access to it. Every table in this list
// has a TEXT primary key (or a composite key of TEXT ids) and, unless
// AppendOnly, the columns updated_at and deleted_at (tombstone).
//
// Tables not listed here are local to the device and must never be synced;
// they are listed in LocalTables.
var SyncedTables = []Table{
	{Name: "members", Scope: ScopeMembersCore, PK: idPK},
	{Name: "roles", Scope: ScopeMembersCore, PK: idPK},
	{Name: "groups", Scope: ScopeMembersCore, PK: idPK},
	{Name: "sections", Scope: ScopeMembersCore, PK: idPK},
	{Name: "member_roles", Scope: ScopeMembersCore, PK: []string{"member_id", "role_id"}},
	{Name: "member_groups", Scope: ScopeMembersCore, PK: []string{"member_id", "group_id"}},
	{Name: "member_sections", Scope: ScopeMembersCore, PK: []string{"member_id", "section_id"}},
	{Name: "member_honors", Scope: ScopeMembersCore, PK: idPK},
	{Name: "member_membership", Scope: ScopeMembersCore, PK: []string{"member_id"}},
	{Name: "tasks", Scope: ScopeMembersCore, PK: idPK},
	{Name: "qualification_types", Scope: ScopeMembersCore, PK: idPK},
	{Name: "member_qualifications", Scope: ScopeMembersCore, PK: idPK},
	{Name: "households", Scope: ScopeMembersCore, PK: idPK},
	{Name: "household_members", Scope: ScopeMembersCore, PK: []string{"member_id"}},

	{Name: "member_contacts", Scope: ScopeMembersContact, PK: []string{"member_id"}},
	{Name: "member_guardians", Scope: ScopeMembersContact, PK: idPK},

	{Name: "member_bank_details", Scope: ScopeMembersBank, PK: []string{"member_id"}},

	{Name: "finance_transactions", Scope: ScopeFinance, PK: idPK},
	{Name: "member_fees", Scope: ScopeFinance, PK: idPK},

	{Name: "inventory_items", Scope: ScopeInventory, PK: idPK},
	{Name: "inventory_loans", Scope: ScopeInventory, PK: idPK},
	{Name: "inventory_item_status", Scope: ScopeInventory, PK: []string{"item_id"}},
	{Name: "inventory_maintenance", Scope: ScopeInventory, PK: idPK},
}

// LocalTables are application tables that stay on this device.
//
// users and audit_log are local because sync identity is the device, password
// hashes must never leave it, and the signed change files already are the
// shared audit trail (docs/sync-protocol.md §22).
var LocalTables = []string{
	"users", "audit_log", "user_preferences", "refresh_tokens", "password_reset_tokens",
	"sync_device", "sync_field_clock", "sync_outbox", "sync_conflicts",
	"sync_club", "sync_acl", "sync_keys", "sync_key_drafts", "sync_invites", "sync_streams",
	"sync_scopes", "sync_uploads", "sync_pending", "sync_history", "sync_clock_warnings", "sync_clock_epochs",
}

// SyncedTable returns the registry entry of a synced table.
func SyncedTable(name string) (Table, bool) {
	for _, t := range SyncedTables {
		if t.Name == name {
			return t, true
		}
	}
	return Table{}, false
}

// UniqueColumn is a column whose non-NULL values must be unique among live
// rows. SQLite does not enforce this with an index, because two valid changes
// from different devices may collide; the store checks it before local writes
// and the sync layer flags collisions after applying remote changes
// (docs/sync-protocol.md §14.2).
type UniqueColumn struct {
	Table  string
	Column string
}

// UniqueColumns lists the uniqueness rules on synced tables.
var UniqueColumns = []UniqueColumn{
	{Table: "members", Column: "number"},
	{Table: "member_contacts", Column: "email"},
	{Table: "roles", Column: "name"},
	{Table: "groups", Column: "name"},
	{Table: "sections", Column: "name"},
	{Table: "qualification_types", Column: "name"},
}
