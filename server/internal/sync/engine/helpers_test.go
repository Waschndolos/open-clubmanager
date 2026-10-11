package engine

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

var ctx = context.Background()

// node is one installation: its own database, device keys and engine.
type node struct {
	t     *testing.T
	name  string
	conn  *sql.DB
	dev   *identity.Device
	tr    transport.Transport
	club  *club.Club
	eng   *Engine
	store *store.Store
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
	return &node{t: t, name: name, conn: conn, dev: dev, tr: tr, store: store.New(conn)}
}

func folder(t *testing.T, dir string) transport.Transport {
	t.Helper()
	lf, err := transport.OpenLocalFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lf.Close() })
	return lf
}

func (n *node) attach(c *club.Club) {
	n.club = c
	n.eng = New(n.conn, c, n.tr)
}

func (n *node) create() {
	n.t.Helper()
	c, kit, err := club.Create(ctx, n.conn, n.dev, n.tr, "TSV Test", "Person "+n.name, "Laptop")
	if err != nil {
		n.t.Fatal(err)
	}
	kit.Wipe()
	n.attach(c)
}

// admit invites, joins and approves m into n's club with role.
func (n *node) admit(m *node, role string) {
	n.t.Helper()
	inv, err := n.club.CreateInvite(ctx, role, "", 0)
	if err != nil {
		n.t.Fatal(err)
	}
	c, _, err := club.Join(ctx, m.conn, m.dev, m.tr, inv.Code, "Person "+m.name, "Laptop")
	if err != nil {
		n.t.Fatalf("%s join: %v", m.name, err)
	}
	m.attach(c)
	if _, err := n.club.Approve(ctx, m.dev.ID); err != nil {
		n.t.Fatalf("approve %s: %v", m.name, err)
	}
	m.refresh()
	if m.club.Status() != club.StatusMember {
		n.t.Fatalf("%s is %s after approval", m.name, m.club.Status())
	}
}

func (n *node) refresh() {
	n.t.Helper()
	if _, err := n.club.Refresh(ctx); err != nil {
		n.t.Fatalf("%s refresh: %v", n.name, err)
	}
}

// sync refreshes the ACL and runs the engine, failing on errors.
func (n *node) sync() *Report {
	n.t.Helper()
	n.refresh()
	rep, err := n.eng.Sync(ctx)
	if err != nil {
		n.t.Fatalf("%s sync: %v", n.name, err)
	}
	if rep.UploadErr != nil {
		n.t.Fatalf("%s upload: %v", n.name, rep.UploadErr)
	}
	return rep
}

// syncAll syncs every node twice, so that everything written in the first
// round reaches everyone in the second.
func syncAll(nodes ...*node) {
	for range 2 {
		for _, n := range nodes {
			n.sync()
		}
	}
}

func (n *node) member(number int, first, last string) store.Member {
	n.t.Helper()
	m, err := n.store.CreateMember(ctx, store.MemberInput{Number: number, FirstName: first, LastName: last})
	if err != nil {
		n.t.Fatalf("%s create member: %v", n.name, err)
	}
	return m
}

func (n *node) exec(q string, args ...any) {
	n.t.Helper()
	if _, err := n.conn.Exec(q, args...); err != nil {
		n.t.Fatalf("%s: %s: %v", n.name, q, err)
	}
}

func (n *node) count(q string, args ...any) int {
	n.t.Helper()
	var c int
	if err := n.conn.QueryRow(q, args...).Scan(&c); err != nil {
		n.t.Fatalf("%s: %s: %v", n.name, q, err)
	}
	return c
}

// dump returns the replicated state of the given scopes: all rows (with
// updated_at, which every device derives the same way), field clocks and
// conflicts.
func (n *node) dump(scopes ...db.Scope) string {
	n.t.Helper()
	if len(scopes) == 0 {
		scopes = db.Scopes
	}
	var b strings.Builder
	for _, t := range db.SyncedTables {
		in := false
		for _, s := range scopes {
			in = in || t.Scope == s
		}
		if !in {
			continue
		}
		fmt.Fprintf(&b, "== %s\n", t.Name)
		b.WriteString(n.rows(`SELECT * FROM ` + t.Name + ` ORDER BY ` + strings.Join(t.PK, ", ")))
		fmt.Fprintf(&b, "-- clocks\n")
		b.WriteString(n.rows(`SELECT pk, col, clock_epoch, hlc, device_id FROM sync_field_clock WHERE tbl = ? ORDER BY pk, col`, t.Name))
		fmt.Fprintf(&b, "-- conflicts\n")
		b.WriteString(n.rows(`SELECT pk, col, value, kept_pk FROM sync_conflicts WHERE tbl = ? ORDER BY pk, col`, t.Name))
	}
	return b.String()
}

func (n *node) rows(q string, args ...any) string {
	n.t.Helper()
	rows, err := n.conn.Query(q, args...)
	if err != nil {
		n.t.Fatalf("%s: %v", q, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var b strings.Builder
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			n.t.Fatal(err)
		}
		for i, v := range vals {
			if bs, ok := v.([]byte); ok {
				v = string(bs)
			}
			fmt.Fprintf(&b, "%s=%v ", cols[i], v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// converged fails unless all nodes hold the same state of the scopes.
func converged(t *testing.T, scopes []db.Scope, nodes ...*node) {
	t.Helper()
	want := nodes[0].dump(scopes...)
	for _, n := range nodes[1:] {
		if got := n.dump(scopes...); got != want {
			t.Fatalf("%s and %s differ:\n--- %s\n%s\n--- %s\n%s", nodes[0].name, n.name, nodes[0].name, want, n.name, got)
		}
	}
}

// stream returns the status and seq of a stream on n.
func (n *node) stream(device string, s db.Scope) (string, int) {
	n.t.Helper()
	var st string
	var seq int
	err := n.conn.QueryRow(`SELECT status, seq FROM sync_streams WHERE device_id = ? AND scope = ?`, device, s).Scan(&st, &seq)
	if err == sql.ErrNoRows {
		return "", 0
	}
	if err != nil {
		n.t.Fatal(err)
	}
	return st, seq
}

// setHLC moves a node's persisted clock, e.g. to force equal timestamps.
func (n *node) setHLC(ms int64, counter uint16) {
	n.exec(`UPDATE sync_device SET hlc = ? WHERE id = 1`, fmt.Sprintf("%012x-%04x", ms, counter))
}

func futureMS(d time.Duration) int64 { return time.Now().Add(d).UnixMilli() }
