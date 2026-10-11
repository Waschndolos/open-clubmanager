package replica

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

var ctx = context.Background()

const (
	devA = "aaaaaaaaaaaaaaaaaaaaaaaaaa"
	devB = "bbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func open(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(ctx, filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func apply(t *testing.T, conn *sql.DB, changes ...Change) {
	t.Helper()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	a := NewApplier(ctx, tx)
	for _, ch := range changes {
		res, err := a.Apply(ch)
		if err != nil {
			t.Fatal(err)
		}
		if res.Pending != "" {
			if err := AddPending(ctx, tx, db.ScopeMembersCore, ch, res.Pending); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := a.RetryPending(nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func state(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var out string
	err := conn.QueryRow(`SELECT COALESCE(group_concat(x, '|'), '') FROM (
		SELECT id || ',' || number || ',' || first_name || ',' || COALESCE(comment, '') || ',' || COALESCE(deleted_at, '') || ',' || updated_at AS x FROM members ORDER BY id)`).Scan(&out)
	if err != nil {
		t.Fatal(err)
	}
	var clocks string
	conn.QueryRow(`SELECT COALESCE(group_concat(x, '|'), '') FROM (
		SELECT pk || col || clock_epoch || hlc || device_id AS x FROM sync_field_clock ORDER BY pk, col)`).Scan(&clocks)
	return out + "#" + clocks
}

func member(id, hlcStr, dev string, epoch uint64, set map[string]any) Change {
	return Change{T: "members", PK: map[string]any{"id": id}, Set: set, Clock: Clock{Epoch: epoch, HLC: hlcStr, Device: dev}}
}

// The result of LWW does not depend on the order of arrival, also for equal
// HLCs (device id decides) and across clock epochs (§14.1).
func TestLWWOrderIndependent(t *testing.T) {
	insert := member("m", "0000019a1b2c-0001", devA, 1, map[string]any{"id": "m", "number": int64(1), "first_name": "A", "last_name": "L", "created_at": "x"})
	changes := []Change{
		insert,
		member("m", "0000019a1b2c-0005", devA, 1, map[string]any{"first_name": "tie A"}),
		member("m", "0000019a1b2c-0005", devB, 1, map[string]any{"first_name": "tie B"}),
		member("m", "0000019a1b2c-0009", devA, 1, map[string]any{"comment": "late hlc"}),
		member("m", "0000019a1b2c-0002", devB, 2, map[string]any{"comment": "new generation"}),
		member("m", "0000019a1b2c-0003", devA, 1, map[string]any{"deleted_at": "2026-01-01T00:00:00.000Z"}),
	}
	var want string
	perms := [][]int{{0, 1, 2, 3, 4, 5}, {5, 4, 3, 2, 1, 0}, {2, 4, 0, 5, 1, 3}, {3, 1, 5, 0, 2, 4}}
	for i, p := range perms {
		conn := open(t)
		var seq []Change
		for _, j := range p {
			seq = append(seq, changes[j])
		}
		apply(t, conn, seq...)
		apply(t, conn, seq...) // idempotent
		got := state(t, conn)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Errorf("order %v:\n%s\nwant\n%s", p, got, want)
		}
	}
	if want[:len("m,1,tie B,new generation,2026-01-01T00:00:00.000Z,")] != "m,1,tie B,new generation,2026-01-01T00:00:00.000Z," {
		t.Errorf("state %s", want)
	}
}
