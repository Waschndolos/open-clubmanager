package club

import (
	"errors"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

func TestCreateAdoptsLocalData(t *testing.T) {
	a := newNode(t, "a", transport.NewMemory())
	// Rows from local mode: one with a field clock (NULL device, written by
	// the store before the identity existed and backfilled by Ensure), one
	// imported without any clock, plus a pending outbox op.
	if _, err := a.conn.Exec(`INSERT INTO roles (id, name, updated_at) VALUES ('r1', 'Trainer', '2026-01-01T00:00:00.000Z'), ('r2', 'Kassenprüfer', '2026-01-01T00:00:00.000Z')`); err != nil {
		t.Fatal(err)
	}
	a.conn.Exec(`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id) VALUES ('roles', '{"id":"r1"}', 'name', '000000000001-0000', ?)`, a.dev.ID)
	a.conn.Exec(`INSERT INTO sync_outbox (scope, hlc, op) VALUES ('members.core', '000000000001-0000', '{}')`)

	a.create("TSV").Wipe()
	created := a.club.Head().CreatedHLC

	var outbox int
	a.conn.QueryRow(`SELECT COUNT(*) FROM sync_outbox`).Scan(&outbox)
	if outbox != 0 {
		t.Errorf("%d outbox ops left", outbox)
	}
	clock := func(pk, col string) (h, dev string) {
		a.conn.QueryRow(`SELECT hlc, device_id FROM sync_field_clock WHERE tbl = 'roles' AND pk = ? AND col = ?`, pk, col).Scan(&h, &dev)
		return
	}
	if h, _ := clock(`{"id":"r1"}`, "name"); h != "000000000001-0000" {
		t.Errorf("existing clock changed to %s", h)
	}
	for _, col := range []string{"id", "name"} {
		if h, d := clock(`{"id":"r2"}`, col); h != created || d != a.dev.ID {
			t.Errorf("r2.%s clock = %s/%s, want %s/%s", col, h, d, created, a.dev.ID)
		}
	}
	if h, _ := clock(`{"id":"r2"}`, "updated_at"); h != "" {
		t.Error("updated_at got a field clock")
	}
}

func TestJoinRequiresEmptyTables(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create("TSV").Wipe()
	code := a.invite(acl.RoleCoach)

	b.conn.Exec(`INSERT INTO roles (id, name, updated_at) VALUES ('r1', 'Trainer', '2026-01-01T00:00:00.000Z')`)
	if _, _, err := Join(ctx, b.conn, b.dev, mem, code, "b", ""); !errors.Is(err, ErrLocalData) {
		t.Fatalf("join with local data: %v", err)
	}
	if _, _, err := RecoverNew(ctx, b.conn, b.dev, mem, "irrelevant", "b", ""); !errors.Is(err, ErrLocalData) {
		t.Fatalf("recover with local data: %v", err)
	}
	if err := ClearSyncedData(ctx, b.conn); err != nil {
		t.Fatal(err)
	}
	b.join(code)

	// One club per device.
	if _, _, err := Join(ctx, b.conn, b.dev, mem, code, "b", ""); !errors.Is(err, ErrAlreadyInClub) {
		t.Errorf("second join: %v", err)
	}
}
