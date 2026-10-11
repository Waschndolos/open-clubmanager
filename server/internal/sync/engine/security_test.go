package engine

import (
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// craft writes a segment signed by n with arbitrary content, bypassing the
// engine, at the next seq of n's stream in scope.
type craft struct {
	scope      db.Scope
	seq        uint64
	prev       *string
	aclVersion uint64
	epoch      uint64
	schema     int
	ops        []op.Op
	clubID     string
	// signer overrides the signing device (forgeries).
	signer *node
}

func (n *node) craft(c craft) []byte {
	n.t.Helper()
	head := n.club.Head()
	if c.aclVersion == 0 {
		c.aclVersion = head.ACL.Version
	}
	if c.epoch == 0 {
		c.epoch = head.Epochs[c.scope]
	}
	if c.schema == 0 {
		c.schema = segment.SchemaVersion
	}
	if c.clubID == "" {
		c.clubID = n.club.Genesis().ClubID
	}
	key, err := n.club.ScopeKey(ctx, c.scope, c.epoch)
	if err != nil {
		n.t.Fatal(err)
	}
	plain, _ := segment.EncodeOps(c.ops)
	nonce, _ := segment.NewNonce()
	hdr := segment.Header{
		ClubID: c.clubID, DeviceID: n.dev.ID, Scope: c.scope, Seq: c.seq, Prev: c.prev,
		Epoch: c.epoch, ACLVersion: c.aclVersion, Schema: c.schema, Nonce: nonce,
		HLCMin: c.ops[0].HLC, HLCMax: c.ops[len(c.ops)-1].HLC,
	}
	sign := n.dev.Sign
	if c.signer != nil {
		sign = c.signer.dev.Sign
	}
	data, err := segment.Seal(hdr, nonce, plain, key, sign)
	if err != nil {
		n.t.Fatal(err)
	}
	return data
}

func memberOp(id, hlcStr string, number int) op.Op {
	return op.Op{T: "members", PK: map[string]any{"id": id}, HLC: hlcStr,
		Set: map[string]any{"id": id, "number": int64(number), "first_name": "X", "last_name": "Y", "created_at": "2026-01-01T00:00:00.000Z"}}
}

// A revoked device's segments after its cutoff are ignored, and a device
// that already applied such a segment rebuilds the scope (§13.2 rule 6,
// §17.4).
func TestRevokedDeviceAfterCutoff(t *testing.T) {
	mem := transport.NewMemory()
	a, b, c := newNode(t, "a", mem), newNode(t, "b", mem), newNode(t, "c", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	a.admit(c, acl.RoleBoard)
	syncAll(a, b, c)

	x := b.member(1, "Accepted", "B")
	b.sync()
	a.sync()
	c.sync()

	// b's second segment reaches c, but not a, before a revokes b.
	y := b.member(2, "Beyond", "Cutoff")
	b.sync()
	c.sync()
	if n := c.count(`SELECT COUNT(*) FROM members WHERE id = ?`, y.ID); n != 1 {
		t.Fatal("c did not apply b's second segment")
	}
	// c has changes of its own, flushed and not yet flushed.
	w1 := c.member(3, "Own", "Flushed")
	c.sync()
	w2 := c.member(4, "Own", "Unflushed")

	v, err := a.club.Revoke(ctx, b.dev.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	cut := v.Device(b.dev.ID).Cutoff
	if cut[db.ScopeMembersCore].Seq != 1 || cut[db.ScopeFinance].Seq != 0 || len(cut) != len(db.Scopes) {
		t.Fatalf("cutoff %+v", cut)
	}
	// b does not know yet and keeps writing.
	z := b.member(5, "After", "Revocation")
	if _, err := b.eng.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := mem.Get(ctx, layout.SegmentPath(b.dev.ID, db.ScopeMembersCore, 3)); err != nil {
		t.Fatalf("b's third segment: %v", err)
	}

	rep := c.sync()
	if len(rep.Rebuilt) == 0 {
		t.Fatal("c did not rebuild after learning the revocation")
	}
	a.sync()
	c.sync()
	for _, n := range []*node{a, c} {
		for id, want := range map[string]int{x.ID: 1, y.ID: 0, z.ID: 0, w1.ID: 1, w2.ID: 1} {
			if got := n.count(`SELECT COUNT(*) FROM members WHERE id = ?`, id); got != want {
				t.Errorf("%s has %d rows of %s, want %d", n.name, got, id, want)
			}
		}
		if st, seq := n.stream(b.dev.ID, db.ScopeMembersCore); st != StatusOK || seq != 1 {
			t.Errorf("%s: b's stream %s at %d", n.name, st, seq)
		}
	}
	converged(t, nil, a, c)

	// b learns it is revoked and stops.
	b.refresh()
	if rep, err := b.eng.Sync(ctx); err != nil || rep.Written != 0 {
		t.Errorf("revoked device synced: %+v, %v", rep, err)
	}
}

// Tampered and truncated segments are rejected; a file that is still being
// synced is retried, a persistently invalid one quarantines the stream.
func TestTamperedAndPartialSegments(t *testing.T) {
	mem := transport.NewMemory()
	a, b := newNode(t, "a", mem), newNode(t, "b", mem)
	a.create()
	a.admit(b, acl.RoleBoard)
	syncAll(a, b)

	m := a.member(1, "Part", "Ial")
	a.sync()
	path := layout.SegmentPath(a.dev.ID, db.ScopeMembersCore, 1)
	full, _ := mem.Get(ctx, path)

	// Partial file: retried, then accepted once complete.
	mem.Put(path, full[:len(full)/2])
	b.sync()
	if st, seq := b.stream(a.dev.ID, db.ScopeMembersCore); st != StatusWaiting || seq != 0 {
		t.Fatalf("after partial file: %s %d", st, seq)
	}
	mem.Put(path, full)
	b.sync()
	if st, seq := b.stream(a.dev.ID, db.ScopeMembersCore); st != StatusOK || seq != 1 {
		t.Fatalf("after complete file: %s %d", st, seq)
	}
	if n := b.count(`SELECT COUNT(*) FROM members WHERE id = ?`, m.ID); n != 1 {
		t.Fatal("member missing")
	}

	// Tampered second segment: a flipped ciphertext byte breaks the
	// signature; the stream is quarantined after DefaultMaxAttempts tries.
	a.member(2, "Tam", "Pered")
	a.sync()
	path2 := layout.SegmentPath(a.dev.ID, db.ScopeMembersCore, 2)
	seg, _ := mem.Get(ctx, path2)
	bad := append([]byte(nil), seg...)
	bad[len(bad)-80] ^= 1
	mem.Put(path2, bad)
	for i := 1; i <= DefaultMaxAttempts; i++ {
		b.sync()
		st, seq := b.stream(a.dev.ID, db.ScopeMembersCore)
		want := StatusWaiting
		if i == DefaultMaxAttempts {
			want = StatusQuarantined
		}
		if st != want || seq != 1 {
			t.Fatalf("attempt %d: %s %d", i, st, seq)
		}
	}
	// Quarantine is final: restoring the file does not resume the stream.
	mem.Put(path2, seg)
	b.sync()
	if st, _ := b.stream(a.dev.ID, db.ScopeMembersCore); st != StatusQuarantined {
		t.Errorf("quarantine lifted: %s", st)
	}
	sc, _, _, err := Counts(ctx, b.conn)
	if err != nil || sc.Quarantined != 1 {
		t.Errorf("counts %+v, %v", sc, err)
	}

	// Recovery (§17.4): revoking the blocked device at its last valid
	// segment ends the stream and lifts the quarantine.
	v, err := b.club.Revoke(ctx, a.dev.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cut := v.Device(a.dev.ID).Cutoff[db.ScopeMembersCore]; cut.Seq != 1 || *cut.Hash != wire.Hash(full) {
		t.Fatalf("cutoff %+v", cut)
	}
	b.sync()
	if st, seq := b.stream(a.dev.ID, db.ScopeMembersCore); st != StatusOK || seq != 1 {
		t.Errorf("after revoking the blocked device: %s %d", st, seq)
	}
	// Conflict copies and temp files are ignored (§6).
	mem.Put("segments/"+a.dev.ID+"/members.core/000000000003 (conflicted copy).seg", seg)
	mem.Put("segments/"+a.dev.ID+"/members.core/.tmp-123", seg)
	b.sync()
}

// Crafted, validly signed segments that break other rules of §13.2.
func TestVerificationRules(t *testing.T) {
	hash := func(data []byte) *string { h := wire.Hash(data); return &h }
	ts := "0000019a1b2c-0001"

	cases := []struct {
		name   string
		role   string
		scope  db.Scope
		c      func(*craft)
		want   string
		reason string
	}{
		{name: "rule 5: no rw", role: acl.RoleCoach, scope: db.ScopeMembersCore, want: StatusQuarantined, reason: "no rw"},
		{name: "rule 9: table of another scope", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops = []op.Op{{T: "member_bank_details", PK: map[string]any{"member_id": "m"}, HLC: ts, Set: map[string]any{"iban": "DE00"}}}
		}, want: StatusQuarantined, reason: "not in scope"},
		{name: "rule 9: unknown column", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops[0].Set["password"] = "x"
		}, want: StatusQuarantined, reason: "column"},
		{name: "rule 9: updated_at", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops[0].Set["updated_at"] = "2026-01-01T00:00:00.000Z"
		}, want: StatusQuarantined, reason: "column"},
		{name: "rule 2: other club", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.clubID = "0190f5c2-0000-7000-8000-000000000000"
		}, want: StatusQuarantined, reason: "another club"},
		{name: "rule 1: seq differs from file name", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.seq = 2
			c.prev = hash([]byte("x"))
		}, want: StatusQuarantined, reason: "file name"},
		{name: "valid segment of another scope", scope: db.ScopeFinance, c: func(c *craft) {
			c.ops = []op.Op{{T: "finance_transactions", PK: map[string]any{"id": "t"}, HLC: ts,
				Set: map[string]any{"id": "t", "date": "2026-01-01", "description": "d", "amount_cents": int64(1), "type": "income"}}}
			c.seq = 1
		}, want: StatusOK},
		{name: "rule 5: not a writer in the named ACL version", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.aclVersion = 1
			c.epoch = 1
		}, want: StatusQuarantined, reason: "rw"},
		{name: "newer ACL version waits", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.aclVersion = 99
		}, want: StatusWaiting, reason: "ACL version 99"},
		{name: "rule 10: newer schema waits", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.schema = segment.SchemaVersion + 1
		}, want: StatusWaiting, reason: "update"},
		{name: "purge waits", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops[0].Purge = []string{"*"}
		}, want: StatusWaiting, reason: "update"},
		{name: "rule 8: hlc range", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops = append(c.ops, memberOp("m2", "0000019a1b2c-0000", 2))
		}, want: StatusQuarantined, reason: "hlc"},
		{name: "§12: ops out of HLC order", scope: db.ScopeMembersCore, c: func(c *craft) {
			c.ops = []op.Op{memberOp("m1", "0000019a1b2c-0002", 1), memberOp("m2", "0000019a1b2c-0001", 2), memberOp("m3", "0000019a1b2c-0003", 3)}
		}, want: StatusQuarantined, reason: "order"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mem := transport.NewMemory()
			// Each case gets fresh devices: quarantine is per stream.
			a, w, r := newNode(t, "a", mem), newNode(t, "w", mem), newNode(t, "r", mem)
			a.create()
			role := tc.role
			if role == "" {
				role = acl.RoleBoard
			}
			a.admit(w, role)
			a.admit(r, acl.RoleBoard)
			syncAll(a, w, r)
			c := craft{scope: tc.scope, seq: 1, ops: []op.Op{memberOp("m1", ts, 1)}}
			if tc.c != nil {
				tc.c(&c)
			}
			mem.Put(layout.SegmentPath(w.dev.ID, tc.scope, 1), w.craft(c))
			for range DefaultMaxAttempts + 1 {
				r.sync()
			}
			st, _ := r.stream(w.dev.ID, tc.scope)
			var detail string
			r.conn.QueryRow(`SELECT detail FROM sync_streams WHERE device_id = ? AND scope = ?`, w.dev.ID, tc.scope).Scan(&detail)
			if st != tc.want || !strings.Contains(detail, tc.reason) {
				t.Errorf("status %s (%s), want %s with %q", st, detail, tc.want, tc.reason)
			}
			if tc.want != StatusOK && r.count(`SELECT COUNT(*) FROM members WHERE id = 'm1'`) != 0 {
				t.Error("rejected op applied")
			}
		})
	}
}

