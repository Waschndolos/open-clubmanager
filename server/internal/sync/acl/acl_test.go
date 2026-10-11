package acl

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

const testClubID = "0190f5c2-7b1e-7c3a-9d2e-4b8a1f6e3c21"

// fixture is a club with root, an admin device A, a coach B and an admin C
// that is revoked in version 2.
type fixture struct {
	t       *testing.T
	root    *identity.KeyPair
	g       *Genesis
	a, b, c *identity.Device
	v1, v2  *Version
}

func newDevice(t *testing.T) *identity.Device {
	t.Helper()
	k, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return identity.NewDevice(k)
}

func entry(d *identity.Device, role string, added uint64) Device {
	return Device{
		DeviceID: d.ID, SignPub: d.SignPubB64(), AgeRecipient: d.AgeRecipient(),
		Person: "P " + d.ID[:4], Role: role, Status: StatusActive, AddedIn: added,
	}
}

func checks(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = wire.Hash([]byte{byte(i)})
	}
	return out
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, a: newDevice(t), b: newDevice(t), c: newDevice(t)}
	var err error
	if f.root, err = identity.Generate(); err != nil {
		t.Fatal(err)
	}
	raw, sig, err := NewGenesis(testClubID, "TSV", time.Unix(1_800_000_000, 0), f.root.Sign, f.root.AgeRecipient())
	if err != nil {
		t.Fatal(err)
	}
	if f.g, err = ParseGenesis(raw, sig); err != nil {
		t.Fatal(err)
	}

	first := &ACL{
		Type: aclType, ClubID: testClubID, Version: 1, Author: AuthorRoot,
		CreatedHLC: hlc.Timestamp{MS: 1}.String(),
		Roles:      DefaultRoles(),
		Devices: []Device{
			entry(f.a, RoleBoard, 1), entry(f.b, RoleCoach, 1), entry(f.c, RoleBoard, 1),
		},
		Epochs: map[db.Scope]uint64{}, ClockEpochs: map[db.Scope]uint64{}, KeyChecks: map[db.Scope][]string{}, MinSchema: 1,
	}
	for _, s := range db.Scopes {
		first.Epochs[s] = 1
		first.ClockEpochs[s] = 1
		first.KeyChecks[s] = checks(1)
	}
	f.v1 = f.mustVerify(nil, first, f.root.Sign)

	second := f.v1.Next(f.a.ID, hlc.Timestamp{MS: 2})
	c := second.Device(f.c.ID)
	c.Status, c.RevokedIn, c.Cutoff = StatusRevoked, 2, map[db.Scope]Cutoff{}
	for _, s := range db.Scopes {
		c.Cutoff[s] = Cutoff{}
	}
	h := wire.Hash([]byte("segment 3"))
	c.Cutoff[db.ScopeMembersCore] = Cutoff{Seq: 3, Hash: &h}
	for _, s := range db.Scopes {
		second.Epochs[s] = 2
		second.KeyChecks[s] = checks(2)
	}
	f.v2 = f.mustVerify(f.v1, second, f.a.Sign)
	return f
}

func (f *fixture) sign(a *ACL, priv ed25519.PrivateKey) ([]byte, []byte) {
	f.t.Helper()
	raw, sig, err := Sign(a, priv)
	if err != nil {
		f.t.Fatal(err)
	}
	return raw, sig
}

func (f *fixture) mustVerify(prev *Version, a *ACL, priv ed25519.PrivateKey) *Version {
	f.t.Helper()
	raw, sig := f.sign(a, priv)
	v, err := Verify(f.g, prev, raw, sig)
	if err != nil {
		f.t.Fatalf("version %d: %v", a.Version, err)
	}
	return v
}

