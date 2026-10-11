package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

type outboxEntry struct {
	scope string
	op    op.Op
}

func readOutbox(t *testing.T, conn *sql.DB) []outboxEntry {
	t.Helper()
	rows, err := conn.Query(`SELECT scope, hlc, op FROM sync_outbox ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []outboxEntry
	for rows.Next() {
		var e outboxEntry
		var ts, data string
		if err := rows.Scan(&e.scope, &ts, &data); err != nil {
			t.Fatal(err)
		}
		d := json.NewDecoder(strings.NewReader(data))
		d.UseNumber()
		if err := d.Decode(&e.op); err != nil {
			t.Fatalf("outbox op %s: %v", data, err)
		}
		if e.op.HLC != ts {
			t.Errorf("outbox hlc column %s != op hlc %s", ts, e.op.HLC)
		}
		for k, v := range e.op.Set {
			e.op.Set[k] = op.Normalize(v)
		}
		out = append(out, e)
	}
	return out
}

func clearOutbox(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec(`DELETE FROM sync_outbox`); err != nil {
		t.Fatal(err)
	}
}

func opsFor(entries []outboxEntry, table string) []outboxEntry {
	var out []outboxEntry
	for _, e := range entries {
		if e.op.T == table {
			out = append(out, e)
		}
	}
	return out
}

func TestWritesRecordOutboxAndFieldClocks(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)

	role, err := s.CreateNamed(ctx, Roles, "Trainer")
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.CreateMember(ctx, MemberInput{
		Number: 5, FirstName: "Ada", LastName: "Lovelace",
		Contact: MemberContact{Email: ptr("ada@example.org")},
		Bank:    MemberBank{IBAN: ptr("DE02120300000000202051")},
		RoleIDs: &[]string{role.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	ops := readOutbox(t, conn)
	wantScopes := map[string]string{
		"roles": "members.core", "members": "members.core", "member_roles": "members.core",
		"member_contacts": "members.contact", "member_bank_details": "members.bank",
	}
	if len(ops) != len(wantScopes) {
		t.Fatalf("outbox has %d ops, want %d: %+v", len(ops), len(wantScopes), ops)
	}
	prev := ""
	for _, e := range ops {
		if e.scope != wantScopes[e.op.T] {
			t.Errorf("%s op in scope %s, want %s", e.op.T, e.scope, wantScopes[e.op.T])
		}
		if _, err := hlc.Parse(e.op.HLC); err != nil {
			t.Error(err)
		}
		if e.op.HLC <= prev {
			t.Errorf("hlc %s not after %s", e.op.HLC, prev)
		}
		prev = e.op.HLC
		if _, ok := e.op.Set["updated_at"]; ok {
			t.Errorf("%s op contains updated_at", e.op.T)
		}
	}

	ins := opsFor(ops, "members")[0].op
	if ins.PK["id"] != m.ID {
		t.Errorf("members op pk = %v", ins.PK)
	}
	if ins.Set["number"] != int64(5) || ins.Set["first_name"] != "Ada" || ins.Set["id"] != m.ID {
		t.Errorf("insert set = %v", ins.Set)
	}
	if _, ok := ins.Set["birthday"]; ok {
		t.Error("insert op must omit NULL columns")
	}
	link := opsFor(ops, "member_roles")[0].op
	if link.PK["member_id"] != m.ID || link.PK["role_id"] != role.ID {
		t.Errorf("link pk = %v", link.PK)
	}

	// The local HLC is persisted.
	var stored string
	conn.QueryRow(`SELECT hlc FROM sync_device`).Scan(&stored)
	if stored != prev {
		t.Errorf("sync_device.hlc = %s, want %s", stored, prev)
	}

	// Field clocks exist for every column set by the insert.
	pk := op.EncodePK(map[string]any{"id": m.ID})
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM sync_field_clock WHERE tbl = 'members' AND pk = ? AND hlc = ? AND device_id IS NULL`,
		pk, ins.HLC).Scan(&n)
	if n != len(ins.Set) {
		t.Errorf("%d field clocks for the member, want %d", n, len(ins.Set))
	}

	// An update writes only the changed columns.
	clearOutbox(t, conn)
	if _, err := s.UpdateMember(ctx, m.ID, MemberInput{
		Number: 5, FirstName: "Augusta Ada", LastName: "Lovelace",
		Contact: MemberContact{Email: ptr("ada@example.org")},
		Bank:    MemberBank{IBAN: ptr("DE02120300000000202051")},
	}); err != nil {
		t.Fatal(err)
	}
	ops = readOutbox(t, conn)
	if len(ops) != 1 || ops[0].op.T != "members" || len(ops[0].op.Set) != 1 || ops[0].op.Set["first_name"] != "Augusta Ada" {
		t.Fatalf("update ops = %+v", ops)
	}
	var clock string
	conn.QueryRow(`SELECT hlc FROM sync_field_clock WHERE tbl = 'members' AND pk = ? AND col = 'first_name'`, pk).Scan(&clock)
	if clock != ops[0].op.HLC {
		t.Errorf("first_name clock = %s, want %s", clock, ops[0].op.HLC)
	}
	conn.QueryRow(`SELECT hlc FROM sync_field_clock WHERE tbl = 'members' AND pk = ? AND col = 'last_name'`, pk).Scan(&clock)
	if clock != ins.HLC {
		t.Errorf("unchanged last_name clock moved to %s", clock)
	}

	// A no-op update writes nothing.
	clearOutbox(t, conn)
	if _, err := s.UpdateMember(ctx, m.ID, MemberInput{
		Number: 5, FirstName: "Augusta Ada", LastName: "Lovelace",
		Contact: MemberContact{Email: ptr("ada@example.org")},
		Bank:    MemberBank{IBAN: ptr("DE02120300000000202051")},
	}); err != nil {
		t.Fatal(err)
	}
	if ops := readOutbox(t, conn); len(ops) != 0 {
		t.Errorf("no-op update wrote %+v", ops)
	}

	// Delete writes tombstone ops for the member and its dependants.
	if _, err := s.CreateFee(ctx, FeeInput{MemberID: m.ID, Amount: 10, DueDate: *date("2025-01-01"), Year: 2025}); err != nil {
		t.Fatal(err)
	}
	clearOutbox(t, conn)
	if err := s.DeleteMember(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	ops = readOutbox(t, conn)
	for _, table := range []string{"members", "member_contacts", "member_bank_details", "member_roles", "member_fees"} {
		got := opsFor(ops, table)
		if len(got) != 1 || len(got[0].op.Set) != 1 || got[0].op.Set["deleted_at"] == nil {
			t.Errorf("%s delete ops = %+v", table, got)
		}
	}
	if fee := opsFor(ops, "member_fees"); len(fee) == 1 && fee[0].scope != "finance" {
		t.Errorf("fee tombstone in scope %s", fee[0].scope)
	}
}

func TestRestoreAndInventoryScope(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)

	role, _ := s.CreateNamed(ctx, Roles, "Trainer")
	m, _ := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "A", LastName: "A", RoleIDs: &[]string{role.ID}})
	if _, err := s.UpdateMember(ctx, m.ID, MemberInput{Number: 1, FirstName: "A", LastName: "A", RoleIDs: &[]string{}}); err != nil {
		t.Fatal(err)
	}
	clearOutbox(t, conn)
	// Re-adding the link revives the tombstone: a restore op (deleted_at null).
	if _, err := s.UpdateMember(ctx, m.ID, MemberInput{Number: 1, FirstName: "A", LastName: "A", RoleIDs: &[]string{role.ID}}); err != nil {
		t.Fatal(err)
	}
	ops := readOutbox(t, conn)
	if len(ops) != 1 || ops[0].op.T != "member_roles" {
		t.Fatalf("revive ops = %+v", ops)
	}
	if v, ok := ops[0].op.Set["deleted_at"]; !ok || v != nil || len(ops[0].op.Set) != 1 {
		t.Errorf("revive set = %v, want {deleted_at: null}", ops[0].op.Set)
	}

	clearOutbox(t, conn)
	item, err := s.CreateItem(ctx, ItemInput{Name: "Ball", Category: "Sport", Quantity: 3, Location: "Halle", PurchasePrice: ptr(12.5)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateLoan(ctx, LoanInput{ItemID: item.ID, MemberID: m.ID, LoanedAt: *date("2025-02-01")}); err != nil {
		t.Fatal(err)
	}
	ops = readOutbox(t, conn)
	if len(ops) != 2 || ops[0].scope != "inventory" || ops[1].scope != "inventory" {
		t.Errorf("inventory ops = %+v", ops)
	}
	if ops[0].op.Set["purchase_price_cents"] != int64(1250) {
		t.Errorf("money in op = %v, want cents", ops[0].op.Set["purchase_price_cents"])
	}
}

func TestLocalTablesAreNotInOutbox(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)
	u, err := s.CreateUser(ctx, "a@example.org", "hash", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	s.Audit(ctx, "CREATE", "user", u.ID, "admin")
	if err := s.SetPreference(ctx, u.ID, "theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if ops := readOutbox(t, conn); len(ops) != 0 {
		t.Errorf("local writes produced ops: %+v", ops)
	}
}

// The local pre-save checks keep returning ErrConflict now that SQLite no
// longer enforces uniqueness on synced tables.
func TestPreSaveUniquenessChecks(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)

	a, err := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "A", LastName: "A", Contact: MemberContact{Email: ptr("a@example.org")}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateMember(ctx, MemberInput{Number: 2, FirstName: "B", LastName: "B", Contact: MemberContact{Email: ptr("b@example.org")}})
	if err != nil {
		t.Fatal(err)
	}

	// Update to another member's number or e-mail.
	if _, err := s.UpdateMember(ctx, b.ID, MemberInput{Number: 1, FirstName: "B", LastName: "B"}); !errors.Is(err, ErrConflict) {
		t.Errorf("update to taken number: %v, want ErrConflict", err)
	}
	if _, err := s.UpdateMember(ctx, b.ID, MemberInput{Number: 2, FirstName: "B", LastName: "B", Contact: MemberContact{Email: ptr("a@example.org")}}); !errors.Is(err, ErrConflict) {
		t.Errorf("update to taken e-mail: %v, want ErrConflict", err)
	}
	// The failed updates were rolled back.
	got, _ := s.GetMember(ctx, b.ID)
	if got.Number != 2 || got.Contact.Email == nil || *got.Contact.Email != "b@example.org" {
		t.Errorf("failed update left changes: %+v", got)
	}
	// Keeping one's own values is fine; so are several members without e-mail.
	if _, err := s.UpdateMember(ctx, a.ID, MemberInput{Number: 1, FirstName: "A2", LastName: "A", Contact: MemberContact{Email: ptr("a@example.org")}}); err != nil {
		t.Errorf("update keeping own values: %v", err)
	}
	for i := 3; i < 5; i++ {
		if _, err := s.CreateMember(ctx, MemberInput{Number: i, FirstName: "N", LastName: "N", Contact: MemberContact{City: ptr("Bonn")}}); err != nil {
			t.Errorf("member without e-mail: %v", err)
		}
	}
	// Not-found wins over conflict for unknown ids, as before.
	if _, err := s.UpdateMember(ctx, "missing", MemberInput{Number: 1}); !errors.Is(err, ErrNotFound) {
		t.Errorf("update unknown member: %v, want ErrNotFound", err)
	}

	for _, k := range []NamedKind{Roles, Groups, Sections} {
		x, err := s.CreateNamed(ctx, k, "X")
		if err != nil {
			t.Fatal(err)
		}
		y, err := s.CreateNamed(ctx, k, "Y")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateNamed(ctx, k, "X"); !errors.Is(err, ErrConflict) {
			t.Errorf("%s: duplicate create: %v, want ErrConflict", k.table, err)
		}
		if _, err := s.RenameNamed(ctx, k, y.ID, "X"); !errors.Is(err, ErrConflict) {
			t.Errorf("%s: rename to taken name: %v, want ErrConflict", k.table, err)
		}
		if _, err := s.RenameNamed(ctx, k, x.ID, "X"); err != nil {
			t.Errorf("%s: rename to own name: %v", k.table, err)
		}
		if _, err := s.RenameNamed(ctx, k, "missing", "X"); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: rename unknown: %v, want ErrNotFound", k.table, err)
		}
	}
}

