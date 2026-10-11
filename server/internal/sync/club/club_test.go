package club

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

var ctx = context.Background()

// node is one installation: its own data dir, database and device keys.
type node struct {
	t    *testing.T
	name string
	dir  string
	conn *sql.DB
	dev  *identity.Device
	tr   transport.Transport
	club *Club
}

func newNode(t *testing.T, name string, tr transport.Transport) *node {
	t.Helper()
	dir := t.TempDir()
	conn, err := db.Open(ctx, filepath.Join(dir, "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ks := &identity.PassphraseFileStore{Dir: filepath.Join(dir, "keys"), Passphrase: "pw-" + name, WorkFactor: 10}
	dev, err := identity.Ensure(ctx, conn, ks)
	if err != nil {
		t.Fatal(err)
	}
	return &node{t: t, name: name, dir: dir, conn: conn, dev: dev, tr: tr}
}

// folder returns a new handle on a shared sync folder.
func folder(t *testing.T, dir string) transport.Transport {
	t.Helper()
	lf, err := transport.OpenLocalFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lf.Close() })
	return lf
}

func (n *node) create(clubName string) *identity.RecoveryKit {
	n.t.Helper()
	c, kit, err := Create(ctx, n.conn, n.dev, n.tr, clubName, "Person "+n.name)
	if err != nil {
		n.t.Fatal(err)
	}
	n.club = c
	return kit
}

func (n *node) join(code string) string {
	n.t.Helper()
	c, safety, err := Join(ctx, n.conn, n.dev, n.tr, code, "Person "+n.name)
	if err != nil {
		n.t.Fatalf("%s join: %v", n.name, err)
	}
	n.club = c
	return safety
}

func (n *node) refresh() *RefreshResult {
	n.t.Helper()
	r, err := n.club.Refresh(ctx)
	if err != nil {
		n.t.Fatalf("%s refresh: %v", n.name, err)
	}
	return r
}

func (n *node) invite(role string) string {
	n.t.Helper()
	inv, err := n.club.CreateInvite(ctx, role, "", 0, TransportHint{})
	if err != nil {
		n.t.Fatal(err)
	}
	return inv.Code
}

// admit invites, joins and approves m into n's club with role.
func (n *node) admit(m *node, role string) {
	n.t.Helper()
	m.join(n.invite(role))
	if _, err := n.club.Approve(ctx, m.dev.ID); err != nil {
		n.t.Fatalf("approve %s: %v", m.name, err)
	}
	if r := m.refresh(); len(r.MissingKeys) > 0 || len(r.BadKeys) > 0 || m.club.Status() != StatusMember {
		n.t.Fatalf("%s after approval: status %s, missing %v, bad %v", m.name, m.club.Status(), r.MissingKeys, r.BadKeys)
	}
}

func (n *node) key(s db.Scope, e uint64) []byte {
	n.t.Helper()
	k, err := n.club.ScopeKey(ctx, s, e)
	if err != nil {
		n.t.Fatalf("%s key %s/%d: %v", n.name, s, e, err)
	}
	return k
}

