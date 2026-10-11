package acl

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
)

// Candidate is an ACL file found in the sync root or in the local cache.
type Candidate struct {
	// Path is acl/<version>-<hash>.json. Its version and hash prefix must
	// match the content.
	Path string
	Raw  []byte
	Sig  []byte
}

// Rejection explains why a candidate is not part of the winning chain.
type Rejection struct {
	Path string
	Err  error
	// Lost is set for a valid version that lost against a concurrent one
	// (Err is ErrLostConcurrent), so its author can be told to redo it.
	Lost *Version
}

// Resolve builds the winning chain from genesis (§8.3): starting at version 1,
// it verifies every candidate of the next version against the current head
// and continues with the valid one that has the lexicographically smallest
// file hash. Candidates that are invalid, or that build on a losing version,
// are returned as rejections. Duplicate candidates (same content) are fine.
//
// The spec writes <hash> in file names as hex; comparing the full SHA-256
// digests bytewise is the same order as comparing the full hex strings.
func Resolve(g *Genesis, cands []Candidate) ([]*Version, []Rejection) {
	byVersion := map[uint64][]Candidate{}
	var rejected []Rejection
	type key struct {
		version uint64
		sum     [32]byte
	}
	seen := map[key]bool{}
	for _, c := range cands {
		e, ok := layout.Parse(c.Path)
		if !ok || e.Kind != layout.KindACL {
			rejected = append(rejected, Rejection{Path: c.Path, Err: fmt.Errorf("%w: file name", ErrMalformed)})
			continue
		}
		sum := sha256.Sum256(c.Raw)
		if layout.ACLHashPrefix(sum) != e.HashPrefix {
			rejected = append(rejected, Rejection{Path: c.Path, Err: fmt.Errorf("%w: hash in file name does not match content", ErrMalformed)})
			continue
		}
		if seen[key{e.Version, sum}] {
			continue
		}
		seen[key{e.Version, sum}] = true
		byVersion[e.Version] = append(byVersion[e.Version], c)
	}

	var chain []*Version
	var head *Version
	for n := uint64(1); ; n++ {
		var winner *Version
		var valid []*Version
		for _, c := range byVersion[n] {
			v, err := Verify(g, head, c.Raw, c.Sig)
			if err == nil && v.ACL.Version != n {
				err = fmt.Errorf("%w: version in file name does not match content", ErrMalformed)
			}
			if err != nil {
				rejected = append(rejected, Rejection{Path: c.Path, Err: err})
				continue
			}
			valid = append(valid, v)
			if winner == nil || bytes.Compare(v.Sum[:], winner.Sum[:]) < 0 {
				winner = v
			}
		}
		for _, v := range valid {
			if v != winner {
				rejected = append(rejected, Rejection{Path: v.Path(), Err: ErrLostConcurrent, Lost: v})
			}
		}
		if winner == nil {
			// Anything above n cannot link to the chain any more.
			var rest []uint64
			for k := range byVersion {
				if k > n {
					rest = append(rest, k)
				}
			}
			sort.Slice(rest, func(i, j int) bool { return rest[i] < rest[j] })
			for _, k := range rest {
				for _, c := range byVersion[k] {
					rejected = append(rejected, Rejection{Path: c.Path, Err: fmt.Errorf("%w: no valid version %d", ErrBadPrev, n)})
				}
			}
			return chain, rejected
		}
		chain = append(chain, winner)
		head = winner
	}
}

// CheckRollback enforces §8.4: a chain must not end below the highest version
// this device has accepted before.
func CheckRollback(chain []*Version, highestAccepted uint64) error {
	var head uint64
	if len(chain) > 0 {
		head = chain[len(chain)-1].ACL.Version
	}
	if head < highestAccepted {
		return fmt.Errorf("%w: chain ends at %d, accepted %d", ErrRollback, head, highestAccepted)
	}
	return nil
}
