package acl

import (
	"fmt"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Perm is the access a role grants on a scope. A scope missing from a role
// grants nothing.
type Perm string

const (
	Read      Perm = "r"
	ReadWrite Perm = "rw"
)

// CanRead reports whether p allows reading.
func (p Perm) CanRead() bool { return p == Read || p == ReadWrite }

// CanWrite reports whether p allows writing.
func (p Perm) CanWrite() bool { return p == ReadWrite }

// Role maps scopes to permissions.
type Role map[db.Scope]Perm

// Requires lists, per scope, the scopes a role granting it must also be able
// to read (§4.1 "Requires read on").
var Requires = map[db.Scope][]db.Scope{
	db.ScopeMembersContact: {db.ScopeMembersCore},
	db.ScopeMembersBank:    {db.ScopeMembersCore},
	db.ScopeFinance:        {db.ScopeMembersCore},
	db.ScopeInventory:      {db.ScopeMembersCore},
}

// CheckDependencies verifies the scope dependencies of §4.1 for one role.
func (r Role) CheckDependencies() error {
	for s, p := range r {
		if !p.CanRead() {
			continue
		}
		for _, need := range Requires[s] {
			if !r[need].CanRead() {
				return fmt.Errorf("grants %s but not read on %s", s, need)
			}
		}
	}
	return nil
}

// Role ids of the default roles.
const (
	RoleBoard            = "board"
	RoleTreasurer        = "treasurer"
	RoleSecretary        = "secretary"
	RoleCoach            = "coach"
	RoleEquipmentManager = "equipment_manager"
)

// DefaultRoles returns the roles new clubs start with (§4.2).
func DefaultRoles() map[string]Role {
	return map[string]Role{
		RoleBoard: {
			db.ScopeMembersCore: ReadWrite, db.ScopeMembersContact: ReadWrite, db.ScopeMembersBank: ReadWrite,
			db.ScopeFinance: ReadWrite, db.ScopeInventory: ReadWrite, db.ScopeAdmin: ReadWrite,
		},
		RoleTreasurer: {
			db.ScopeMembersCore: Read, db.ScopeMembersContact: Read, db.ScopeMembersBank: ReadWrite,
			db.ScopeFinance: ReadWrite, db.ScopeInventory: Read,
		},
		RoleSecretary: {
			db.ScopeMembersCore: ReadWrite, db.ScopeMembersContact: ReadWrite, db.ScopeInventory: Read,
		},
		RoleCoach: {
			db.ScopeMembersCore: Read, db.ScopeInventory: Read,
		},
		RoleEquipmentManager: {
			db.ScopeMembersCore: Read, db.ScopeInventory: ReadWrite,
		},
	}
}

// RoleLabels are the German labels of the default roles, for the UI.
var RoleLabels = map[string]string{
	RoleBoard:            "Vorstand",
	RoleTreasurer:        "Kassierer",
	RoleSecretary:        "Schriftführer",
	RoleCoach:            "Übungsleiter",
	RoleEquipmentManager: "Gerätewart",
}