// setClock overwrites the field clock of one column, simulating a value that
// was applied from another device.
func setClock(t *testing.T, conn *sql.DB, table, pk, col, ts, device string) {
	t.Helper()
	_, err := conn.Exec(`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (tbl, pk, col) DO UPDATE SET hlc = excluded.hlc, device_id = excluded.device_id`,
		table, pk, col, ts, device)
	if err != nil {
		t.Fatal(err)
	}
}

func detect(t *testing.T, conn *sql.DB, table string, pks ...string) {
	t.Helper()
	tx, err := conn.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := DetectUniqueConflicts(context.Background(), tx, table, pks); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func memberPK(id string) string { return op.EncodePK(map[string]any{"id": id}) }

func TestDetectUniqueConflicts(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)

	// Three members end up with number 7 after remote changes were applied
	// (simulated by writing SQL directly; SQLite accepts duplicates).
	ids := []string{"0190f5d0-0000-7000-8000-00000000000a", "0190f5d0-0000-7000-8000-00000000000b", "0190f5d0-0000-7000-8000-00000000000c"}
	for _, id := range ids {
		if _, err := conn.Exec(`INSERT INTO members (id, number, first_name, last_name) VALUES (?, 7, 'X', 'Y')`, id); err != nil {
			t.Fatal(err)
		}
	}
	setClock(t, conn, "members", memberPK(ids[0]), "number", "0000019a1b2c-0005", "dev-b")
	setClock(t, conn, "members", memberPK(ids[1]), "number", "0000019a1b2c-0005", "dev-a") // same hlc, smaller device
	setClock(t, conn, "members", memberPK(ids[2]), "number", "0000019a1b2c-0001", "dev-z") // smallest hlc

	detect(t, conn, "members", memberPK(ids[0]))
	conflicts, err := s.ListConflicts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 2 {
		t.Fatalf("conflicts = %+v, want 2", conflicts)
	}
	for _, c := range conflicts {
		if c.Table != "members" || c.Column != "number" || c.Value != int64(7) || c.KeptPK != memberPK(ids[2]) {
			t.Errorf("conflict = %+v", c)
		}
		if c.PK == memberPK(ids[2]) {
			t.Error("the oldest value holder was flagged")
		}
	}

	// Running again is idempotent.
	detect(t, conn, "members", memberPK(ids[0]), memberPK(ids[1]), memberPK(ids[2]))
	if again, _ := s.ListConflicts(ctx); len(again) != 2 {
		t.Errorf("after re-run: %+v", again)
	}

	// The keeper is deleted: of the remaining two, (hlc, device) decides.
	if _, err := conn.Exec(`UPDATE members SET deleted_at = '2025-01-01T00:00:00.000Z' WHERE id = ?`, ids[2]); err != nil {
		t.Fatal(err)
	}
	detect(t, conn, "members", memberPK(ids[2]))
	conflicts, _ = s.ListConflicts(ctx)
	if len(conflicts) != 1 || conflicts[0].PK != memberPK(ids[0]) || conflicts[0].KeptPK != memberPK(ids[1]) {
		t.Fatalf("after keeper deleted: %+v", conflicts)
	}

	// A local fix through the store clears the flag.
	if _, err := s.UpdateMember(ctx, ids[0], MemberInput{Number: 8, FirstName: "X", LastName: "Y"}); err != nil {
		t.Fatal(err)
	}
	if conflicts, _ := s.ListConflicts(ctx); len(conflicts) != 0 {
		t.Errorf("after local fix: %+v", conflicts)
	}
}

