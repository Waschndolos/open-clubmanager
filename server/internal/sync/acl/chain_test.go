package acl

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
)

func cand(v *Version) Candidate { return Candidate{Path: v.Path(), Raw: v.Raw, Sig: v.Sig} }

// competing returns two valid versions following prev, ordered by hash.
func (f *fixture) competing(prev *Version) (small, large *Version) {
	f.t.Helper()
	x := prev.Next(f.a.ID, hlc.Timestamp{MS: 100})
	x.Devices[0].Person = "x"
	y := prev.Next(f.a.ID, hlc.Timestamp{MS: 101})
	y.Devices[0].Person = "y"
	vx, vy := f.mustVerify(prev, x, f.a.Sign), f.mustVerify(prev, y, f.a.Sign)
	if bytes.Compare(vx.Sum[:], vy.Sum[:]) < 0 {
		return vx, vy
	}
	return vy, vx
}

func TestResolveLinear(t *testing.T) {
	f := newFixture(t)
	chain, rej := Resolve(f.g, []Candidate{cand(f.v2), cand(f.v1)})
	if len(rej) != 0 || len(chain) != 2 || chain[1].Sum != f.v2.Sum {
		t.Fatalf("chain %d, rejected %v", len(chain), rej)
	}
	// Duplicates (the same file from the folder and the local cache).
	chain, rej = Resolve(f.g, []Candidate{cand(f.v1), cand(f.v2), cand(f.v1), cand(f.v2)})
	if len(rej) != 0 || len(chain) != 2 {
		t.Fatalf("duplicates: chain %d, rejected %v", len(chain), rej)
	}
}

func TestResolveConcurrentSmallestHashWins(t *testing.T) {
	f := newFixture(t)
	small, large := f.competing(f.v2)
	// A version 4 that builds on the losing version 3.
	onLoser := f.mustVerify(large, large.Next(f.a.ID, hlc.Timestamp{MS: 200}), f.a.Sign)
	// And one on the winner.
	onWinner := f.mustVerify(small, small.Next(f.a.ID, hlc.Timestamp{MS: 201}), f.a.Sign)

	orders := [][]Candidate{
		{cand(f.v1), cand(f.v2), cand(small), cand(large), cand(onLoser), cand(onWinner)},
		{cand(onLoser), cand(large), cand(onWinner), cand(small), cand(f.v2), cand(f.v1)},
	}
	for i, cands := range orders {
		chain, rej := Resolve(f.g, cands)
		if len(chain) != 4 || chain[2].Sum != small.Sum || chain[3].Sum != onWinner.Sum {
			t.Fatalf("order %d: wrong chain", i)
		}
		var lost, badPrev int
		for _, r := range rej {
			switch {
			case errors.Is(r.Err, ErrLostConcurrent):
				lost++
				if r.Lost == nil || r.Lost.Sum != large.Sum {
					t.Errorf("order %d: lost version not reported", i)
				}
			case errors.Is(r.Err, ErrBadPrev):
				badPrev++
				if r.Path != onLoser.Path() {
					t.Errorf("order %d: unexpected rejection %s", i, r.Path)
				}
			default:
				t.Errorf("order %d: unexpected rejection %v", i, r.Err)
			}
		}
		if lost != 1 || badPrev != 1 {
			t.Errorf("order %d: lost %d, bad prev %d", i, lost, badPrev)
		}
	}
}

func TestResolveIgnoresForgeries(t *testing.T) {
	f := newFixture(t)
	forger := newDevice(t)
	valid := f.mustVerify(f.v2, f.v2.Next(f.a.ID, hlc.Timestamp{MS: 300}), f.a.Sign)

	// A forged version 3 by an unknown key, with a hash that may be smaller.
	forged := f.v2.Next(f.a.ID, hlc.Timestamp{MS: 301})
	forged.Roles[RoleCoach] = DefaultRoles()[RoleBoard]
	fRaw, fSig := f.sign(forged, forger.Sign)
	// An unsigned version 3 (signature of other content).
	unsigned := f.v2.Next(f.a.ID, hlc.Timestamp{MS: 302})
	uRaw, _ := f.sign(unsigned, f.a.Sign)
	// The revoked admin C tries to undo its revocation.
	byRevoked := f.v2.Next(f.c.ID, hlc.Timestamp{MS: 303})
	rRaw, rSig := f.sign(byRevoked, f.c.Sign)

	cands := []Candidate{
		// The valid file under a wrong version, listed before the real one:
		// it must not shadow it.
		{Path: layout.ACLPath(4, valid.Sum), Raw: valid.Raw, Sig: valid.Sig},
		cand(f.v1), cand(f.v2), cand(valid),
		{Path: layout.ACLPath(3, sum(fRaw)), Raw: fRaw, Sig: fSig},
		{Path: layout.ACLPath(3, sum(uRaw)), Raw: uRaw, Sig: valid.Sig},
		{Path: layout.ACLPath(3, sum(rRaw)), Raw: rRaw, Sig: rSig},
		{Path: "acl/00000003-0000000000000000.json", Raw: valid.Raw, Sig: valid.Sig},
		{Path: "acl/00000003-abc (conflicted copy).json", Raw: valid.Raw, Sig: valid.Sig},
	}
	chain, rej := Resolve(f.g, cands)
	if len(chain) != 3 || chain[2].Sum != valid.Sum {
		t.Fatalf("chain has %d versions", len(chain))
	}
	want := map[error]int{ErrBadSignature: 2, ErrNotAuthorized: 1, ErrMalformed: 2, ErrBadPrev: 1}
	got := map[error]int{}
	for _, r := range rej {
		for e := range want {
			if errors.Is(r.Err, e) {
				got[e]++
			}
		}
	}
	for e, n := range want {
		if got[e] != n {
			t.Errorf("%v: %d rejections, want %d (all: %v)", e, got[e], n, rej)
		}
	}
}

func TestResolveRequiresGenesisRoot(t *testing.T) {
	f := newFixture(t)
	other := newFixture(t) // another club's root, same club id
	chain, rej := Resolve(other.g, []Candidate{cand(f.v1), cand(f.v2)})
	if len(chain) != 0 || len(rej) != 2 {
		t.Fatalf("chain %d, rejections %d", len(chain), len(rej))
	}
}

func TestCheckRollback(t *testing.T) {
	f := newFixture(t)
	chain, _ := Resolve(f.g, []Candidate{cand(f.v1), cand(f.v2)})
	if err := CheckRollback(chain, 2); err != nil {
		t.Error(err)
	}
	if err := CheckRollback(chain, 1); err != nil {
		t.Error(err)
	}
	if err := CheckRollback(chain[:1], 2); !errors.Is(err, ErrRollback) {
		t.Errorf("rollback to 1: %v", err)
	}
	if err := CheckRollback(nil, 1); !errors.Is(err, ErrRollback) {
		t.Errorf("empty chain: %v", err)
	}
	if err := CheckRollback(nil, 0); err != nil {
		t.Error(err)
	}
}

func sum(b []byte) [32]byte { return sha256.Sum256(b) }