// A segment whose prev does not match the accepted predecessor is rejected
// (§13.2 rule 4), and a gap stops the stream until the missing file arrives.
func TestBrokenChainAndGap(t *testing.T) {
	mem := transport.NewMemory()
	a, w, r := newNode(t, "a", mem), newNode(t, "w", mem), newNode(t, "r", mem)
	a.create()
	a.admit(w, acl.RoleBoard)
	a.admit(r, acl.RoleBoard)
	syncAll(a, w, r)

	first := w.craft(craft{scope: db.ScopeMembersCore, seq: 1, ops: []op.Op{memberOp("m1", "0000019a1b2c-0001", 1)}})
	wrong := wire.Hash([]byte("not the first segment"))
	second := w.craft(craft{scope: db.ScopeMembersCore, seq: 2, prev: &wrong, ops: []op.Op{memberOp("m2", "0000019a1b2c-0002", 2)}})

	mem.Put(layout.SegmentPath(w.dev.ID, db.ScopeMembersCore, 2), second)
	r.sync()
	if st, seq := r.stream(w.dev.ID, db.ScopeMembersCore); st != StatusWaiting || seq != 0 {
		t.Fatalf("gap: %s %d", st, seq)
	}
	mem.Put(layout.SegmentPath(w.dev.ID, db.ScopeMembersCore, 1), first)
	for range DefaultMaxAttempts {
		r.sync()
	}
	if st, seq := r.stream(w.dev.ID, db.ScopeMembersCore); st != StatusQuarantined || seq != 1 {
		t.Errorf("broken chain: %s %d", st, seq)
	}
	if r.count(`SELECT COUNT(*) FROM members WHERE id = 'm2'`) != 0 || r.count(`SELECT COUNT(*) FROM members WHERE id = 'm1'`) != 1 {
		t.Error("wrong members applied")
	}
}

// A segment in another device's name, signed by a club member who knows the
// scope key, is rejected (§13.2 rule 3).
func TestForgedSigner(t *testing.T) {
	mem := transport.NewMemory()
	a, w, r := newNode(t, "a", mem), newNode(t, "w", mem), newNode(t, "r", mem)
	a.create()
	a.admit(w, acl.RoleBoard)
	a.admit(r, acl.RoleBoard)
	syncAll(a, w, r)

	forged := w.craft(craft{scope: db.ScopeMembersCore, seq: 1, signer: r, ops: []op.Op{memberOp("m1", "0000019a1b2c-0001", 1)}})
	mem.Put(layout.SegmentPath(w.dev.ID, db.ScopeMembersCore, 1), forged)
	for range DefaultMaxAttempts {
		a.sync()
	}
	if st, _ := a.stream(w.dev.ID, db.ScopeMembersCore); st != StatusQuarantined {
		t.Errorf("forged segment: stream %s", st)
	}
	if a.count(`SELECT COUNT(*) FROM members WHERE id = 'm1'`) != 0 {
		t.Error("forged op applied")
	}
}
