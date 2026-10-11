package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

func strp(s string) *string { return &s }

// Two devices on a shared folder: data from local mode reaches the joining
// device through the initial snapshot (§15.4), later changes through
// segments, and concurrent edits of different fields of the same row merge.
func TestTwoDevicesFolder(t *testing.T) {
	shared := t.TempDir()
	a := newNode(t, "a", folder(t, shared))
	b := newNode(t, "b", folder(t, shared))

	pre := a.member(1, "Erika", "Mustermann") // written in local mode
	a.create()
	if n := a.count(`SELECT COUNT(*) FROM sync_uploads`); n != 0 {
		t.Fatalf("%d initial snapshots not uploaded", n)
	}
	for _, s := range db.Scopes {
		files, _ := a.tr.List(ctx, "snapshots/"+string(s)+"/")
		if len(files) != 1 {
			t.Fatalf("snapshots of %s: %v", s, files)
		}
	}
	a.admit(b, acl.RoleBoard)
	rep := b.sync()
	if len(rep.Loaded) != len(db.Scopes) || len(rep.Incomplete) != 0 {
		t.Fatalf("b loaded %v, incomplete %v", rep.Loaded, rep.Incomplete)
	}
	if got, err := b.store.GetMember(ctx, pre.ID); err != nil || got.FirstName != "Erika" {
		t.Fatalf("pre-club member on b: %+v, %v", got, err)
	}
	converged(t, nil, a, b)

	// Concurrent edits of different fields of the same member.
	in := store.MemberInput{Number: 1, FirstName: "Erika", LastName: "Mustermann"}
	inA := in
	inA.Comment = strp("from a")
	if _, err := a.store.UpdateMember(ctx, pre.ID, inA); err != nil {
		t.Fatal(err)
	}
	inB := in
	inB.LastName = "Musterfrau"
	inB.Contact.Email = strp("erika@example.org")
	if _, err := b.store.UpdateMember(ctx, pre.ID, inB); err != nil {
		t.Fatal(err)
	}
	a.member(2, "Max", "Muster")
	syncAll(a, b)
	converged(t, nil, a, b)
	got, _ := a.store.GetMember(ctx, pre.ID)
	if got.LastName != "Musterfrau" || got.Comment == nil || *got.Comment != "from a" ||
		got.Contact.Email == nil || *got.Contact.Email != "erika@example.org" {
		t.Errorf("merged member %+v", got)
	}
	if n := a.count(`SELECT COUNT(*) FROM sync_outbox`) + b.count(`SELECT COUNT(*) FROM sync_outbox`); n != 0 {
		t.Errorf("%d ops left in outboxes", n)
	}
	if n := a.count(`SELECT COUNT(*) FROM sync_history WHERE device_id = ?`, b.dev.ID); n == 0 {
		t.Error("no history of b's changes on a")
	}
	// Segments are named, chained and numbered as §6/§11 require.
	files, _ := a.tr.List(ctx, "segments/"+b.dev.ID+"/members.core/")
	if len(files) != 1 || files[0].Path != layout.SegmentPath(b.dev.ID, db.ScopeMembersCore, 1) {
		t.Errorf("b's core segments: %v", files)
	}
	if st, seq := a.stream(b.dev.ID, db.ScopeMembersCore); st != StatusOK || seq != 1 {
		t.Errorf("stream on a: %s %d", st, seq)
	}
}

