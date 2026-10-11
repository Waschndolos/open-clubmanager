package club

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/keys"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

var (
	ErrUnknownDevice = errors.New("club: device is not in the ACL")
	ErrDeviceRevoked = errors.New("club: device is revoked; add a new device instead")
	// ErrLastAdmin protects against locking the club out by accident. The
	// recovery kit could still restore administration.
	ErrLastAdmin = errors.New("club: change would leave no active admin device")
	// ErrKitMismatch is returned for a recovery kit of another club.
	ErrKitMismatch = errors.New("club: recovery kit does not belong to this club")
)

// keyJob is one key file to write before an ACL version is published.
type keyJob struct {
	scope     db.Scope
	epoch     uint64
	recipient string // device id or "root"
	ageRcpt   string
	key       []byte
	// strict jobs belong to a freshly generated epoch: an existing file at
	// their path was written by someone else.
	strict bool
}

// mutate creates, signs and publishes the next ACL version. fn edits the
// draft; mutate then derives the key changes of §9 from the difference in
// readers per scope:
//   - a scope where some device lost read access gets a new epoch, wrapped
//     for all remaining readers and root (§9, §17.3, §17.4 step 4);
//   - devices that gained read access receive the keys of all existing
//     epochs (§9, §17.2 step 5, §17.3).
//
// Key files are written before the ACL, so a device that finds itself in
// the ACL also finds its keys. If root is set, the version is signed by the
// root key (§17.5) and missing keys are taken from the root's key files.
// c.mu must be held.
func (c *Club) mutate(ctx context.Context, root *identity.KeyPair, fn func(next *acl.ACL) error) (*acl.Version, error) {
	if _, err := c.refresh(ctx); err != nil {
		return nil, err
	}
	head := c.head()
	if head == nil {
		return nil, ErrNotMember
	}
	author, signer := c.dev.ID, c.dev.Sign
	var rootAge age.Identity
	if root != nil {
		author, signer, rootAge = acl.AuthorRoot, root.Sign, root.Age
	} else if !head.IsAdmin(c.dev.ID) {
		return nil, ErrNotAdmin
	}
	ts, err := c.tick(ctx)
	if err != nil {
		return nil, err
	}
	next := head.Next(author, ts)
	if err := fn(next); err != nil {
		return nil, err
	}

	var jobs []keyJob
	defer func() {
		for _, j := range jobs {
			clear(j.key)
		}
	}()
	for _, s := range db.Scopes {
		before, after := head.Readers(s), next.Readers(s)
		lost := slices.ContainsFunc(before, func(id string) bool { return !slices.Contains(after, id) })
		if lost {
			e := head.Epochs[s] + 1
			k, fresh, err := c.draftKey(ctx, s, e)
			if err != nil {
				return nil, err
			}
			next.Epochs[s] = e
			next.KeyChecks[s] = append(slices.Clone(head.KeyChecks[s]), keys.Check(s, e, k))
			for _, id := range after {
				jobs = append(jobs, keyJob{s, e, id, next.Device(id).AgeRecipient, slices.Clone(k), fresh})
			}
			jobs = append(jobs, keyJob{s, e, layout.RootRecipient, c.genesis.RootAgeRecipient, slices.Clone(k), fresh})
			clear(k)
		}
		for _, id := range after {
			if slices.Contains(before, id) {
				continue
			}
			for e := uint64(1); e <= head.Epochs[s]; e++ {
				k, err := c.scopeKey(ctx, s, e, rootAge)
				if err != nil {
					return nil, err
				}
				jobs = append(jobs, keyJob{s, e, id, next.Device(id).AgeRecipient, k, false})
			}
		}
	}

	raw, sig, err := acl.Sign(next, signer)
	if err != nil {
		return nil, err
	}
	v, err := acl.Verify(c.genesis, head, raw, sig)
	if err != nil {
		return nil, fmt.Errorf("club: new ACL version would be invalid: %w", err)
	}
	for _, j := range jobs {
		err := writeKey(ctx, c.tr, j.scope, j.epoch, j.recipient, j.ageRcpt, j.key)
		switch {
		case err == nil:
		case errors.Is(err, transport.ErrExists) && j.strict:
			return nil, fmt.Errorf("%w: %s", ErrKeyConflict, layout.KeyPath(j.scope, j.epoch, j.recipient))
		case errors.Is(err, transport.ErrExists):
			// Written by an earlier attempt, or the device had this key
			// before. The receiver verifies it against the key check.
		default:
			return nil, err
		}
	}
	if err := publish(ctx, c.tr, v); err != nil {
		return nil, err
	}
	if _, err := c.refresh(ctx); err != nil {
		return nil, err
	}
	if n := v.ACL.Version; uint64(len(c.chain)) < n || c.chain[n-1].Sum != v.Sum {
		return nil, acl.ErrLostConcurrent
	}
	return v, nil
}

