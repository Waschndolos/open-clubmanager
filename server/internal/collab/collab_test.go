package collab

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

type node struct {
	svc  *Service
	conn *sql.DB
	path string
}

func newNode(t *testing.T) *node {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "clubmanager.db")
	conn, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	n := &node{conn: conn, path: path}
	n.svc = New(func() *sql.DB { return n.conn }, func() string { return n.path })
	n.svc.WorkFactor = 10
	return n
}

func (n *node) status(t *testing.T) *Status {
	t.Helper()
	st, err := n.svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func (n *node) sync(t *testing.T) *Status {
	t.Helper()
	st, err := n.svc.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestCreateInviteJoinApprove(t *testing.T) {
	ctx := context.Background()
	shared := t.TempDir()
	a, b := newNode(t), newNode(t)

	if st := a.status(t); st.Mode != "local" || st.Permissions[db.ScopeMembersBank] != PermWrite {
		t.Fatalf("local status = %+v", st)
	}

	res, err := a.svc.Create(ctx, shared, "TSV Beispielstadt e.V.", "Petra Schmidt", "Laptop", "pw-a")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.RecoveryKit, "OPEN CLUBMANAGER RECOVERY KIT") {
		t.Fatalf("kit = %q", res.RecoveryKit)
	}
	if want := filepath.Join(shared, "Open ClubManager – TSV Beispielstadt e.V"); res.Path != want {
		t.Fatalf("path = %q, want %q", res.Path, want)
	}
	st := a.status(t)
	if st.Mode != "member" || !st.IsAdmin || st.Role != acl.RoleBoard || st.DeviceName != "Laptop" || st.ClubName != "TSV Beispielstadt e.V." {
		t.Fatalf("creator status = %+v", st)
	}
	if _, err := a.svc.Create(ctx, shared, "TSV Beispielstadt e.V.", "x", "y", "pw-a"); err == nil {
		t.Fatal("second create on the same device succeeded")
	}

	// The shared folder is offered when joining, with the club inside.
	info, err := CheckFolder(shared)
	if err != nil || len(info.ClubsInside) != 1 || info.ClubsInside[0].Name != "TSV Beispielstadt e.V." || !info.Writable {
		t.Fatalf("CheckFolder = %+v, %v", info, err)
	}

	inv, err := a.svc.CreateInvite(ctx, acl.RoleTreasurer, "Erika Mustermann")
	if err != nil {
		t.Fatal(err)
	}
	safety, err := b.svc.Join(ctx, shared, strings.ToLower(inv.Code), "Erika Mustermann", "Laptop", "pw-b", false)
	if err != nil {
		t.Fatal(err)
	}
	if st := b.status(t); st.Mode != "joining" || st.Permissions[db.ScopeMembersCore] != PermNone {
		t.Fatalf("joining status = %+v", st)
	}
	if got, _ := b.svc.SafetyCode(); got != safety {
		t.Fatalf("SafetyCode = %q, want %q", got, safety)
	}

	if st := a.sync(t); st.PendingRequests != 1 {
		t.Fatalf("pending requests = %d", st.PendingRequests)
	}
	reqs := a.svc.Requests()
	if len(reqs) != 1 || reqs[0].SafetyCode != "" || reqs[0].Person != "Erika Mustermann" || reqs[0].DeviceName != "Laptop" || reqs[0].Role != acl.RoleTreasurer {
		t.Fatalf("requests = %+v", reqs)
	}
	wrong := "000000"
	if safety == wrong {
		wrong = "111111"
	}
	if err := a.svc.Approve(ctx, reqs[0].DeviceID, wrong); !errors.Is(err, ErrWrongSafetyCode) {
		t.Fatalf("wrong code: %v", err)
	}
	if err := a.svc.Approve(ctx, reqs[0].DeviceID, safety[:3]+" "+safety[3:]); err != nil {
		t.Fatal(err)
	}

	st = b.sync(t)
	if st.Mode != "member" || st.Role != acl.RoleTreasurer || st.IsAdmin ||
		st.Permissions[db.ScopeMembersBank] != PermWrite || st.Permissions[db.ScopeMembersCore] != PermRead ||
		st.Permissions[db.ScopeAdmin] != PermNone {
		t.Fatalf("member status = %+v", st)
	}
	if b.svc.CanWrite(ctx, db.ScopeMembersCore) || !b.svc.CanWrite(ctx, db.ScopeFinance) {
		t.Fatal("CanWrite does not follow the role")
	}
	if _, err := b.svc.CreateInvite(ctx, acl.RoleCoach, "x"); !errors.Is(err, club.ErrNotAdmin) {
		t.Fatalf("invite by non-admin: %v", err)
	}

	devs, err := a.svc.Devices()
	if err != nil || len(devs) != 2 {
		t.Fatalf("devices = %+v, %v", devs, err)
	}
	if err := a.svc.Revoke(ctx, st.DeviceID); err != nil {
		t.Fatal(err)
	}
	if err := a.svc.Revoke(ctx, a.status(t).DeviceID); !errors.Is(err, ErrOwnDevice) {
		t.Fatalf("revoke own device: %v", err)
	}
	if st := b.sync(t); st.Mode != "revoked" {
		t.Fatalf("revoked status = %+v", st)
	}
}

func TestTooManyWrongCodesRejects(t *testing.T) {
	ctx := context.Background()
	shared := t.TempDir()
	a, b := newNode(t), newNode(t)
	if _, err := a.svc.Create(ctx, shared, "Club", "A", "PC", "pw"); err != nil {
		t.Fatal(err)
	}
	inv, err := a.svc.CreateInvite(ctx, acl.RoleCoach, "B")
	if err != nil {
		t.Fatal(err)
	}
	safety, err := b.svc.Join(ctx, shared, inv.Code, "B", "PC", "pw", false)
	if err != nil {
		t.Fatal(err)
	}
	a.sync(t)
	id := a.svc.Requests()[0].DeviceID
	wrong := "123456"
	if wrong == safety {
		wrong = "654321"
	}
	for i := 1; i < MaxApproveAttempts; i++ {
		if err := a.svc.Approve(ctx, id, wrong); !errors.Is(err, ErrWrongSafetyCode) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := a.svc.Approve(ctx, id, wrong); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("last attempt: %v", err)
	}
	if st := a.sync(t); st.PendingRequests != 0 {
		t.Fatalf("request still pending after rejection: %+v", st)
	}
	if err := a.svc.Approve(ctx, id, safety); !errors.Is(err, club.ErrNoRequest) {
		t.Fatalf("approve after rejection: %v", err)
	}
}

func TestUnlockAfterRestart(t *testing.T) {
	ctx := context.Background()
	shared := t.TempDir()
	a := newNode(t)
	if _, err := a.svc.Create(ctx, shared, "Club", "A", "PC", "secret"); err != nil {
		t.Fatal(err)
	}

	// A new service on the same database, as after an app restart.
	restarted := New(func() *sql.DB { return a.conn }, func() string { return a.path })
	restarted.WorkFactor = 10
	st, err := restarted.Status(ctx)
	if err != nil || !st.Locked || st.Mode != "member" {
		t.Fatalf("status before unlock = %+v, %v", st, err)
	}
	if err := restarted.Unlock(ctx, "wrong"); err != nil {
		t.Fatal(err)
	}
	if st, _ := restarted.Status(ctx); !st.Locked {
		t.Fatal("unlocked with a wrong password")
	}
	if _, err := restarted.CreateInvite(ctx, acl.RoleCoach, "x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("invite while locked: %v", err)
	}
	if err := restarted.Unlock(ctx, "secret"); err != nil {
		t.Fatal(err)
	}
	st, err = restarted.Status(ctx)
	if err != nil || st.Locked || !st.IsAdmin || st.StoragePath == "" {
		t.Fatalf("status after unlock = %+v, %v", st, err)
	}
	if _, err := restarted.CreateInvite(ctx, acl.RoleCoach, "x"); err != nil {
		t.Fatal(err)
	}
}

func TestInviteWithdraw(t *testing.T) {
	ctx := context.Background()
	a, b := newNode(t), newNode(t)
	shared := t.TempDir()
	if _, err := a.svc.Create(ctx, shared, "Club", "A", "PC", "pw"); err != nil {
		t.Fatal(err)
	}
	inv, err := a.svc.CreateInvite(ctx, acl.RoleCoach, "B")
	if err != nil {
		t.Fatal(err)
	}
	invs, _ := a.svc.Invites(ctx)
	if len(invs) != 1 {
		t.Fatalf("invites = %d", len(invs))
	}
	if err := a.svc.WithdrawInvite(ctx, encodeID(invs[0].ID)); err != nil {
		t.Fatal(err)
	}
	if err := a.svc.WithdrawInvite(ctx, encodeID(invs[0].ID)); !errors.Is(err, club.ErrNoInvite) {
		t.Fatalf("second withdraw: %v", err)
	}
	if _, err := b.svc.Join(ctx, shared, inv.Code, "B", "PC", "pw", false); err != nil {
		t.Fatal(err)
	}
	a.sync(t)
	if r := a.svc.Requests(); len(r) != 1 || !errors.Is(r[0].Err, club.ErrInviteUnknown) {
		t.Fatalf("request with withdrawn invite = %+v", r)
	}
	if st := a.status(t); st.PendingRequests != 0 {
		t.Fatalf("invalid request counted as pending: %+v", st)
	}
}

func TestJoinWithLocalData(t *testing.T) {
	ctx := context.Background()
	a, b := newNode(t), newNode(t)
	shared := t.TempDir()
	if _, err := a.svc.Create(ctx, shared, "Club", "A", "PC", "pw"); err != nil {
		t.Fatal(err)
	}
	inv, err := a.svc.CreateInvite(ctx, acl.RoleCoach, "B")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.conn.Exec(`INSERT INTO groups (id, name, created_at, updated_at) VALUES ('g1', 'Jugend', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if has, err := b.svc.HasLocalData(ctx); err != nil || !has {
		t.Fatalf("HasLocalData = %v, %v", has, err)
	}
	if _, err := b.svc.Join(ctx, shared, inv.Code, "B", "PC", "pw", false); !errors.Is(err, club.ErrLocalData) {
		t.Fatalf("join with local data: %v", err)
	}
	if _, err := b.svc.Join(ctx, shared, inv.Code, "B", "PC", "pw", true); err != nil {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(filepath.Join(filepath.Dir(b.path), "clubmanager-vor-beitritt-*.db"))
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	if has, _ := b.svc.HasLocalData(ctx); has {
		t.Fatal("local data not cleared")
	}
}

func TestCheckFolderKinds(t *testing.T) {
	dir := t.TempDir()
	nc := filepath.Join(dir, "Nextcloud", "Verein")
	if err := os.MkdirAll(nc, 0o700); err != nil {
		t.Fatal(err)
	}
	info, err := CheckFolder(nc)
	if err != nil || info.Kind != KindSync || info.Provider != "Nextcloud" || !info.Writable || info.Club != nil {
		t.Fatalf("nextcloud = %+v, %v", info, err)
	}
	info, err = CheckFolder(dir)
	if err != nil || info.Kind != KindLocal {
		t.Fatalf("local = %+v, %v", info, err)
	}
	if _, err := CheckFolder(filepath.Join(dir, "missing")); !errors.Is(err, ErrFolderMissing) {
		t.Fatalf("missing: %v", err)
	}
	if got := ClubFolderName(` TSV "Nord"/Süd. `); got != "Open ClubManager – TSV -Nord--Süd" {
		t.Fatalf("ClubFolderName = %q", got)
	}
}

func encodeID(b []byte) string { return wire.EncodeB64(b) }