// Three devices on transport.Memory converge after concurrent edits of the
// same field (last writer wins), inserts and deletes.
func TestThreeDevicesMemory(t *testing.T) {
	mem := transport.NewMemory()
	a, b, c := newNode(t, "a", mem), newNode(t, "b", mem), newNode(t, "c", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	a.admit(c, acl.RoleSecretary)
	syncAll(a, b, c)

	m := a.member(10, "Anna", "A")
	syncAll(a, b, c)
	for i, n := range []*node{a, b, c} {
		in := store.MemberInput{Number: 10, FirstName: "Anna", LastName: "A", Comment: strp("comment " + n.name)}
		if i == 1 {
			time.Sleep(2 * time.Millisecond)
		}
		if _, err := n.store.UpdateMember(ctx, m.ID, in); err != nil {
			t.Fatal(err)
		}
		n.member(100+i, "New", n.name)
	}
	syncAll(a, b, c)
	converged(t, nil, a, b, c)
	// The newest write wins; c wrote last.
	got, _ := b.store.GetMember(ctx, m.ID)
	if got.Comment == nil || *got.Comment != "comment c" {
		t.Errorf("comment %v", got.Comment)
	}
	if _, total, _ := b.store.ListMembers(ctx, store.MemberFilter{Limit: 100}); total != 4 {
		t.Errorf("b has %d members", total)
	}
}

// Equal HLCs on two devices are ordered by device id (§12).
func TestLWWTie(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	m := a.member(1, "Tie", "Break")
	syncAll(a, b)

	ms := futureMS(10 * time.Minute)
	for _, n := range []*node{a, b} {
		n.setHLC(ms, 5)
		if _, err := n.store.UpdateMember(ctx, m.ID, store.MemberInput{Number: 1, FirstName: "by " + n.name, LastName: "Break"}); err != nil {
			t.Fatal(err)
		}
	}
	clocks := a.rows(`SELECT hlc FROM sync_field_clock WHERE col = 'first_name'`) + b.rows(`SELECT hlc FROM sync_field_clock WHERE col = 'first_name'`)
	if lines := strings.Fields(clocks); len(lines) != 2 || lines[0] != lines[1] {
		t.Fatalf("no tie: %q", clocks)
	}
	syncAll(a, b)
	converged(t, nil, a, b)
	winner := a
	if b.dev.ID > a.dev.ID {
		winner = b
	}
	got, _ := a.store.GetMember(ctx, m.ID)
	if got.FirstName != "by "+winner.name {
		t.Errorf("tie won by %q, want %s", got.FirstName, winner.name)
	}
	if n := a.count(`SELECT COUNT(*) FROM sync_clock_warnings`); n != 0 {
		t.Errorf("clock warning for 10 minutes ahead")
	}
}

// Deleting a row and concurrently editing another field do not conflict:
// the row stays deleted with the edited value (§14.1).
func TestDeleteVersusUpdate(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	m := a.member(1, "Del", "Ete")
	syncAll(a, b)

	if err := a.store.DeleteMember(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.store.UpdateMember(ctx, m.ID, store.MemberInput{Number: 1, FirstName: "Del", LastName: "Ete", Comment: strp("late edit")}); err != nil {
		t.Fatal(err)
	}
	syncAll(a, b)
	converged(t, nil, a, b)
	var deleted, comment any
	if err := b.conn.QueryRow(`SELECT deleted_at, comment FROM members WHERE id = ?`, m.ID).Scan(&deleted, &comment); err != nil {
		t.Fatal(err)
	}
	if deleted == nil || comment != "late edit" {
		t.Errorf("deleted_at %v, comment %v", deleted, comment)
	}
}

// A duplicate member number created concurrently ends in sync_conflicts on
// all devices identically (§14.2).
func TestDuplicateMemberNumber(t *testing.T) {
	mem := transport.NewMemory()
	a, b, c := newNode(t, "a", mem), newNode(t, "b", mem), newNode(t, "c", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	a.admit(c, acl.RoleCoach)
	syncAll(a, b, c)

	ma := a.member(7, "From", "A")
	time.Sleep(2 * time.Millisecond)
	mb := b.member(7, "From", "B")
	syncAll(a, b, c)
	converged(t, nil, a, b, c)
	for _, n := range []*node{a, b, c} {
		conflicts, err := n.store.ListConflicts(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(conflicts) != 1 || conflicts[0].Column != "number" || !strings.Contains(conflicts[0].PK, mb.ID) ||
			!strings.Contains(conflicts[0].KeptPK, ma.ID) {
			t.Errorf("%s conflicts: %+v", n.name, conflicts)
		}
	}
	// Fixing the number on b clears the flag everywhere.
	if _, err := b.store.UpdateMember(ctx, mb.ID, store.MemberInput{Number: 8, FirstName: "From", LastName: "B"}); err != nil {
		t.Fatal(err)
	}
	syncAll(a, b, c)
	converged(t, nil, a, b, c)
	if n := c.count(`SELECT COUNT(*) FROM sync_conflicts`); n != 0 {
		t.Errorf("%d conflicts left on c", n)
	}
}

// A row referencing a row of another stream that has not arrived yet waits
// in sync_pending and is applied once the referenced row arrives (§13.3).
func TestOutOfOrderAcrossStreams(t *testing.T) {
	mem := transport.NewMemory()
	a, b, c := newNode(t, "a", mem), newNode(t, "b", mem), newNode(t, "c", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	a.admit(c, acl.RoleBoard)
	syncAll(a, b, c)

	// b creates a member; a receives it and adds a contact (other scope),
	// a group link (same scope, other stream) and an inventory loan.
	m := b.member(5, "Out", "Oforder")
	b.sync()
	a.sync()
	in := store.MemberInput{Number: 5, FirstName: "Out", LastName: "Oforder", Contact: store.MemberContact{City: strp("Berlin")}}
	g, err := a.store.CreateNamed(ctx, store.Groups, "Jugend")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{g.ID}
	in.GroupIDs = &ids
	if _, err := a.store.UpdateMember(ctx, m.ID, in); err != nil {
		t.Fatal(err)
	}
	item, err := a.store.CreateItem(ctx, store.ItemInput{Name: "Ball", Category: "Sport", Quantity: 1, Location: "Halle"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.CreateLoan(ctx, store.LoanInput{ItemID: item.ID, MemberID: m.ID, LoanedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	a.sync()

	// c sees a's segments but b's segment is still on its way.
	bPath := layout.SegmentPath(b.dev.ID, db.ScopeMembersCore, 1)
	held, err := mem.Get(ctx, bPath)
	if err != nil {
		t.Fatal(err)
	}
	mem.Delete(ctx, bPath)
	c.sync()
	if n := c.count(`SELECT COUNT(*) FROM sync_pending`); n < 3 {
		t.Fatalf("%d pending changes on c, want contact, link and loan", n)
	}
	if n := c.count(`SELECT COUNT(*) FROM members WHERE id = ?`, m.ID); n != 0 {
		t.Fatal("member arrived without b's segment")
	}
	mem.Put(bPath, held)
	c.sync()
	if n := c.count(`SELECT COUNT(*) FROM sync_pending`); n != 0 {
		t.Errorf("%d changes still pending", n)
	}
	syncAll(a, b, c)
	converged(t, nil, a, b, c)
	got, _ := c.store.GetMember(ctx, m.ID)
	if got.Contact.City == nil || *got.Contact.City != "Berlin" || len(got.Groups) != 1 {
		t.Errorf("member on c: %+v", got)
	}
}

// Applying a segment again has no effect (§13.3).
func TestIdempotentReapply(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	syncAll(a, b)
	m := a.member(3, "Idem", "Potent")
	if _, err := a.store.UpdateMember(ctx, m.ID, store.MemberInput{Number: 3, FirstName: "Idem", LastName: "Potent", Comment: strp("x")}); err != nil {
		t.Fatal(err)
	}
	a.sync()
	b.sync()
	// b makes a newer change to the same field, then replays a's stream.
	if _, err := b.store.UpdateMember(ctx, m.ID, store.MemberInput{Number: 3, FirstName: "Idem", LastName: "Potent", Comment: strp("newer")}); err != nil {
		t.Fatal(err)
	}
	before := b.dump()
	history := b.count(`SELECT COUNT(*) FROM sync_history WHERE device_id = ?`, a.dev.ID)
	b.exec(`DELETE FROM sync_streams WHERE device_id = ?`, a.dev.ID)
	if rep := b.sync(); rep.Applied == 0 {
		t.Fatal("nothing re-applied")
	}
	if after := b.dump(); after != before {
		t.Errorf("re-applying changed the state:\n%s\n---\n%s", before, after)
	}
	if n := b.count(`SELECT COUNT(*) FROM sync_history WHERE device_id = ?`, a.dev.ID); n != history {
		t.Errorf("history grew from %d to %d", history, n)
	}
	syncAll(a, b)
	converged(t, nil, a, b)
}

// A device joining later starts from the initial snapshot and replays every
// stream, including those of devices that wrote in several scopes; a
// device with fewer rights only receives what it may read.
func TestLateJoinerAndReadScopes(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.member(1, "Before", "Club")
	a.create()
	a.admit(b, acl.RoleBoard)
	syncAll(a, b)
	m := b.member(2, "Bank", "Holder")
	in := store.MemberInput{Number: 2, FirstName: "Bank", LastName: "Holder", Bank: store.MemberBank{IBAN: strp("DE02120300000000202051")}}
	if _, err := b.store.UpdateMember(ctx, m.ID, in); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.CreateTransaction(ctx, store.TransactionInput{Date: time.Now(), Description: "Beitrag", Amount: 12.5, Type: "income"}); err != nil {
		t.Fatal(err)
	}
	syncAll(a, b)

	late, coach := newNode(t, "late", mem), newNode(t, "coach", mem)
	a.admit(late, acl.RoleBoard)
	a.admit(coach, acl.RoleCoach)
	syncAll(a, b, late, coach)
	converged(t, nil, a, b, late)
	converged(t, []db.Scope{db.ScopeMembersCore, db.ScopeInventory}, a, coach)
	if n := coach.count(`SELECT COUNT(*) FROM member_bank_details`) + coach.count(`SELECT COUNT(*) FROM finance_transactions`); n != 0 {
		t.Errorf("coach holds %d rows of scopes it cannot read", n)
	}
	if n := late.count(`SELECT COUNT(*) FROM member_bank_details`); n != 1 {
		t.Errorf("late joiner has %d bank rows", n)
	}
}