func TestVerifyRules(t *testing.T) {
	f := newFixture(t)
	other := newDevice(t)

	tests := []struct {
		name string
		// prev is the version the candidate follows; nil means version 1.
		prev *Version
		// edit changes a valid draft; signer defaults to admin A (root
		// for version 1).
		edit   func(a *ACL)
		signer ed25519.PrivateKey
		// raw post-processes the signed bytes (signature stays the same).
		raw  func([]byte) []byte
		want error
	}{
		{name: "valid admin version", prev: f.v2, edit: func(a *ACL) {}},
		{name: "valid root version", prev: f.v2, edit: func(a *ACL) { a.Author = AuthorRoot }, signer: f.root.Sign},
		{name: "valid version 1", prev: nil, edit: func(a *ACL) {}},

		// Rule 1.
		{name: "v1 with prev", prev: nil, edit: func(a *ACL) { p := f.v1.Hash(); a.Prev = &p }, want: ErrBadPrev},
		{name: "v1 by a device", prev: nil, edit: func(a *ACL) { a.Author = f.a.ID }, signer: f.a.Sign, want: ErrNotAuthorized},
		{name: "v1 forged root signature", prev: nil, edit: func(a *ACL) {}, signer: other.Sign, want: ErrBadSignature},
		{name: "v1 with version 2", prev: nil, edit: func(a *ACL) { a.Version = 2 }, want: ErrBadPrev},

		// Rule 2.
		{name: "prev of another version", prev: f.v2, edit: func(a *ACL) { p := f.v1.Hash(); a.Prev = &p }, want: ErrBadPrev},
		{name: "prev null", prev: f.v2, edit: func(a *ACL) { a.Prev = nil }, want: ErrBadPrev},
		{name: "version gap", prev: f.v2, edit: func(a *ACL) { a.Version = 4 }, want: ErrBadPrev},
		{name: "version repeated", prev: f.v2, edit: func(a *ACL) { a.Version = 2 }, want: ErrBadPrev},

		// Rule 3.
		{name: "non-admin author", prev: f.v2, edit: func(a *ACL) { a.Author = f.b.ID }, signer: f.b.Sign, want: ErrNotAuthorized},
		{name: "unknown author", prev: f.v2, edit: func(a *ACL) { a.Author = other.ID }, signer: other.Sign, want: ErrNotAuthorized},
		{name: "revoked admin author", prev: f.v2, edit: func(a *ACL) { a.Author = f.c.ID }, signer: f.c.Sign, want: ErrNotAuthorized},
		{name: "admin signed by other key", prev: f.v2, edit: func(a *ACL) {}, signer: other.Sign, want: ErrBadSignature},
		{name: "root author signed by device", prev: f.v2, edit: func(a *ACL) { a.Author = AuthorRoot }, want: ErrBadSignature},
		{name: "content changed after signing", prev: f.v2, edit: func(a *ACL) {},
			raw: func(b []byte) []byte {
				return bytes.Replace(b, []byte(`"min_schema": 1`), []byte(`"min_schema": 2`), 1)
			},
			want: ErrBadSignature},

		// Rule 4.
		{name: "epoch decreases", prev: f.v2, edit: func(a *ACL) {
			a.Epochs[db.ScopeFinance] = 1
			a.KeyChecks[db.ScopeFinance] = a.KeyChecks[db.ScopeFinance][:1]
		}, want: ErrEpochDecreased},
		{name: "epoch increases with check", prev: f.v2, edit: func(a *ACL) {
			a.Epochs[db.ScopeFinance] = 3
			a.KeyChecks[db.ScopeFinance] = checks(3)
		}},
		{name: "epoch without key check", prev: f.v2, edit: func(a *ACL) { a.Epochs[db.ScopeFinance] = 3 }, want: ErrKeyChecks},
		{name: "key check rewritten", prev: f.v2, edit: func(a *ACL) {
			a.KeyChecks[db.ScopeFinance][0] = wire.Hash([]byte("other"))
		}, want: ErrKeyChecks},
		{name: "key check malformed", prev: f.v2, edit: func(a *ACL) {
			a.Epochs[db.ScopeFinance] = 3
			a.KeyChecks[db.ScopeFinance] = append(a.KeyChecks[db.ScopeFinance], "md5:xyz")
		}, want: ErrKeyChecks},

		// Rule 5.
		{name: "device removed", prev: f.v2, edit: func(a *ACL) { a.Devices = a.Devices[:2] }, want: ErrDeviceChanged},
		{name: "device key replaced", prev: f.v2, edit: func(a *ACL) {
			a.Devices[1].AgeRecipient = other.AgeRecipient()
		}, want: ErrDeviceChanged},
		{name: "added_in changed", prev: f.v2, edit: func(a *ACL) { a.Devices[1].AddedIn = 2 }, want: ErrDeviceChanged},
		{name: "revoked reactivated", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.c.ID)
			d.Status, d.RevokedIn, d.Cutoff = StatusActive, 0, nil
		}, want: ErrDeviceChanged},
		{name: "cutoff raised later", prev: f.v2, edit: func(a *ACL) {
			h := wire.Hash([]byte("segment 99"))
			a.Device(f.c.ID).Cutoff[db.ScopeMembersCore] = Cutoff{Seq: 99, Hash: &h}
		}, want: ErrDeviceChanged},
		{name: "cutoff hash rewritten", prev: f.v2, edit: func(a *ACL) {
			h := wire.Hash([]byte("other segment 3"))
			a.Device(f.c.ID).Cutoff[db.ScopeMembersCore] = Cutoff{Seq: 3, Hash: &h}
		}, want: ErrDeviceChanged},
		{name: "clock epoch decreased", prev: f.v2, edit: func(a *ACL) { a.ClockEpochs[db.ScopeFinance] = 0 }, want: ErrMalformed},
		{name: "clock epoch increased", prev: f.v2, edit: func(a *ACL) { a.ClockEpochs[db.ScopeFinance] = 2 }},
		{name: "clock epochs missing", prev: f.v2, edit: func(a *ACL) { a.ClockEpochs = nil }, want: ErrMalformed},
		{name: "revoked_in rewritten", prev: f.v2, edit: func(a *ACL) { a.Device(f.c.ID).RevokedIn = 3 }, want: ErrDeviceChanged},
		{name: "revocation with wrong revoked_in", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.b.ID)
			d.Status, d.RevokedIn = StatusRevoked, 2
		}, want: ErrDeviceChanged},
		{name: "revocation", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.b.ID)
			d.Status, d.RevokedIn = StatusRevoked, 3 // a coach writes nothing: no cutoffs
		}},
		{name: "revocation with cutoff for a read-only scope", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.b.ID)
			d.Status, d.RevokedIn, d.Cutoff = StatusRevoked, 3, map[db.Scope]Cutoff{db.ScopeInventory: {}}
		}, want: ErrCutoff},
		{name: "revocation without cutoff for a writable scope", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.a.ID)
			d.Status, d.RevokedIn = StatusRevoked, 3
		}, want: ErrCutoff},
		{name: "cutoff with seq but without hash", prev: f.v2, edit: func(a *ACL) {
			d := a.Device(f.b.ID)
			d.Status, d.RevokedIn = StatusRevoked, 3
			a.Device(f.c.ID).Cutoff[db.ScopeFinance] = Cutoff{Seq: 2}
		}, want: ErrMalformed},
		{name: "new device", prev: f.v2, edit: func(a *ACL) { a.Devices = append(a.Devices, entry(other, RoleCoach, 3)) }},
		{name: "new device with old added_in", prev: f.v2, edit: func(a *ACL) {
			a.Devices = append(a.Devices, entry(other, RoleCoach, 2))
		}, want: ErrDeviceChanged},
		{name: "new device already revoked", prev: f.v2, edit: func(a *ACL) {
			d := entry(other, RoleCoach, 3)
			d.Status, d.RevokedIn = StatusRevoked, 3
			a.Devices = append(a.Devices, d)
		}, want: ErrDeviceChanged},
		{name: "v1 device with added_in 2", prev: nil, edit: func(a *ACL) { a.Devices[0].AddedIn = 2 }, want: ErrMalformed},

		// Rule 6.
		{name: "finance without members.core", prev: f.v2, edit: func(a *ACL) {
			a.Roles["kasse"] = Role{db.ScopeFinance: ReadWrite}
		}, want: ErrScopeDeps},
		{name: "contact needs core read, write on core not needed", prev: f.v2, edit: func(a *ACL) {
			a.Roles["kontakt"] = Role{db.ScopeMembersContact: ReadWrite, db.ScopeMembersCore: Read}
		}},
		{name: "default role weakened", prev: f.v2, edit: func(a *ACL) {
			delete(a.Roles[RoleTreasurer], db.ScopeMembersCore)
		}, want: ErrScopeDeps},

		// Format.
		{name: "wrong club", prev: f.v2, edit: func(a *ACL) { a.ClubID = "0190f5c2-7b1e-7c3a-9d2e-4b8a1f6e3c22" }, want: ErrWrongClub},
		{name: "wrong type", prev: f.v2, edit: func(a *ACL) { a.Type = "ocm.club" }, want: ErrMalformed},
		{name: "unknown scope in role", prev: f.v2, edit: func(a *ACL) { a.Roles[RoleCoach]["photos"] = Read }, want: ErrMalformed},
		{name: "unknown permission", prev: f.v2, edit: func(a *ACL) { a.Roles[RoleCoach][db.ScopeMembersCore] = "w" }, want: ErrMalformed},
		{name: "unknown role", prev: f.v2, edit: func(a *ACL) { a.Devices[1].Role = "president" }, want: ErrMalformed},
		{name: "device id not derived from key", prev: f.v2, edit: func(a *ACL) {
			d := entry(other, RoleCoach, 3)
			d.DeviceID = f.b.ID[:25] + "a"
			a.Devices = append(a.Devices, d)
		}, want: ErrMalformed},
		{name: "device listed twice", prev: f.v2, edit: func(a *ACL) { a.Devices = append(a.Devices, a.Devices[1]) }, want: ErrMalformed},
		{name: "bad age recipient", prev: f.v2, edit: func(a *ACL) {
			d := entry(other, RoleCoach, 3)
			d.AgeRecipient = "age1nope"
			a.Devices = append(a.Devices, d)
		}, want: ErrMalformed},
		{name: "missing scope epoch", prev: f.v2, edit: func(a *ACL) { delete(a.Epochs, db.ScopeAdmin) }, want: ErrMalformed},
		{name: "bad hlc", prev: f.v2, edit: func(a *ACL) { a.CreatedHLC = "yesterday" }, want: ErrMalformed},
		{name: "min_schema zero", prev: f.v2, edit: func(a *ACL) { a.MinSchema = 0 }, want: ErrMalformed},
		{name: "active device with cutoff", prev: f.v2, edit: func(a *ACL) {
			a.Devices[1].Cutoff = map[db.Scope]Cutoff{db.ScopeInventory: {}}
		}, want: ErrMalformed},
		{name: "unknown field", prev: f.v2, edit: func(a *ACL) {},
			raw: func(b []byte) []byte {
				return bytes.Replace(b, []byte(`"min_schema"`), []byte(`"extra": 1, "min_schema"`), 1)
			},
			want: ErrMalformed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var draft *ACL
			signer := f.a.Sign
			if tc.prev == nil {
				draft = f.v1.ACL.Clone()
				signer = f.root.Sign
			} else {
				draft = tc.prev.Next(f.a.ID, hlc.Timestamp{MS: 10})
			}
			tc.edit(draft)
			if tc.signer != nil {
				signer = tc.signer
			}
			raw, sig := f.sign(draft, signer)
			if tc.raw != nil {
				raw = tc.raw(raw)
				if tc.want != ErrBadSignature {
					// Re-sign so that only the content is at fault.
					sig = wire.Sign(signer, raw)
				}
			}
			_, err := Verify(f.g, tc.prev, raw, sig)
			if tc.want == nil && err != nil {
				t.Fatalf("want valid, got %v", err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestDefaultRolesSatisfyDependencies(t *testing.T) {
	for id, r := range DefaultRoles() {
		if err := r.CheckDependencies(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
		if _, ok := RoleLabels[id]; !ok {
			t.Errorf("%s has no label", id)
		}
	}
	if !DefaultRoles()[RoleBoard][db.ScopeAdmin].CanWrite() {
		t.Error("board must be an admin role")
	}
	for id, r := range DefaultRoles() {
		if id != RoleBoard && r[db.ScopeAdmin] != "" {
			t.Errorf("%s must not have admin access", id)
		}
	}
}

func TestPermissions(t *testing.T) {
	f := newFixture(t)
	v := f.v2
	if !v.IsAdmin(f.a.ID) || v.IsAdmin(f.b.ID) || v.IsAdmin(f.c.ID) {
		t.Error("admin flags wrong")
	}
	if v.Perm(f.b.ID, db.ScopeMembersCore) != Read || v.Perm(f.b.ID, db.ScopeFinance) != "" {
		t.Error("coach permissions wrong")
	}
	if v.Perm(f.c.ID, db.ScopeMembersCore) != "" {
		t.Error("revoked device must have no permissions")
	}
	readers := v.Readers(db.ScopeFinance)
	if len(readers) != 1 || readers[0] != f.a.ID {
		t.Errorf("finance readers = %v", readers)
	}
}

func TestGenesis(t *testing.T) {
	root, _ := identity.Generate()
	raw, sig, err := NewGenesis(testClubID, "TSV", time.Now(), root.Sign, root.AgeRecipient())
	if err != nil {
		t.Fatal(err)
	}
	g, err := ParseGenesis(raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Matches(testClubID, root.SignPubB64()); err != nil {
		t.Error(err)
	}
	other, _ := identity.Generate()
	if err := g.Matches(testClubID, other.SignPubB64()); !errors.Is(err, ErrGenesisMismatch) {
		t.Errorf("other root: %v", err)
	}
	if err := g.Matches("0190f5c2-0000-7c3a-9d2e-4b8a1f6e3c21", root.SignPubB64()); !errors.Is(err, ErrGenesisMismatch) {
		t.Errorf("other club: %v", err)
	}

	tampered := bytes.Replace(raw, []byte("TSV"), []byte("FCB"), 1)
	if _, err := ParseGenesis(tampered, sig); !errors.Is(err, ErrBadGenesis) {
		t.Errorf("tampered: %v", err)
	}
	// A club.json signed by a different key that names that key as root is
	// self-consistent, but does not match the pinned values.
	fRaw, fSig, _ := NewGenesis(testClubID, "TSV", time.Now(), other.Sign, other.AgeRecipient())
	fg, err := ParseGenesis(fRaw, fSig)
	if err != nil {
		t.Fatal(err)
	}
	if err := fg.Matches(g.ClubID, g.RootSignPub); !errors.Is(err, ErrGenesisMismatch) {
		t.Errorf("forged genesis: %v", err)
	}
	if _, err := ParseGenesis(raw, nil); !errors.Is(err, ErrBadGenesis) {
		t.Errorf("unsigned: %v", err)
	}
}
