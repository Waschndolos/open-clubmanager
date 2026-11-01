package club

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// inClub reports whether n has local club state.
func (n *node) inClub() bool {
	n.t.Helper()
	var c int
	if err := n.conn.QueryRow(`SELECT COUNT(*) FROM sync_club`).Scan(&c); err != nil {
		n.t.Fatal(err)
	}
	return c > 0
}

func exists(t *testing.T, tr transport.Transport, path string) bool {
	t.Helper()
	_, err := tr.Get(ctx, path)
	if err != nil && !errors.Is(err, transport.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

func TestJoinChecksInviteMarker(t *testing.T) {
	mem := transport.NewMemory()
	a := newNode(t, "a", mem)
	a.create("TSV").Wipe()

	tryJoin := func(code string) error {
		b := newNode(t, "b", mem)
		_, _, err := Join(ctx, b.conn, b.dev, b.tr, code, "B", "Laptop")
		if err != nil && (b.inClub() || exists(t, mem, layout.RequestPath(b.dev.ID))) {
			t.Fatalf("refused join left state behind")
		}
		return err
	}

	t.Run("unknown code", func(t *testing.T) {
		code := FormatInviteCode(bytes.Repeat([]byte{1}, inviteIDSize), bytes.Repeat([]byte{2}, inviteSecretSize))
		if err := tryJoin(code); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("typo in the secret", func(t *testing.T) {
		id, _, _ := ParseInviteCode(a.invite(acl.RoleCoach))
		if err := tryJoin(FormatInviteCode(id, bytes.Repeat([]byte{7}, inviteSecretSize))); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("code of another club", func(t *testing.T) {
		other := newNode(t, "other", transport.NewMemory())
		other.create("FC").Wipe()
		if err := tryJoin(other.invite(acl.RoleCoach)); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("withdrawn invite", func(t *testing.T) {
		inv, err := a.club.CreateInvite(ctx, acl.RoleCoach, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.club.WithdrawInvite(ctx, wire.EncodeB64(inv.ID)); err != nil {
			t.Fatal(err)
		}
		if err := tryJoin(inv.Code); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("used invite", func(t *testing.T) {
		code := a.invite(acl.RoleCoach)
		b := newNode(t, "b", mem)
		b.join(code)
		if _, err := a.club.Approve(ctx, b.dev.ID); err != nil {
			t.Fatal(err)
		}
		if err := tryJoin(code); !errors.Is(err, ErrInviteUsed) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("expired invite", func(t *testing.T) {
		inv, err := a.club.CreateInvite(ctx, acl.RoleCoach, "", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkInviteMarker(ctx, mem, a.club.Genesis(), inv.ID, inv.Secret, time.Now().Add(2*time.Hour)); !errors.Is(err, ErrInviteExpired) {
			t.Fatalf("expired by the joining device's clock: %v", err)
		}
		// Once the admin sees the invite expired, the marker goes away.
		a.club.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
		defer func() { a.club.Now = time.Now }()
		a.refresh()
		if err := tryJoin(inv.Code); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("marker published by refresh", func(t *testing.T) {
		// Invites created before markers existed get one with the next refresh.
		inv, err := a.club.CreateInvite(ctx, acl.RoleCoach, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		mem.Delete(ctx, layout.InvitePath(InviteMarker(inv.ID, inv.Secret)))
		if err := tryJoin(inv.Code); !errors.Is(err, ErrInviteUnknown) {
			t.Fatalf("without marker: %v", err)
		}
		a.refresh()
		if err := tryJoin(inv.Code); err != nil {
			t.Fatalf("after refresh: %v", err)
		}
	})

	t.Run("marker names neither id nor secret", func(t *testing.T) {
		inv, err := a.club.CreateInvite(ctx, acl.RoleCoach, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		path := layout.InvitePath(InviteMarker(inv.ID, inv.Secret))
		raw, err := mem.Get(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range [][]byte{inv.ID, inv.Secret, []byte(wire.EncodeB64(inv.ID)), []byte(wire.EncodeB64(inv.Secret))} {
			if bytes.Contains(raw, secret) || bytes.Contains([]byte(path), secret) {
				t.Fatalf("marker %s leaks invite data: %s", path, raw)
			}
		}
	})
}

func TestJoinRejected(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()

	b.join(a.invite(acl.RoleCoach))
	b.refresh()
	if b.club.Status() != StatusJoining {
		t.Fatalf("status %s", b.club.Status())
	}
	if err := a.club.RejectRequest(ctx, b.dev.ID); err != nil {
		t.Fatal(err)
	}
	if exists(t, mem, layout.RequestPath(b.dev.ID)) || !exists(t, mem, layout.RejectionPath(b.dev.ID)) {
		t.Fatal("rejection did not replace the request by a rejection notice")
	}
	if _, err := b.club.Refresh(ctx); !errors.Is(err, ErrJoinRejected) {
		t.Fatalf("refresh after rejection: %v", err)
	}
	if b.inClub() || exists(t, mem, layout.RejectionPath(b.dev.ID)) {
		t.Fatal("rejected device kept its club state or the notice")
	}
	if _, err := Open(ctx, b.conn, b.dev, mem); !errors.Is(err, ErrNoClub) {
		t.Fatalf("open after rejection: %v", err)
	}
	if _, err := b.club.Refresh(ctx); !errors.Is(err, ErrNoClub) {
		t.Fatalf("refresh of the left club: %v", err)
	}

	// A new attempt works.
	a.admit(b, acl.RoleCoach)
}

func TestJoinRejectionMustBeGenuine(t *testing.T) {
	mem := transport.NewMemory()
	a, m, b := newNode(t, "a", mem), newNode(t, "m", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()
	a.admit(m, acl.RoleCoach)
	b.join(a.invite(acl.RoleCoach))

	var inviteID string
	if err := b.conn.QueryRow(`SELECT join_invite_id FROM sync_club`).Scan(&inviteID); err != nil {
		t.Fatal(err)
	}
	put := func(signer *node, r joinRejection) {
		raw, _ := json.MarshalIndent(r, "", "  ")
		mem.Put(layout.RejectionPath(b.dev.ID), raw)
		mem.Put(layout.SigPath(layout.RejectionPath(b.dev.ID)), wire.Sign(signer.dev.Sign, raw))
	}
	genuine := joinRejection{Type: rejectionType, ClubID: a.club.Genesis().ClubID, DeviceID: b.dev.ID, InviteID: inviteID, By: a.dev.ID}
	forgeries := map[string]func(){
		"signed by a non-admin member": func() { r := genuine; r.By = m.dev.ID; put(m, r) },
		"signed by the device itself":  func() { r := genuine; r.By = b.dev.ID; put(b, r) },
		"claims an admin, wrong key":   func() { put(m, genuine) },
		"for another invite":           func() { r := genuine; r.InviteID = wire.EncodeB64([]byte("12345")); put(a, r) },
		"for another club":             func() { r := genuine; r.ClubID = "other"; put(a, r) },
	}
	for name, forge := range forgeries {
		forge()
		if _, err := b.club.Refresh(ctx); err != nil || b.club.Status() != StatusJoining || !b.inClub() {
			t.Fatalf("%s: refresh %v, status %s", name, err, b.club.Status())
		}
	}
	put(a, genuine)
	if _, err := b.club.Refresh(ctx); !errors.Is(err, ErrJoinRejected) {
		t.Fatalf("genuine rejection: %v", err)
	}
}

func TestCancelJoin(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()

	b.join(a.invite(acl.RoleCoach))
	b.refresh()
	if err := b.club.CancelJoin(ctx); err != nil {
		t.Fatal(err)
	}
	if b.inClub() || exists(t, mem, layout.RequestPath(b.dev.ID)) || exists(t, mem, layout.SigPath(layout.RequestPath(b.dev.ID))) {
		t.Fatal("cancel left the request or local state behind")
	}
	if reqs, err := a.club.JoinRequests(ctx); err != nil || len(reqs) != 0 {
		t.Fatalf("admin still sees requests: %+v, %v", reqs, err)
	}
	if err := b.club.CancelJoin(ctx); !errors.Is(err, ErrNotJoining) {
		t.Fatalf("second cancel: %v", err)
	}

	// Joining again with another code works.
	a.admit(b, acl.RoleTreasurer)
	if err := b.club.CancelJoin(ctx); !errors.Is(err, ErrNotJoining) {
		t.Fatalf("cancel as member: %v", err)
	}

	// An approval that arrived before the cancel wins.
	c := newNode(t, "c", mem)
	c.join(a.invite(acl.RoleCoach))
	if _, err := a.club.Approve(ctx, c.dev.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.club.CancelJoin(ctx); !errors.Is(err, ErrNotJoining) || c.club.Status() != StatusMember {
		t.Fatalf("cancel after approval: %v, status %s", err, c.club.Status())
	}
}

func TestAbandonJoin(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()
	if err := AbandonJoin(ctx, a.conn); !errors.Is(err, ErrNotJoining) {
		t.Fatalf("abandon as member: %v", err)
	}
	if err := AbandonJoin(ctx, b.conn); !errors.Is(err, ErrNotJoining) {
		t.Fatalf("abandon without club: %v", err)
	}
	b.join(a.invite(acl.RoleCoach))
	b.refresh()
	if err := AbandonJoin(ctx, b.conn); err != nil || b.inClub() {
		t.Fatalf("abandon: %v", err)
	}
	// The request left behind is replaced by the next attempt.
	b.join(a.invite(acl.RoleCoach))
	if reqs, err := a.club.JoinRequests(ctx); err != nil || len(reqs) != 1 || reqs[0].Err != nil {
		t.Fatalf("requests: %+v, %v", reqs, err)
	}
}