func (n *node) cachedScopes() []string {
	rows, err := n.conn.Query(`SELECT scope || '/' || epoch FROM sync_keys ORDER BY scope, epoch`)
	if err != nil {
		n.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func TestMultiDeviceLifecycle(t *testing.T) {
	shared := t.TempDir()
	a := newNode(t, "a", folder(t, shared))
	b := newNode(t, "b", folder(t, shared))

	// §17.1: create club.
	kit := a.create("TSV Beispielstadt e.V.")
	assertRootNotPersisted(t, kit, a.dir, shared)
	kitText := kit.Text()
	kit.Wipe()
	if !strings.Contains(kitText, a.club.Genesis().ClubID) {
		t.Error("kit does not name the club")
	}

	head := a.club.Head()
	if head.ACL.Version != 1 || head.Author != acl.AuthorRoot || !head.IsAdmin(a.dev.ID) {
		t.Fatalf("version 1: %+v", head.ACL)
	}
	for _, s := range db.Scopes {
		a.key(s, 1)
		for _, rid := range []string{a.dev.ID, layout.RootRecipient} {
			if _, err := a.tr.Get(ctx, layout.KeyPath(s, 1, rid)); err != nil {
				t.Errorf("key file %s/%s: %v", s, rid, err)
			}
		}
	}
	if _, _, err := Create(ctx, a.conn, a.dev, a.tr, "Second", "x"); !errors.Is(err, ErrAlreadyInClub) {
		t.Errorf("second club: %v", err)
	}

	// §17.2: invite, join, check, approve.
	code := a.invite(acl.RoleCoach)
	safety := b.join(code)
	if b.club.Status() != StatusJoining {
		t.Fatalf("b status %s", b.club.Status())
	}
	reqs, err := a.club.JoinRequests(ctx)
	if err != nil || len(reqs) != 1 {
		t.Fatalf("requests %v, %v", reqs, err)
	}
	if r := reqs[0]; r.Err != nil || r.DeviceID != b.dev.ID || r.SafetyCode != safety || r.Role != acl.RoleCoach || r.Person != "Person b" {
		t.Fatalf("request %+v (safety on b: %s)", r, safety)
	}
	if _, err := b.club.JoinRequests(ctx); !errors.Is(err, ErrNotAdmin) {
		t.Errorf("non-admin lists requests: %v", err)
	}
	v2, err := a.club.Approve(ctx, b.dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v2.ACL.Version != 2 || v2.Device(b.dev.ID).Role != acl.RoleCoach || v2.Device(b.dev.ID).AddedIn != 2 {
		t.Fatalf("version 2: %+v", v2.Device(b.dev.ID))
	}
	if files, _ := a.tr.List(ctx, layout.RequestsDir); len(files) != 0 {
		t.Errorf("request files left: %v", files)
	}
	invs, _ := a.club.Invites(ctx)
	if len(invs) != 1 || invs[0].UsedAt == nil || invs[0].UsedBy != b.dev.ID {
		t.Errorf("invite not marked used: %+v", invs)
	}

	// §17.2 step 6: b finds itself and unwraps its keys.
	b.refresh()
	if b.club.Status() != StatusMember || b.club.Head().ACL.Version != 2 {
		t.Fatalf("b: %s at %d", b.club.Status(), b.club.Head().ACL.Version)
	}
	if !bytes.Equal(b.key(db.ScopeMembersCore, 1), a.key(db.ScopeMembersCore, 1)) {
		t.Error("b has another members.core key")
	}
	if got := b.cachedScopes(); !slices.Equal(got, []string{"inventory/1", "members.core/1"}) {
		t.Errorf("coach keys: %v", got)
	}
	if _, err := b.club.ScopeKey(ctx, db.ScopeFinance, 1); !errors.Is(err, ErrMissingKey) {
		t.Errorf("coach reads finance: %v", err)
	}
	if _, err := b.club.CreateInvite(ctx, acl.RoleBoard, "", 0, TransportHint{}); !errors.Is(err, ErrNotAdmin) {
		t.Errorf("coach invites: %v", err)
	}

	// §17.3: role upgrade wraps all epochs of the new scopes, no new epoch.
	v3, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleTreasurer)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range db.Scopes {
		if v3.Epochs[s] != 1 {
			t.Errorf("upgrade started epoch %d of %s", v3.Epochs[s], s)
		}
	}
	b.refresh()
	if !bytes.Equal(b.key(db.ScopeFinance, 1), a.key(db.ScopeFinance, 1)) {
		t.Error("treasurer has another finance key")
	}

	// §17.3: downgrade starts new epochs for the scopes b can no longer read.
	v4, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleCoach)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range db.Scopes {
		want := uint64(1)
		if s == db.ScopeFinance || s == db.ScopeMembersBank || s == db.ScopeMembersContact {
			want = 2
		}
		if v4.Epochs[s] != want {
			t.Errorf("after downgrade %s is at epoch %d, want %d", s, v4.Epochs[s], want)
		}
	}
	a.key(db.ScopeFinance, 2)
	if _, err := a.tr.Get(ctx, layout.KeyPath(db.ScopeFinance, 2, layout.RootRecipient)); err != nil {
		t.Errorf("root lacks the new epoch: %v", err)
	}
	if _, err := a.tr.Get(ctx, layout.KeyPath(db.ScopeFinance, 2, b.dev.ID)); !errors.Is(err, transport.ErrNotExist) {
		t.Errorf("new epoch wrapped for the downgraded device: %v", err)
	}
	b.refresh()
	b.key(db.ScopeFinance, 1) // history stays readable
	if _, err := b.club.ScopeKey(ctx, db.ScopeFinance, 2); !errors.Is(err, ErrMissingKey) {
		t.Errorf("downgraded device reads new epoch: %v", err)
	}

	// §17.4: revocation with the cutoff from sync_streams.
	if _, err := a.conn.Exec(`INSERT INTO sync_streams (device_id, scope, seq) VALUES (?, 'members.core', 41), (?, 'inventory', 3)`,
		b.dev.ID, b.dev.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.club.Revoke(ctx, a.dev.ID, nil); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("revoking the last admin: %v", err)
	}
	v5, err := a.club.Revoke(ctx, b.dev.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	rb := v5.Device(b.dev.ID)
	if rb.Status != acl.StatusRevoked || rb.RevokedIn != 5 ||
		rb.Cutoff[db.ScopeMembersCore] != 41 || rb.Cutoff[db.ScopeInventory] != 3 || len(rb.Cutoff) != 2 {
		t.Errorf("revoked entry %+v", rb)
	}
	if v5.Epochs[db.ScopeMembersCore] != 2 || v5.Epochs[db.ScopeInventory] != 2 || v5.Epochs[db.ScopeFinance] != 2 {
		t.Errorf("epochs after revocation: %v", v5.Epochs)
	}
	b.refresh()
	if b.club.Status() != StatusRevoked {
		t.Errorf("b status %s", b.club.Status())
	}
	if _, err := b.club.ScopeKey(ctx, db.ScopeMembersCore, 2); !errors.Is(err, ErrMissingKey) {
		t.Errorf("revoked device reads new epoch: %v", err)
	}
	if _, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleBoard); !errors.Is(err, ErrDeviceRevoked) {
		t.Errorf("role change of revoked device: %v", err)
	}
	// A revoked device's own ACL version, reactivating itself, is rejected.
	forged := b.club.Head().Next(b.dev.ID, hlc.Timestamp{MS: 1 << 41})
	fd := forged.Device(b.dev.ID)
	fd.Status, fd.RevokedIn, fd.Cutoff, fd.Role = acl.StatusActive, 0, nil, acl.RoleBoard
	raw, sig, _ := acl.Sign(forged, b.dev.Sign)
	if err := publish(ctx, b.tr, &acl.Version{ACL: forged, Raw: raw, Sig: sig, Sum: sha256.Sum256(raw)}); err != nil {
		t.Fatal(err)
	}
	r := a.refresh()
	if a.club.Head().ACL.Version != 5 || len(r.Rejected) != 1 || !errors.Is(r.Rejected[0].Err, acl.ErrNotAuthorized) {
		t.Errorf("forged version by revoked device: head %d, rejected %v", a.club.Head().ACL.Version, r.Rejected)
	}

	// A device opened from local state sees the same.
	re, err := Open(ctx, a.conn, a.dev, a.tr)
	if err != nil || re.Head().Sum != a.club.Head().Sum || re.Status() != StatusMember {
		t.Fatalf("Open: %v", err)
	}
}

// assertRootNotPersisted scans every file under dirs (data dir with the
// database and key file, and the sync folder) for the root private key.
func assertRootNotPersisted(t *testing.T, kit *identity.RecoveryKit, dirs ...string) {
	t.Helper()
	secrets := [][]byte{
		kit.Root.Sign.Seed(),
		[]byte(wire.EncodeB64(kit.Root.Sign.Seed())),
		[]byte(kit.Root.Age.String()),
		[]byte(strings.ToLower(kit.Root.Age.String())),
	}
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range secrets {
				if bytes.Contains(data, s) {
					t.Errorf("%s contains the root private key", p)
				}
			}
			return nil
		})
	}
}
