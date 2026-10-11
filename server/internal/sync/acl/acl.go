// Package acl implements the club genesis file and the signed, versioned
// access control list (docs/sync-protocol.md §4, §7, §8).
package acl

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

const aclType = "ocm.acl"

// AuthorRoot is the author of ACL versions signed with the root key.
const AuthorRoot = "root"

// Device statuses.
const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

// Device is one entry of the ACL's device list.
type Device struct {
	DeviceID     string `json:"device_id"`
	SignPub      string `json:"sign_pub"`
	AgeRecipient string `json:"age_recipient"`
	Person       string `json:"person"`
	// DeviceName is the name the person gave the device ("Laptop").
	DeviceName string `json:"device_name,omitempty"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	AddedIn    uint64 `json:"added_in"`
	RevokedIn  uint64 `json:"revoked_in,omitempty"`
	// Cutoff is, per scope the device could write immediately before its
	// revocation, the last stream position that is still accepted
	// (§8.2 rule 9, §13.2 rule 6).
	Cutoff map[db.Scope]Cutoff `json:"cutoff,omitempty"`
}

// Cutoff is the terminal position of a revoked device's stream: the highest
// contiguous, verified segment the revoking admin had accepted and the
// complete-file hash of that segment. Seq 0 (nothing accepted) has a nil Hash.
type Cutoff struct {
	Seq  uint64  `json:"seq"`
	Hash *string `json:"hash"`
}

// Equal reports whether two cutoffs are the same.
func (c Cutoff) Equal(o Cutoff) bool {
	if c.Seq != o.Seq || (c.Hash == nil) != (o.Hash == nil) {
		return false
	}
	return c.Hash == nil || *c.Hash == *o.Hash
}

// Active reports whether the device is active.
func (d *Device) Active() bool { return d.Status == StatusActive }

// ACL is one version of the access control list (§8.1).
type ACL struct {
	Type       string              `json:"type"`
	ClubID     string              `json:"club_id"`
	Version    uint64              `json:"version"`
	Prev       *string             `json:"prev"`
	Author     string              `json:"author"`
	CreatedHLC string              `json:"created_hlc"`
	Roles      map[string]Role     `json:"roles"`
	Devices    []Device            `json:"devices"`
	Epochs     map[db.Scope]uint64 `json:"epochs"`
	// ClockEpochs starts a new conflict-ordering generation per scope
	// (§8.1, §12): changes ordered by (clock_epoch, hlc, device_id).
	ClockEpochs map[db.Scope]uint64 `json:"clock_epochs"`
	// KeyChecks commits to the scope key of every epoch: KeyChecks[S][e-1]
	// is keys.Check(S, e, K(S,e)). This field is an addition to §8.1, see
	// keys.Check for why it is needed.
	KeyChecks map[db.Scope][]string `json:"key_checks"`
	MinSchema int                   `json:"min_schema"`
}

// Device returns the entry of a device, or nil.
func (a *ACL) Device(id string) *Device {
	for i := range a.Devices {
		if a.Devices[i].DeviceID == id {
			return &a.Devices[i]
		}
	}
	return nil
}

// Perm returns the permission of a device on a scope. Revoked and unknown
// devices have none.
func (a *ACL) Perm(deviceID string, s db.Scope) Perm {
	d := a.Device(deviceID)
	if d == nil || !d.Active() {
		return ""
	}
	return a.Roles[d.Role][s]
}

// IsAdmin reports whether the device is an active admin device, i.e. has rw
// on admin (§4.2).
func (a *ACL) IsAdmin(deviceID string) bool { return a.Perm(deviceID, db.ScopeAdmin).CanWrite() }

// Readers returns the ids of the active devices that may read s.
func (a *ACL) Readers(s db.Scope) []string {
	var out []string
	for _, d := range a.Devices {
		if d.Active() && a.Roles[d.Role][s].CanRead() {
			out = append(out, d.DeviceID)
		}
	}
	return out
}

// Clone returns a deep copy.
func (a *ACL) Clone() *ACL {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	var c ACL
	if err := json.Unmarshal(b, &c); err != nil {
		panic(err)
	}
	return &c
}

// Encode returns the file content of the ACL.
func (a *ACL) Encode() ([]byte, error) { return json.MarshalIndent(a, "", "  ") }

// Version is an accepted ACL version: the parsed list plus the exact bytes it
// was verified over.
type Version struct {
	*ACL
	Raw []byte
	Sig []byte
	Sum [32]byte
}

// Hash returns the "sha256:<b64>" form used in prev.
func (v *Version) Hash() string { return wire.FormatHash(v.Sum) }

// Path returns acl/<version>-<hash>.json.
func (v *Version) Path() string { return layout.ACLPath(v.ACL.Version, v.Sum) }

// Next returns a copy of v's list as the draft of the following version.
func (v *Version) Next(author string, created hlc.Timestamp) *ACL {
	n := v.ACL.Clone()
	n.Version = v.ACL.Version + 1
	prev := v.Hash()
	n.Prev = &prev
	n.Author = author
	n.CreatedHLC = created.String()
	return n
}

// Sign encodes a and signs it with the author's key.
func Sign(a *ACL, priv ed25519.PrivateKey) (raw, sig []byte, err error) {
	raw, err = a.Encode()
	if err != nil {
		return nil, nil, err
	}
	return raw, wire.Sign(priv, raw), nil
}

// Validation errors. Each maps to a rule of §8.2 or to the format of §8.1.
var (
	ErrMalformed      = errors.New("acl: malformed")
	ErrWrongClub      = errors.New("acl: belongs to another club")
	ErrBadPrev        = errors.New("acl: prev does not match the previous version")      // rule 1, 2
	ErrNotAuthorized  = errors.New("acl: author may not sign this version")              // rule 1, 3
	ErrBadSignature   = errors.New("acl: signature does not verify")                     // rule 1, 3
	ErrEpochDecreased = errors.New("acl: epochs decrease")                               // rule 4
	ErrKeyChecks      = errors.New("acl: key checks changed or do not match epochs")     // rule 4 (addition)
	ErrDeviceChanged  = errors.New("acl: device removed, altered or reactivated")        // rule 5
	ErrScopeDeps      = errors.New("acl: role violates scope dependencies")              // rule 6
	ErrCutoff         = errors.New("acl: revocation cutoff invalid")                     // rule 9
	ErrLostConcurrent = errors.New("acl: lost against a concurrent version")             // §8.3
	ErrRollback       = errors.New("acl: chain ends below the highest accepted version") // §8.4
)

// Verify checks raw/sig as the version following prev (nil for version 1)
// against all rules of §8.2.
func Verify(g *Genesis, prev *Version, raw, sig []byte) (*Version, error) {
	var a ACL
	if err := wire.DecodeStrict(raw, &a); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if a.Type != aclType {
		return nil, fmt.Errorf("%w: type %q", ErrMalformed, a.Type)
	}
	if a.ClubID != g.ClubID {
		return nil, ErrWrongClub
	}

	// Rules 1 and 2: chain position.
	var signer ed25519.PublicKey
	if prev == nil {
		if a.Version != 1 || a.Prev != nil {
			return nil, fmt.Errorf("%w: version 1 must have prev null", ErrBadPrev)
		}
		if a.Author != AuthorRoot {
			return nil, fmt.Errorf("%w: version 1 must be signed by root", ErrNotAuthorized)
		}
	} else {
		if a.Version != prev.ACL.Version+1 {
			return nil, fmt.Errorf("%w: version %d after %d", ErrBadPrev, a.Version, prev.ACL.Version)
		}
		if a.Prev == nil || *a.Prev != prev.Hash() {
			return nil, ErrBadPrev
		}
	}

	// Rule 3: author and signature.
	if a.Author == AuthorRoot {
		signer = g.RootPub()
	} else {
		if prev == nil || !prev.IsAdmin(a.Author) {
			return nil, fmt.Errorf("%w: %s is not an active admin device in version %d", ErrNotAuthorized, a.Author, a.Version-1)
		}
		pub, err := wire.ParsePublicKey(prev.Device(a.Author).SignPub)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
		}
		signer = pub
	}
	if err := wire.Verify(signer, raw, sig); err != nil {
		return nil, ErrBadSignature
	}

	if err := checkFormat(&a); err != nil {
		return nil, err
	}
	var prevACL *ACL
	if prev != nil {
		prevACL = prev.ACL
	}
	if err := checkEpochs(prevACL, &a); err != nil {
		return nil, err
	}
	if err := checkDevices(prevACL, &a); err != nil {
		return nil, err
	}
	// Rule 6.
	for id, r := range a.Roles {
		if err := r.CheckDependencies(); err != nil {
			return nil, fmt.Errorf("%w: role %s %v", ErrScopeDeps, id, err)
		}
	}
	return &Version{ACL: &a, Raw: raw, Sig: sig, Sum: sha256.Sum256(raw)}, nil
}

// checkFormat validates the fields of one version on their own.
func checkFormat(a *ACL) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrMalformed}, args...)...)
	}
	if a.Author != AuthorRoot && !layout.ValidDeviceID(a.Author) {
		return bad("author %q", a.Author)
	}
	if _, err := hlc.Parse(a.CreatedHLC); err != nil {
		return bad("created_hlc: %v", err)
	}
	if a.MinSchema < 1 {
		return bad("min_schema %d", a.MinSchema)
	}
	for id, r := range a.Roles {
		if id == "" {
			return bad("empty role id")
		}
		for s, p := range r {
			if !layout.ValidScope(s) {
				return bad("role %s: unknown scope %q", id, s)
			}
			if p != Read && p != ReadWrite {
				return bad("role %s: permission %q on %s", id, p, s)
			}
		}
	}
	if len(a.Epochs) != len(db.Scopes) {
		return bad("epochs must list every scope")
	}
	for _, s := range db.Scopes {
		if a.Epochs[s] < 1 {
			return bad("epoch of %s", s)
		}
	}
	if len(a.ClockEpochs) != len(db.Scopes) {
		return bad("clock_epochs must list every scope")
	}
	for _, s := range db.Scopes {
		if a.ClockEpochs[s] < 1 {
			return bad("clock epoch of %s", s)
		}
	}
	seen := map[string]bool{}
	for _, d := range a.Devices {
		if seen[d.DeviceID] {
			return bad("device %s listed twice", d.DeviceID)
		}
		seen[d.DeviceID] = true
		pub, err := wire.ParsePublicKey(d.SignPub)
		if err != nil {
			return bad("device %s sign_pub: %v", d.DeviceID, err)
		}
		if identity.DeviceID(pub) != d.DeviceID {
			return bad("device_id %s does not match its sign_pub", d.DeviceID)
		}
		if _, err := age.ParseX25519Recipient(d.AgeRecipient); err != nil {
			return bad("device %s age_recipient: %v", d.DeviceID, err)
		}
		if d.AddedIn < 1 || d.AddedIn > a.Version {
			return bad("device %s added_in %d", d.DeviceID, d.AddedIn)
		}
		switch d.Status {
		case StatusActive:
			if d.RevokedIn != 0 || d.Cutoff != nil {
				return bad("active device %s has revocation fields", d.DeviceID)
			}
			if _, ok := a.Roles[d.Role]; !ok {
				return bad("device %s has unknown role %q", d.DeviceID, d.Role)
			}
		case StatusRevoked:
			if d.RevokedIn < d.AddedIn || d.RevokedIn > a.Version {
				return bad("device %s revoked_in %d", d.DeviceID, d.RevokedIn)
			}
			for s, c := range d.Cutoff {
				if !layout.ValidScope(s) {
					return bad("device %s cutoff for unknown scope %q", d.DeviceID, s)
				}
				if c.Seq == 0 && c.Hash != nil {
					return bad("device %s cutoff %s: seq 0 must have hash null", d.DeviceID, s)
				}
				if c.Seq != 0 {
					if c.Hash == nil {
						return bad("device %s cutoff %s: hash missing", d.DeviceID, s)
					}
					if _, err := wire.ParseHash(*c.Hash); err != nil {
						return bad("device %s cutoff %s: %v", d.DeviceID, s, err)
					}
				}
			}
		default:
			return bad("device %s status %q", d.DeviceID, d.Status)
		}
	}
	return nil
}

// checkEpochs implements rule 4, plus the key check commitments: one per
// epoch, never changed once published.
func checkEpochs(prev, a *ACL) error {
	for _, s := range db.Scopes {
		if prev != nil && a.Epochs[s] < prev.Epochs[s] {
			return fmt.Errorf("%w: %s from %d to %d", ErrEpochDecreased, s, prev.Epochs[s], a.Epochs[s])
		}
		checks := a.KeyChecks[s]
		if uint64(len(checks)) != a.Epochs[s] {
			return fmt.Errorf("%w: %s has %d checks for %d epochs", ErrKeyChecks, s, len(checks), a.Epochs[s])
		}
		for _, c := range checks {
			if _, err := wire.ParseHash(c); err != nil {
				return fmt.Errorf("%w: %v", ErrKeyChecks, err)
			}
		}
		if prev != nil && !slices.Equal(checks[:len(prev.KeyChecks[s])], prev.KeyChecks[s]) {
			return fmt.Errorf("%w: %s", ErrKeyChecks, s)
		}
		// Rule 8.
		if prev != nil && a.ClockEpochs[s] < prev.ClockEpochs[s] {
			return fmt.Errorf("%w: clock epoch of %s from %d to %d", ErrEpochDecreased, s, prev.ClockEpochs[s], a.ClockEpochs[s])
		}
	}
	if len(a.KeyChecks) != len(db.Scopes) {
		return fmt.Errorf("%w: unknown scope", ErrKeyChecks)
	}
	return nil
}

// checkDevices implements rule 5: devices are never removed, their keys and
// added_in never change, and revoked devices stay revoked with the same
// revoked_in and cutoff. New devices enter as active in this version.
func checkDevices(prev, a *ACL) error {
	if prev == nil {
		for _, d := range a.Devices {
			if d.AddedIn != 1 || !d.Active() {
				return fmt.Errorf("%w: device %s in version 1", ErrDeviceChanged, d.DeviceID)
			}
		}
		return nil
	}
	for _, old := range prev.Devices {
		d := a.Device(old.DeviceID)
		switch {
		case d == nil:
			return fmt.Errorf("%w: %s removed", ErrDeviceChanged, old.DeviceID)
		case d.SignPub != old.SignPub || d.AgeRecipient != old.AgeRecipient || d.AddedIn != old.AddedIn:
			return fmt.Errorf("%w: keys of %s changed", ErrDeviceChanged, old.DeviceID)
		case !old.Active() && d.Active():
			return fmt.Errorf("%w: %s reactivated", ErrDeviceChanged, old.DeviceID)
		case !old.Active() && (d.RevokedIn != old.RevokedIn || !maps.EqualFunc(d.Cutoff, old.Cutoff, Cutoff.Equal)):
			return fmt.Errorf("%w: revocation of %s changed", ErrDeviceChanged, old.DeviceID)
		case old.Active() && !d.Active() && d.RevokedIn != a.Version:
			return fmt.Errorf("%w: %s revoked with revoked_in %d", ErrDeviceChanged, old.DeviceID, d.RevokedIn)
		case old.Active() && !d.Active():
			// Rule 9: exactly one cutoff per scope the device could write
			// immediately before its revocation.
			for _, s := range db.Scopes {
				_, has := d.Cutoff[s]
				if has != prev.Perm(old.DeviceID, s).CanWrite() {
					return fmt.Errorf("%w: %s scope %s", ErrCutoff, old.DeviceID, s)
				}
			}
		}
	}
	for _, d := range a.Devices {
		if prev.Device(d.DeviceID) == nil && (d.AddedIn != a.Version || !d.Active()) {
			return fmt.Errorf("%w: new device %s must be active with added_in %d", ErrDeviceChanged, d.DeviceID, a.Version)
		}
	}
	return nil
}