func TestDetectUniqueConflictsOnContactEmail(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)

	a, _ := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "A", LastName: "A", Contact: MemberContact{Email: ptr("x@example.org")}})
	b, _ := s.CreateMember(ctx, MemberInput{Number: 2, FirstName: "B", LastName: "B", Contact: MemberContact{City: ptr("Bonn")}})
	// A remote change gives b the same e-mail with a newer clock.
	if _, err := conn.Exec(`UPDATE member_contacts SET email = 'x@example.org' WHERE member_id = ?`, b.ID); err != nil {
		t.Fatal(err)
	}
	bPK := op.EncodePK(map[string]any{"member_id": b.ID})
	setClock(t, conn, "member_contacts", bPK, "email", "ffffffffffff-0000", "dev-remote")

	detect(t, conn, "member_contacts", bPK)
	conflicts, _ := s.ListConflicts(ctx)
	if len(conflicts) != 1 || conflicts[0].PK != bPK || conflicts[0].Value != "x@example.org" ||
		conflicts[0].KeptPK != op.EncodePK(map[string]any{"member_id": a.ID}) {
		t.Fatalf("conflicts = %+v", conflicts)
	}

	// Removing b's contact data (tombstone) resolves it.
	if _, err := s.UpdateMember(ctx, b.ID, MemberInput{Number: 2, FirstName: "B", LastName: "B"}); err != nil {
		t.Fatal(err)
	}
	if conflicts, _ := s.ListConflicts(ctx); len(conflicts) != 0 {
		t.Errorf("after tombstone: %+v", conflicts)
	}
}

func TestDetectUniqueConflictsWithoutClocks(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)
	// Rows without field clocks (e.g. imported before sync) are ordered by
	// primary key, so the result is still deterministic.
	for _, id := range []string{"r2", "r1"} {
		if _, err := conn.Exec(`INSERT INTO roles (id, name) VALUES (?, 'Trainer')`, id); err != nil {
			t.Fatal(err)
		}
	}
	detect(t, conn, "roles", memberPK("r2"))
	conflicts, _ := s.ListConflicts(ctx)
	if len(conflicts) != 1 || conflicts[0].PK != memberPK("r2") || conflicts[0].KeptPK != memberPK("r1") {
		t.Errorf("conflicts = %+v", conflicts)
	}
}