// draftKey returns the key for a new epoch: the draft from an earlier
// attempt if there is one, otherwise a fresh key, which is stored as a draft
// before any file is written.
func (c *Club) draftKey(ctx context.Context, s db.Scope, e uint64) (k []byte, fresh bool, err error) {
	var w []byte
	err = c.db.QueryRowContext(ctx, `SELECT wrapped FROM sync_key_drafts WHERE scope = ? AND epoch = ?`, s, e).Scan(&w)
	if err == nil {
		k, err := keys.Unwrap(w, c.dev.Age)
		return k, false, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if k, err = keys.New(); err != nil {
		return nil, false, err
	}
	if w, err = keys.Wrap(k, c.dev.AgeRecipient()); err != nil {
		return nil, false, err
	}
	if _, err := c.db.ExecContext(ctx, `INSERT INTO sync_key_drafts (scope, epoch, wrapped) VALUES (?, ?, ?)`, s, e, w); err != nil {
		return nil, false, err
	}
	return k, true, nil
}

func countAdmins(a *acl.ACL) int {
	n := 0
	for _, d := range a.Devices {
		if a.IsAdmin(d.DeviceID) {
			n++
		}
	}
	return n
}

// ChangeRole gives an active device another role (§17.3). Scopes it can no
// longer read get a new epoch; scopes it can now read are wrapped for it.
func (c *Club) ChangeRole(ctx context.Context, deviceID, role string) (*acl.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mutate(ctx, nil, func(next *acl.ACL) error {
		d := next.Device(deviceID)
		switch {
		case d == nil:
			return ErrUnknownDevice
		case !d.Active():
			return ErrDeviceRevoked
		}
		if _, ok := next.Roles[role]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownRole, role)
		}
		d.Role = role
		if countAdmins(next) == 0 {
			return ErrLastAdmin
		}
		return nil
	})
}

// Revoke revokes a device (§17.4). cutoff is the highest accepted seq per
// scope; if nil, it is taken from sync_streams, and every scope the device
// could write gets an entry (0 if nothing was accepted). Every scope the
// device could read gets a new epoch.
func (c *Club) Revoke(ctx context.Context, deviceID string, cutoff map[db.Scope]uint64) (*acl.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.revoke(ctx, nil, deviceID, cutoff)
}

func (c *Club) revoke(ctx context.Context, root *identity.KeyPair, deviceID string, cutoff map[db.Scope]uint64) (*acl.Version, error) {
	if cutoff == nil {
		head := c.head()
		if head == nil {
			return nil, ErrNotMember
		}
		var err error
		if cutoff, err = c.acceptedCutoff(ctx, head, deviceID); err != nil {
			return nil, err
		}
	}
	return c.mutate(ctx, root, func(next *acl.ACL) error {
		d := next.Device(deviceID)
		switch {
		case d == nil:
			return ErrUnknownDevice
		case !d.Active():
			return ErrDeviceRevoked
		}
		d.Status, d.RevokedIn, d.Cutoff = acl.StatusRevoked, next.Version, cutoff
		if root == nil && countAdmins(next) == 0 {
			return ErrLastAdmin
		}
		return nil
	})
}

