// Package layout builds and parses the file names of the sync root
// (docs/sync-protocol.md §6).
//
// Readers MUST ignore every file whose name does not match one of the
// patterns exactly. Parse is that filter: temporary files, ".tmp-*" files and
// conflict copies made by sync clients ("000000000001 (conflicted copy).seg",
// "000000000001-Konflikt.seg", ...) do not parse and are therefore ignored.
package layout

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Kind is the type of a file in the sync root.
type Kind int

const (
	KindGenesis Kind = iota + 1
	KindGenesisSig
	KindACL
	KindACLSig
	KindKey
	KindRequest
	KindRequestSig
	KindSegment
	KindSnapshot
)

const (
	Genesis    = "club.json"
	GenesisSig = "club.json.sig"
	// SigSuffix is appended to the name of a signed JSON file to name its
	// detached signature.
	SigSuffix = ".sig"

	// RootRecipient is the recipient id of the root key in key file names.
	RootRecipient = "root"

	ACLDir      = "acl/"
	KeysDir     = "keys/"
	RequestsDir = "requests/"
)

// Entry is a parsed file name. Only the fields of its Kind are set.
type Entry struct {
	Kind Kind
	// Version and HashPrefix (16 lowercase hex characters) of an ACL file.
	Version    uint64
	HashPrefix string
	// DeviceID of a request, segment or snapshot.
	DeviceID string
	Scope    db.Scope
	Epoch    uint64
	// Recipient of a key file: a device id or "root".
	Recipient string
	Seq       uint64
	HLC       string
}

const (
	deviceIDPat = `[a-z2-7]{26}`
	hlcPat      = `[0-9a-f]{12}-[0-9a-f]{4}`
	// Epochs are plain decimal numbers without leading zeros, starting at 1.
	epochPat = `[1-9][0-9]{0,17}`
)

var (
	deviceIDRe = regexp.MustCompile(`^` + deviceIDPat + `$`)
	aclRe      = regexp.MustCompile(`^acl/([0-9]{8})-([0-9a-f]{16})\.json(\.sig)?$`)
	keyRe      = regexp.MustCompile(`^keys/([a-z.]+)/(` + epochPat + `)/(` + deviceIDPat + `|root)\.age$`)
	requestRe  = regexp.MustCompile(`^requests/(` + deviceIDPat + `)\.json(\.sig)?$`)
	segmentRe  = regexp.MustCompile(`^segments/(` + deviceIDPat + `)/([a-z.]+)/([0-9]{12})\.seg$`)
	snapshotRe = regexp.MustCompile(`^snapshots/([a-z.]+)/(` + epochPat + `)/(` + hlcPat + `)-(` + deviceIDPat + `)\.snap$`)
)

// ValidDeviceID reports whether id has the form of §5.1.
func ValidDeviceID(id string) bool { return deviceIDRe.MatchString(id) }

// ValidScope reports whether s is a scope of protocol version 1.
func ValidScope(s db.Scope) bool {
	for _, k := range db.Scopes {
		if k == s {
			return true
		}
	}
	return false
}

// Parse classifies a path relative to the sync root. It returns false for
// every name that does not match a pattern of §6 exactly.
func Parse(path string) (Entry, bool) {
	switch path {
	case Genesis:
		return Entry{Kind: KindGenesis}, true
	case GenesisSig:
		return Entry{Kind: KindGenesisSig}, true
	}
	if m := aclRe.FindStringSubmatch(path); m != nil {
		v, ok := parseUint(m[1])
		if !ok || v == 0 {
			return Entry{}, false
		}
		k := KindACL
		if m[3] != "" {
			k = KindACLSig
		}
		return Entry{Kind: k, Version: v, HashPrefix: m[2]}, true
	}
	if m := keyRe.FindStringSubmatch(path); m != nil {
		e, ok := parseUint(m[2])
		if !ok || !ValidScope(db.Scope(m[1])) {
			return Entry{}, false
		}
		return Entry{Kind: KindKey, Scope: db.Scope(m[1]), Epoch: e, Recipient: m[3]}, true
	}
	if m := requestRe.FindStringSubmatch(path); m != nil {
		k := KindRequest
		if m[2] != "" {
			k = KindRequestSig
		}
		return Entry{Kind: k, DeviceID: m[1]}, true
	}
	if m := segmentRe.FindStringSubmatch(path); m != nil {
		seq, ok := parseUint(m[3])
		if !ok || seq == 0 || !ValidScope(db.Scope(m[2])) {
			return Entry{}, false
		}
		return Entry{Kind: KindSegment, DeviceID: m[1], Scope: db.Scope(m[2]), Seq: seq}, true
	}
	if m := snapshotRe.FindStringSubmatch(path); m != nil {
		e, ok := parseUint(m[2])
		if !ok || !ValidScope(db.Scope(m[1])) {
			return Entry{}, false
		}
		return Entry{Kind: KindSnapshot, Scope: db.Scope(m[1]), Epoch: e, HLC: m[3], DeviceID: m[4]}, true
	}
	return Entry{}, false
}

func parseUint(s string) (uint64, bool) {
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil
}

// ACLHashPrefix returns the <hash> part of an ACL file name: the first 16
// characters of the lowercase hex SHA-256 of the file.
func ACLHashPrefix(sum [32]byte) string { return hex.EncodeToString(sum[:])[:16] }

// ACLPath returns acl/<version>-<hash>.json.
func ACLPath(version uint64, sum [32]byte) string {
	return fmt.Sprintf("acl/%08d-%s.json", version, ACLHashPrefix(sum))
}

// KeyPath returns keys/<scope>/<epoch>/<recipient>.age.
func KeyPath(scope db.Scope, epoch uint64, recipient string) string {
	return fmt.Sprintf("keys/%s/%d/%s.age", scope, epoch, recipient)
}

// RequestPath returns requests/<device_id>.json.
func RequestPath(deviceID string) string { return RequestsDir + deviceID + ".json" }

// SegmentPath returns segments/<device_id>/<scope>/<seq>.seg.
func SegmentPath(deviceID string, scope db.Scope, seq uint64) string {
	return fmt.Sprintf("segments/%s/%s/%012d.seg", deviceID, scope, seq)
}

// SnapshotPath returns snapshots/<scope>/<epoch>/<hlc>-<device_id>.snap.
func SnapshotPath(scope db.Scope, epoch uint64, hlc, deviceID string) string {
	return fmt.Sprintf("snapshots/%s/%d/%s-%s.snap", scope, epoch, hlc, deviceID)
}

// SigPath returns the name of the detached signature of a JSON file.
func SigPath(path string) string { return path + SigSuffix }

// IsSig reports whether path names a detached signature.
func IsSig(path string) bool { return strings.HasSuffix(path, SigSuffix) }
