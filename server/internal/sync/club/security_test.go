package club

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/keys"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// requestErr returns the check result of the request of device id.
func requestErr(t *testing.T, admin *node, id string) error {
	t.Helper()
	reqs, err := admin.club.JoinRequests(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reqs {
		if r.DeviceID == id {
			return r.Err
		}
	}
	t.Fatalf("no request of %s", id)
	return nil
}

func TestInviteChecks(t *testing.T) {
	mem := transport.NewMemory()
	a := newNode(t, "a", mem)
	kit := a.create("TSV")
	kit.Wipe()

	t.Run("wrong HMAC proof", func(t *testing.T) {
		b := newNode(t, "b", mem)
		id, _, err := ParseInviteCode(a.invite(acl.RoleCoach))
		if err != nil {
			t.Fatal(err)
		}
		b.join(FormatInviteCode(id, bytes.Repeat([]byte{7}, inviteSecretSize)))
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrBadProof) {
			t.Fatalf("got %v", err)
		}
		if _, err := a.club.Approve(ctx, b.dev.ID); !errors.Is(err, ErrBadProof) {
			t.Fatalf("approve: %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("unknown invite", func(t *testing.T) {
		b := newNode(t, "b", mem)
		_, secret, _ := ParseInviteCode(a.invite(acl.RoleCoach))
		b.join(FormatInviteCode([]byte("12345"), secret))
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("expired invite", func(t *testing.T) {
		b := newNode(t, "b", mem)
		inv, err := a.club.CreateInvite(ctx, acl.RoleCoach, "", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		b.join(inv.Code)
		if err := requestErr(t, a, b.dev.ID); err != nil {
			t.Fatalf("fresh invite: %v", err)
		}
		a.club.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
		defer func() { a.club.Now = time.Now }()
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrInviteExpired) {
			t.Fatalf("got %v", err)
		}
		if _, err := a.club.Approve(ctx, b.dev.ID); !errors.Is(err, ErrInviteExpired) {
			t.Fatalf("approve: %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("used invite", func(t *testing.T) {
		b, c := newNode(t, "b", mem), newNode(t, "c", mem)
		code := a.invite(acl.RoleCoach)
		b.join(code)
		if _, err := a.club.Approve(ctx, b.dev.ID); err != nil {
			t.Fatal(err)
		}
		c.join(code)
		if err := requestErr(t, a, c.dev.ID); !errors.Is(err, ErrInviteUsed) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, c.dev.ID)
	})

	t.Run("contested invite", func(t *testing.T) {
		b, c := newNode(t, "b", mem), newNode(t, "c", mem)
		code := a.invite(acl.RoleCoach)
		b.join(code)
		c.join(code)
		for _, n := range []*node{b, c} {
			if err := requestErr(t, a, n.dev.ID); !errors.Is(err, ErrContested) {
				t.Fatalf("%s: %v", n.name, err)
			}
		}
		if _, err := a.club.Approve(ctx, b.dev.ID); !errors.Is(err, ErrContested) {
			t.Fatalf("approve: %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
		a.club.RejectRequest(ctx, c.dev.ID)
	})

	t.Run("forged request", func(t *testing.T) {
		b := newNode(t, "b", mem)
		b.join(a.invite(acl.RoleCoach))
		path := layout.RequestPath(b.dev.ID)
		raw, _ := mem.Get(ctx, path)
		mem.Put(path, bytes.Replace(raw, []byte("Person b"), []byte("Person x"), 1))
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrBadRequest) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("request under another device id", func(t *testing.T) {
		b, c := newNode(t, "b", mem), newNode(t, "c", mem)
		b.join(a.invite(acl.RoleCoach))
		raw, _ := mem.Get(ctx, layout.RequestPath(b.dev.ID))
		sig, _ := mem.Get(ctx, layout.SigPath(layout.RequestPath(b.dev.ID)))
		mem.Put(layout.RequestPath(c.dev.ID), raw)
		mem.Put(layout.SigPath(layout.RequestPath(c.dev.ID)), sig)
		if err := requestErr(t, a, c.dev.ID); !errors.Is(err, ErrBadRequest) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
		a.club.RejectRequest(ctx, c.dev.ID)
	})

	t.Run("already listed device", func(t *testing.T) {
		b := newNode(t, "b", mem)
		a.admit(b, acl.RoleCoach)
		code := a.invite(acl.RoleBoard)
		id, secret, _ := ParseInviteCode(code)
		// b re-sends a request for another invite, hoping for board.
		raw := []byte(`{"type":"ocm.join","club_id":"` + a.club.Genesis().ClubID + `","invite_id":"` + wire.EncodeB64(id) +
			`","device_id":"` + b.dev.ID + `","sign_pub":"` + b.dev.SignPubB64() + `","age_recipient":"` + b.dev.AgeRecipient() +
			`","person":"b","proof":"` + wire.EncodeB64(Proof(secret, id, b.dev.SignPub(), b.dev.AgeRecipient())) + `"}`)
		mem.Put(layout.RequestPath(b.dev.ID), raw)
		mem.Put(layout.SigPath(layout.RequestPath(b.dev.ID)), wire.Sign(b.dev.Sign, raw))
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrAlreadyListed) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("invite code of another club", func(t *testing.T) {
		// The code does not name the club: the request lands in the club of
		// the selected storage, which does not know the invite.
		other := newNode(t, "other", transport.NewMemory())
		other.create("FC").Wipe()
		b := newNode(t, "b", mem)
		b.join(other.invite(acl.RoleCoach))
		if b.club.Genesis().ClubID != a.club.Genesis().ClubID {
			t.Fatal("joined the club of the code instead of the storage")
		}
		if err := requestErr(t, a, b.dev.ID); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
		a.club.RejectRequest(ctx, b.dev.ID)
	})

	t.Run("invite codes", func(t *testing.T) {
		code := a.invite(acl.RoleCoach)
		if !regexp.MustCompile(`^[A-Z2-7]{4}(-[A-Z2-7]{4}){5}$`).MatchString(code) {
			t.Fatalf("code %q", code)
		}
		id, secret, err := ParseInviteCode(code)
		if err != nil || len(id) != 5 || len(secret) != 10 || FormatInviteCode(id, secret) != code {
			t.Fatalf("round trip: %v", err)
		}
		typed := strings.ToLower(strings.ReplaceAll(code, "-", " ")) + "\n"
		if id2, s2, err := ParseInviteCode(typed); err != nil || !bytes.Equal(id2, id) || !bytes.Equal(s2, secret) {
			t.Errorf("typed code: %v", err)
		}
		for _, bad := range []string{"", code[:len(code)-1], code + "A", strings.Replace(code, code[:1], "1", 1), strings.Replace(code, code[:1], "0", 1)} {
			if _, _, err := ParseInviteCode(bad); !errors.Is(err, ErrBadInviteCode) {
				t.Errorf("ParseInviteCode(%q): %v", bad, err)
			}
		}
		if _, err := a.club.CreateInvite(ctx, "president", "", 0); !errors.Is(err, ErrUnknownRole) {
			t.Errorf("unknown role: %v", err)
		}
	})
}

func TestRecovery(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	kit := a.create("TSV")
	kitText := kit.Text()
	kit.Wipe()
	a.admit(b, acl.RoleCoach)
	if _, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleSecretary); err != nil { // contact epoch stays 1
		t.Fatal(err)
	}
	// Another admin-only epoch change so that recovery has to unwrap
	// several epochs from the root's key files.
	c := newNode(t, "c", mem)
	a.admit(c, acl.RoleTreasurer)
	if _, err := a.club.Revoke(ctx, c.dev.ID, nil); err != nil {
		t.Fatal(err)
	}

	// All admin devices are lost; a new device recovers with the kit.
	e := newNode(t, "e", mem)
	if _, _, err := RecoverNew(ctx, e.conn, e.dev, mem, strings.Replace(kitText, "root-age-key: AGE", "root-age-key: XGE", 1), "Eva", "PC"); err == nil {
		t.Fatal("damaged kit accepted")
	}
	ce, v, err := RecoverNew(ctx, e.conn, e.dev, mem, kitText, "Eva", "PC")
	if err != nil {
		t.Fatal(err)
	}
	e.club = ce
	if v.Author != acl.AuthorRoot || !v.IsAdmin(e.dev.ID) {
		t.Fatalf("recovery version: author %s", v.Author)
	}
	for _, s := range db.Scopes {
		for ep := uint64(1); ep <= v.Epochs[s]; ep++ {
			if !bytes.Equal(e.key(s, ep), a.key(s, ep)) {
				t.Errorf("recovered key %s/%d differs", s, ep)
			}
		}
	}
	assertNoRootKey(t, kitText, e.dir)

	// The recovered admin revokes the lost admin device.
	if _, err := e.club.Revoke(ctx, a.dev.ID, nil); err != nil {
		t.Fatal(err)
	}
	a.refresh()
	if a.club.Status() != StatusRevoked {
		t.Errorf("a status %s", a.club.Status())
	}

	// b uses the kit on its existing device and becomes admin.
	if _, err := b.club.Recover(ctx, kitText, "", ""); err != nil {
		t.Fatal(err)
	}
	if !b.club.Head().IsAdmin(b.dev.ID) || b.club.Head().Device(b.dev.ID).Role != acl.RoleBoard {
		t.Error("b is not admin after recovery")
	}
	b.key(db.ScopeFinance, b.club.Head().Epochs[db.ScopeFinance])

	// A revoked device cannot recover itself (§8.2 rule 5).
	if _, err := a.club.Recover(ctx, kitText, "", ""); !errors.Is(err, ErrDeviceRevoked) {
		t.Errorf("revoked device recovered: %v", err)
	}

	// Root revocation without any admin device.
	if _, err := b.club.RevokeWithRoot(ctx, kitText, e.dev.ID, nil); err != nil {
		t.Fatal(err)
	}

	// A kit of another club is refused.
	other := newNode(t, "o", transport.NewMemory())
	okit := other.create("FC")
	otherText := okit.Text()
	okit.Wipe()
	if _, err := b.club.Recover(ctx, otherText, "", ""); !errors.Is(err, ErrKitMismatch) {
		t.Errorf("other club's kit: %v", err)
	}
}

func assertNoRootKey(t *testing.T, kitText, dir string) {
	t.Helper()
	k, err := identity.ParseRecoveryKit(kitText)
	if err != nil {
		t.Fatal(err)
	}
	assertRootNotPersisted(t, k, dir)
}

func TestRollbackAndTampering(t *testing.T) {
	shared := t.TempDir()
	a, b := newNode(t, "a", folder(t, shared)), newNode(t, "b", folder(t, shared))
	a.create("TSV").Wipe()
	a.admit(b, acl.RoleTreasurer)
	if _, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleSecretary); err != nil {
		t.Fatal(err)
	}
	b.refresh()
	head := b.club.Head()
	if head.ACL.Version != 3 {
		t.Fatalf("head %d", head.ACL.Version)
	}

	// Someone deletes the newest ACL files: the local cache keeps the chain.
	os.Remove(filepath.Join(shared, filepath.FromSlash(head.Path())))
	os.Remove(filepath.Join(shared, filepath.FromSlash(layout.SigPath(head.Path()))))
	if b.refresh(); b.club.Head().Sum != head.Sum {
		t.Fatal("deleting files rolled the chain back")
	}
	// Without the cache, the chain ends below the accepted version (§8.4).
	b.conn.Exec(`DELETE FROM sync_acl`)
	if _, err := b.club.Refresh(ctx); !errors.Is(err, acl.ErrRollback) {
		t.Fatalf("rollback: %v", err)
	}
	if v, _ := accepted(b); v != 3 {
		t.Errorf("accepted version changed to %d", v)
	}
	// The file comes back (e.g. the sync client restores it).
	os.WriteFile(filepath.Join(shared, filepath.FromSlash(head.Path())), head.Raw, 0o600)
	os.WriteFile(filepath.Join(shared, filepath.FromSlash(layout.SigPath(head.Path()))), head.Sig, 0o600)
	b.refresh()

	// A swapped key file is detected by the key check.
	attacker, _ := keys.New()
	w, _ := keys.Wrap(attacker, b.dev.AgeRecipient())
	keyPath := filepath.Join(shared, filepath.FromSlash(layout.KeyPath(db.ScopeMembersCore, 1, b.dev.ID)))
	if err := os.WriteFile(keyPath, w, 0o600); err != nil {
		t.Fatal(err)
	}
	b.conn.Exec(`DELETE FROM sync_keys`)
	r := b.refresh()
	if !slices.Contains(r.BadKeys, KeyRef{db.ScopeMembersCore, 1}) {
		t.Errorf("swapped key not reported: %+v", r)
	}
	if _, err := b.club.ScopeKey(ctx, db.ScopeMembersCore, 1); !errors.Is(err, ErrMissingKey) {
		t.Errorf("swapped key used: %v", err)
	}

	// Sync-client conflict copies and temp files in acl/ are ignored.
	os.WriteFile(filepath.Join(shared, "acl", "00000004-0000000000000000 (conflicted copy).json"), []byte("{}"), 0o600)
	os.WriteFile(filepath.Join(shared, "acl", ".tmp-1234"), []byte("{}"), 0o600)
	if r := b.refresh(); len(r.Rejected) != 0 {
		t.Errorf("rejections for ignored files: %v", r.Rejected)
	}

	// A replaced club.json (another root) is refused (§7).
	other := newNode(t, "o", transport.NewMemory())
	other.create("FC").Wipe()
	g, _ := other.tr.Get(ctx, layout.Genesis)
	gs, _ := other.tr.Get(ctx, layout.GenesisSig)
	os.WriteFile(filepath.Join(shared, layout.Genesis), g, 0o600)
	os.WriteFile(filepath.Join(shared, layout.GenesisSig), gs, 0o600)
	if _, err := b.club.Refresh(ctx); !errors.Is(err, acl.ErrGenesisMismatch) {
		t.Errorf("replaced club.json: %v", err)
	}
	// So is an unsigned edit.
	edited := bytes.Replace(pinnedGenesis(t, a), []byte(`"TSV"`), []byte(`"XYZ"`), 1)
	os.WriteFile(filepath.Join(shared, layout.Genesis), edited, 0o600)
	os.WriteFile(filepath.Join(shared, layout.GenesisSig), pinnedGenesisSig(t, a), 0o600)
	if _, err := b.club.Refresh(ctx); !errors.Is(err, acl.ErrBadGenesis) {
		t.Errorf("edited club.json: %v", err)
	}
}

func accepted(n *node) (uint64, string) {
	var v uint64
	var h string
	n.conn.QueryRow(`SELECT acl_version, acl_hash FROM sync_club`).Scan(&v, &h)
	return v, h
}

func pinnedGenesis(t *testing.T, n *node) []byte {
	var b []byte
	if err := n.conn.QueryRow(`SELECT genesis FROM sync_club`).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

func pinnedGenesisSig(t *testing.T, n *node) []byte {
	var b []byte
	if err := n.conn.QueryRow(`SELECT genesis_sig FROM sync_club`).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

// gate is a transport that hides other devices' new files until opened, to
// simulate two admin devices editing the ACL while their sync clients have
// not exchanged files yet.
type gate struct {
	transport.Transport
	mu      sync.Mutex
	visible map[string]bool // nil: everything visible
}

func (g *gate) close(ctx context.Context) {
	files, _ := g.Transport.List(ctx, "")
	g.mu.Lock()
	defer g.mu.Unlock()
	g.visible = map[string]bool{}
	for _, f := range files {
		g.visible[f.Path] = true
	}
}

func (g *gate) open() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.visible = nil
}

func (g *gate) List(ctx context.Context, prefix string) ([]transport.FileInfo, error) {
	files, err := g.Transport.List(ctx, prefix)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.visible == nil {
		return files, err
	}
	var out []transport.FileInfo
	for _, f := range files {
		if g.visible[f.Path] {
			out = append(out, f)
		}
	}
	return out, err
}

func (g *gate) Create(ctx context.Context, path string, data []byte) error {
	err := g.Transport.Create(ctx, path, data)
	g.mu.Lock()
	defer g.mu.Unlock()
	if err == nil && g.visible != nil {
		g.visible[path] = true
	}
	return err
}

func TestConcurrentAdmins(t *testing.T) {
	mem := transport.NewMemory()
	ga, gc := &gate{Transport: mem}, &gate{Transport: mem}
	a, b, c := newNode(t, "a", ga), newNode(t, "b", mem), newNode(t, "c", gc)
	a.create("TSV").Wipe()
	a.admit(b, acl.RoleCoach)
	a.admit(c, acl.RoleBoard)
	c.refresh()

	// Both admins change b's role from the same version 3, unaware of each
	// other. Both scopes changes are upgrades, so no epochs collide.
	ga.close(ctx)
	gc.close(ctx)
	va, err := a.club.ChangeRole(ctx, b.dev.ID, acl.RoleTreasurer)
	if err != nil {
		t.Fatal(err)
	}
	vc, err := c.club.ChangeRole(ctx, b.dev.ID, acl.RoleSecretary)
	if err != nil {
		t.Fatal(err)
	}
	if va.ACL.Version != 4 || vc.ACL.Version != 4 {
		t.Fatalf("versions %d and %d", va.ACL.Version, vc.ACL.Version)
	}
	ga.open()
	gc.open()

	winner, loser, loserNode := va, vc, c
	if bytes.Compare(vc.Sum[:], va.Sum[:]) < 0 {
		winner, loser, loserNode = vc, va, a
	}
	ra, rc := a.refresh(), c.refresh()
	b.refresh()
	for _, n := range []*node{a, b, c} {
		if n.club.Head().Sum != winner.Sum {
			t.Errorf("%s follows another chain", n.name)
		}
	}
	lostReport := map[*node]*RefreshResult{a: ra, c: rc}[loserNode]
	if len(lostReport.LostOwn) != 1 || lostReport.LostOwn[0].Sum != loser.Sum || !lostReport.Reorged {
		t.Errorf("loser not told: %+v", lostReport)
	}
	winnerReport := map[*node]*RefreshResult{a: rc, c: ra}[loserNode]
	if len(winnerReport.LostOwn) != 0 || winnerReport.Reorged {
		t.Errorf("winner report %+v", winnerReport)
	}
	if b.club.Head().Device(b.dev.ID).Role != winner.Device(b.dev.ID).Role {
		t.Error("b has the losing role")
	}

	// The loser redoes its change on top of the winner.
	role := loser.Device(b.dev.ID).Role
	v5, err := loserNode.club.ChangeRole(ctx, b.dev.ID, role)
	if err != nil {
		t.Fatal(err)
	}
	if v5.ACL.Version != 5 || *v5.Prev != winner.Hash() {
		t.Errorf("redo: version %d", v5.ACL.Version)
	}
}

func TestNewEpochKeyConflict(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()
	a.admit(b, acl.RoleTreasurer)

	// Someone with folder access plants a's key file of the next finance
	// epoch, wrapped to a's public key with a key they know.
	planted, _ := keys.New()
	w, _ := keys.Wrap(planted, a.dev.AgeRecipient())
	mem.Put(layout.KeyPath(db.ScopeFinance, 2, a.dev.ID), w)
	if _, err := a.club.Revoke(ctx, b.dev.ID, nil); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("got %v", err)
	}
	if a.club.Head().ACL.Version != 2 {
		t.Fatal("ACL published despite the conflict")
	}
	// A retry reuses the drafted key; the planted file stays and is
	// detected by the key check instead of being trusted.
	v, err := a.club.Revoke(ctx, b.dev.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	k, err := a.club.ScopeKey(ctx, db.ScopeFinance, 2)
	if err == nil && bytes.Equal(k, planted) {
		t.Fatal("planted key accepted")
	}
	if keys.Verify(db.ScopeFinance, 2, planted, v.KeyChecks[db.ScopeFinance][1]) == nil {
		t.Fatal("ACL commits to the planted key")
	}
	var drafts int
	a.conn.QueryRow(`SELECT COUNT(*) FROM sync_key_drafts`).Scan(&drafts)
	if drafts != 0 {
		t.Errorf("%d drafts left after the epoch was accepted", drafts)
	}
}