// acceptedCutoff returns, per scope, the highest seq this device has
// accepted from deviceID (§17.4 step 2), covering every scope with a stream
// and every scope the device may currently write.
func (c *Club) acceptedCutoff(ctx context.Context, head *acl.Version, deviceID string) (map[db.Scope]uint64, error) {
	cutoff := map[db.Scope]uint64{}
	for _, s := range db.Scopes {
		if head.Perm(deviceID, s).CanWrite() {
			cutoff[s] = 0
		}
	}
	rows, err := c.db.QueryContext(ctx, `SELECT scope, seq FROM sync_streams WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var seq uint64
		if err := rows.Scan(&s, &seq); err != nil {
			return nil, err
		}
		if layout.ValidScope(db.Scope(s)) {
			cutoff[db.Scope(s)] = seq
		}
	}
	return cutoff, rows.Err()
}

// Recover makes this device an admin device with the recovery kit (§17.5):
// a version signed by the root key adds the device (or changes its role) to
// an admin role, and the keys it lacks are taken from the root's key files.
// The root key is wiped from memory afterwards.
func (c *Club) Recover(ctx context.Context, kitText, person string) (*acl.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	kit, err := c.openKit(kitText)
	if err != nil {
		return nil, err
	}
	defer kit.Wipe()
	return c.mutate(ctx, kit.Root, func(next *acl.ACL) error {
		role := adminRole(next)
		d := next.Device(c.dev.ID)
		switch {
		case d == nil:
			next.Devices = append(next.Devices, acl.Device{
				DeviceID: c.dev.ID, SignPub: c.dev.SignPubB64(), AgeRecipient: c.dev.AgeRecipient(),
				Person: person, Role: role, Status: acl.StatusActive, AddedIn: next.Version,
			})
		case !d.Active():
			return ErrDeviceRevoked
		default:
			d.Role = role
		}
		return nil
	})
}

// RevokeWithRoot revokes a device with the recovery kit, e.g. a compromised
// admin device when no other admin device is left (§21).
func (c *Club) RevokeWithRoot(ctx context.Context, kitText, deviceID string, cutoff map[db.Scope]uint64) (*acl.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	kit, err := c.openKit(kitText)
	if err != nil {
		return nil, err
	}
	defer kit.Wipe()
	return c.revoke(ctx, kit.Root, deviceID, cutoff)
}

// openKit parses a recovery kit and checks it against the pinned genesis.
func (c *Club) openKit(text string) (*identity.RecoveryKit, error) {
	kit, err := identity.ParseRecoveryKit(text)
	if err != nil {
		return nil, err
	}
	if kit.ClubID != c.genesis.ClubID || kit.Root.SignPubB64() != c.genesis.RootSignPub ||
		kit.Root.AgeRecipient() != c.genesis.RootAgeRecipient {
		kit.Wipe()
		return nil, ErrKitMismatch
	}
	return kit, nil
}

// adminRole returns the role a recovering device gets: board if it still
// grants rw on admin, otherwise the first role (by id) that does. If no role
// does, board is restored with its default permissions.
func adminRole(a *acl.ACL) string {
	if a.Roles[acl.RoleBoard][db.ScopeAdmin].CanWrite() {
		return acl.RoleBoard
	}
	ids := make([]string, 0, len(a.Roles))
	for id := range a.Roles {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if a.Roles[id][db.ScopeAdmin].CanWrite() {
			return id
		}
	}
	a.Roles[acl.RoleBoard] = acl.DefaultRoles()[acl.RoleBoard]
	return acl.RoleBoard
}

// RecoverNew joins a club with the recovery kit on a device that has no club
// yet, e.g. after all admin devices were lost (§17.5 "or a new device").
func RecoverNew(ctx context.Context, conn *sql.DB, dev *identity.Device, tr transport.Transport, kitText, person string) (*Club, *acl.Version, error) {
	if err := requireNoClub(ctx, conn); err != nil {
		return nil, nil, err
	}
	g, gRaw, gSig, err := fetchGenesis(ctx, tr)
	if err != nil {
		return nil, nil, err
	}
	c := &Club{db: conn, dev: dev, tr: tr, genesis: g, status: StatusJoining, Now: time.Now}
	kit, err := c.openKit(kitText)
	if err != nil {
		return nil, nil, err
	}
	kit.Wipe()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO sync_club (id, club_id, name, root_sign_pub, root_age_recipient, genesis, genesis_sig, status)
		 VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
		g.ClubID, g.Name, g.RootSignPub, g.RootAgeRecipient, gRaw, gSig, StatusJoining); err != nil {
		return nil, nil, err
	}
	v, err := c.Recover(ctx, kitText, person)
	if err != nil {
		return nil, nil, err
	}
	return c, v, nil
}
