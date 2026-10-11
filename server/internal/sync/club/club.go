// Package club implements the device lifecycle of the sync protocol
// (docs/sync-protocol.md §17) on top of the local SQLite state (§20):
// creating a club, inviting and approving devices, changing roles, revoking
// devices and recovering with the root key. It keeps the accepted ACL chain
// and this device's scope keys up to date (Refresh).
package club

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"filippo.io/age"
	"github.com/google/uuid"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/keys"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

var (
	// ErrNoClub is returned by Open when this device has neither created
	// nor joined a club.
	ErrNoClub = errors.New("club: device is not part of a club")
	// ErrAlreadyInClub is returned when creating or joining a second club.
	ErrAlreadyInClub = errors.New("club: device already belongs to a club")
	// ErrNotAdmin is returned for admin actions on a device without rw on
	// admin in the current ACL.
	ErrNotAdmin = errors.New("club: this device is not an admin device")
	// ErrNotMember is returned when the device is not active in the ACL.
	ErrNotMember = errors.New("club: this device is not an active member")
	// ErrMissingKey is returned when a scope key needed for an action is not
	// available on this device.
	ErrMissingKey = errors.New("club: scope key not available")
	// ErrKeyConflict is returned when a key file of a new epoch already
	// exists although this device did not write it.
	ErrKeyConflict = errors.New("club: key file of a new epoch already exists")
)

// Status of this device in its club, as stored in sync_club.
const (
	StatusJoining = "joining"
	StatusMember  = "member"
	StatusRevoked = "revoked"
)

// Club is this device's view of its club.
type Club struct {
	mu      sync.Mutex
	db      *sql.DB
	dev     *identity.Device
	tr      transport.Transport
	genesis *acl.Genesis
	chain   []*acl.Version
	status  string
	// Now is the wall clock, replaceable in tests.
	Now func() time.Time
}

// Genesis returns the pinned club.json.
func (c *Club) Genesis() *acl.Genesis { return c.genesis }

// Head returns the newest accepted ACL version, or nil while joining before
// any version was seen.
func (c *Club) Head() *acl.Version {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.head()
}

func (c *Club) head() *acl.Version {
	if len(c.chain) == 0 {
		return nil
	}
	return c.chain[len(c.chain)-1]
}

// Status returns StatusJoining, StatusMember or StatusRevoked.
func (c *Club) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Device returns this device.
func (c *Club) Device() *identity.Device { return c.dev }

// Create creates a new club (§17.1): it generates the root key, writes
// club.json, the keys of epoch 1 of every scope for this device and root, and
// ACL version 1 with this device as board.
//
// The returned kit holds the root private key. It is not stored anywhere; the
// caller shows kit.Text(), lets the user confirm they printed or saved it and
// then calls kit.Wipe().
func Create(ctx context.Context, conn *sql.DB, dev *identity.Device, tr transport.Transport, name, person string) (*Club, *identity.RecoveryKit, error) {
	if err := requireNoClub(ctx, conn); err != nil {
		return nil, nil, err
	}
	root, err := identity.Generate()
	if err != nil {
		return nil, nil, err
	}
	kit := &identity.RecoveryKit{Root: root, ClubName: name, CreatedAt: time.Now().UTC().Truncate(time.Second)}
	fail := func(err error) (*Club, *identity.RecoveryKit, error) {
		kit.Wipe()
		return nil, nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fail(err)
	}
	kit.ClubID = id.String()

	gRaw, gSig, err := acl.NewGenesis(kit.ClubID, name, kit.CreatedAt, root.Sign, root.AgeRecipient())
	if err != nil {
		return fail(err)
	}
	g, err := acl.ParseGenesis(gRaw, gSig)
	if err != nil {
		return fail(err)
	}
	if err := tr.Create(ctx, layout.GenesisSig, gSig); err != nil {
		return fail(fmt.Errorf("club: write %s: %w", layout.GenesisSig, err))
	}
	if err := tr.Create(ctx, layout.Genesis, gRaw); err != nil {
		return fail(fmt.Errorf("club: write %s: %w", layout.Genesis, err))
	}

	c := &Club{db: conn, dev: dev, tr: tr, genesis: g, status: StatusJoining, Now: time.Now}
	ts, err := c.tick(ctx)
	if err != nil {
		return fail(err)
	}
	first := &acl.ACL{
		Type:       "ocm.acl",
		ClubID:     g.ClubID,
		Version:    1,
		Author:     acl.AuthorRoot,
		CreatedHLC: ts.String(),
		Roles:      acl.DefaultRoles(),
		Devices: []acl.Device{{
			DeviceID: dev.ID, SignPub: dev.SignPubB64(), AgeRecipient: dev.AgeRecipient(),
			Person: person, Role: acl.RoleBoard, Status: acl.StatusActive, AddedIn: 1,
		}},
		Epochs:    map[db.Scope]uint64{},
		KeyChecks: map[db.Scope][]string{},
		MinSchema: 1,
	}
	for _, s := range db.Scopes {
		k, err := keys.New()
		if err != nil {
			return fail(err)
		}
		first.Epochs[s] = 1
		first.KeyChecks[s] = []string{keys.Check(s, 1, k)}
		for rid, rcpt := range map[string]string{dev.ID: dev.AgeRecipient(), layout.RootRecipient: root.AgeRecipient()} {
			if err := writeKey(ctx, tr, s, 1, rid, rcpt, k); err != nil {
				return fail(err)
			}
		}
		clear(k)
	}
	raw, sig, err := acl.Sign(first, root.Sign)
	if err != nil {
		return fail(err)
	}
	v, err := acl.Verify(g, nil, raw, sig)
	if err != nil {
		return fail(fmt.Errorf("club: own ACL version 1 is invalid: %w", err))
	}
	if err := publish(ctx, tr, v); err != nil {
		return fail(err)
	}

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO sync_club (id, club_id, name, root_sign_pub, root_age_recipient, genesis, genesis_sig, status, transport)
		 VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ClubID, g.Name, g.RootSignPub, g.RootAgeRecipient, gRaw, gSig, StatusJoining, `{"kind":"folder"}`); err != nil {
		return fail(err)
	}
	if _, err := c.Refresh(ctx); err != nil {
		return fail(err)
	}
	if c.Status() != StatusMember {
		return fail(errors.New("club: creating device is not a member of its own club"))
	}
	return c, kit, nil
}

func requireNoClub(ctx context.Context, conn *sql.DB) error {
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_club`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrAlreadyInClub
	}
	return nil
}

// Open loads the club this device created or joined. It uses only local
// state; call Refresh to pick up remote changes.
func Open(ctx context.Context, conn *sql.DB, dev *identity.Device, tr transport.Transport) (*Club, error) {
	var gRaw, gSig []byte
	var status string
	err := conn.QueryRowContext(ctx, `SELECT genesis, genesis_sig, status FROM sync_club WHERE id = 1`).Scan(&gRaw, &gSig, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoClub
	}
	if err != nil {
		return nil, err
	}
	g, err := acl.ParseGenesis(gRaw, gSig)
	if err != nil {
		return nil, err
	}
	c := &Club{db: conn, dev: dev, tr: tr, genesis: g, status: status, Now: time.Now}
	cached, err := c.cachedCandidates(ctx)
	if err != nil {
		return nil, err
	}
	chain, rejected := acl.Resolve(g, cached)
	if len(rejected) > 0 {
		return nil, fmt.Errorf("club: local ACL cache is inconsistent: %s: %w", rejected[0].Path, rejected[0].Err)
	}
	c.chain = chain
	return c, nil
}

// publish writes an ACL version: the signature first, so that a reader that
// sees the JSON file usually finds its signature too.
func publish(ctx context.Context, tr transport.Transport, v *acl.Version) error {
	if err := tr.Create(ctx, layout.SigPath(v.Path()), v.Sig); err != nil {
		return fmt.Errorf("club: write ACL signature: %w", err)
	}
	if err := tr.Create(ctx, v.Path(), v.Raw); err != nil {
		return fmt.Errorf("club: write ACL: %w", err)
	}
	return nil
}

// writeKey wraps k for one recipient and writes its key file.
func writeKey(ctx context.Context, tr transport.Transport, s db.Scope, epoch uint64, rid, recipient string, k []byte) error {
	w, err := keys.Wrap(k, recipient)
	if err != nil {
		return err
	}
	return tr.Create(ctx, layout.KeyPath(s, epoch, rid), w)
}

// tick advances the device's persisted HLC (§12) for a local event.
func (c *Club) tick(ctx context.Context) (hlc.Timestamp, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return hlc.Timestamp{}, err
	}
	defer tx.Rollback()
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT hlc FROM sync_device WHERE id = 1`).Scan(&stored); err != nil {
		return hlc.Timestamp{}, err
	}
	last, err := hlc.Parse(stored)
	if err != nil {
		return hlc.Timestamp{}, err
	}
	ts := hlc.NewWithSource(last, c.Now, time.Sleep).Now()
	if _, err := tx.ExecContext(ctx, `UPDATE sync_device SET hlc = ? WHERE id = 1`, ts.String()); err != nil {
		return hlc.Timestamp{}, err
	}
	return ts, tx.Commit()
}

func (c *Club) cachedCandidates(ctx context.Context) ([]acl.Candidate, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT version, body, sig FROM sync_acl ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []acl.Candidate
	for rows.Next() {
		var version uint64
		var cand acl.Candidate
		if err := rows.Scan(&version, &cand.Raw, &cand.Sig); err != nil {
			return nil, err
		}
		cand.Path = layout.ACLPath(version, sha256.Sum256(cand.Raw))
		out = append(out, cand)
	}
	return out, rows.Err()
}

// RefreshResult reports what Refresh found.
type RefreshResult struct {
	// Head is the newest accepted version (nil if none yet).
	Head *acl.Version
	// Reorged is true when a concurrent version with a smaller hash replaced
	// a version this device had accepted before (§8.3).
	Reorged bool
	// Rejected lists ACL files that are not part of the winning chain.
	Rejected []acl.Rejection
	// LostOwn lists versions authored by this device that lost against a
	// concurrent version; the user must redo those changes (§8.3).
	LostOwn []*acl.Version
	// MissingKeys lists scope keys this device may read but whose key file
	// has not arrived yet. Streams of these epochs wait (§13.2 rule 7).
	MissingKeys []KeyRef
	// BadKeys lists key files that do not decrypt or do not match the key
	// check in the ACL: a security warning.
	BadKeys []KeyRef
}

// KeyRef names one scope key.
type KeyRef struct {
	Scope db.Scope
	Epoch uint64
}

// Refresh reads club.json and the ACL files from the transport, resolves the
// winning chain (§8.3), enforces rollback protection (§8.4), updates the local
// state and fetches this device's scope keys (§9).
func (c *Club) Refresh(ctx context.Context) (*RefreshResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refresh(ctx)
}

func (c *Club) refresh(ctx context.Context) (*RefreshResult, error) {
	if err := c.checkRemoteGenesis(ctx); err != nil {
		return nil, err
	}
	cands, err := c.remoteCandidates(ctx)
	if err != nil {
		return nil, err
	}
	cached, err := c.cachedCandidates(ctx)
	if err != nil {
		return nil, err
	}
	chain, rejected := acl.Resolve(c.genesis, append(cands, cached...))

	var accepted uint64
	var acceptedHash sql.NullString
	if err := c.db.QueryRowContext(ctx, `SELECT acl_version, acl_hash FROM sync_club WHERE id = 1`).Scan(&accepted, &acceptedHash); err != nil {
		return nil, err
	}
	if err := acl.CheckRollback(chain, accepted); err != nil {
		return nil, err
	}

	res := &RefreshResult{Rejected: rejected}
	for _, r := range rejected {
		if r.Lost != nil && r.Lost.Author == c.dev.ID {
			res.LostOwn = append(res.LostOwn, r.Lost)
		}
	}
	if len(c.chain) > 0 {
		for i, v := range c.chain {
			if i >= len(chain) || chain[i].Sum != v.Sum {
				res.Reorged = true
				break
			}
		}
	}

	status := StatusJoining
	var head *acl.Version
	if len(chain) > 0 {
		head = chain[len(chain)-1]
		if d := head.Device(c.dev.ID); d != nil {
			status = StatusMember
			if !d.Active() {
				status = StatusRevoked
			}
		}
	}
	res.Head = head

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_acl`); err != nil {
		return nil, err
	}
	for _, v := range chain {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sync_acl (version, hash, body, sig) VALUES (?, ?, ?, ?)`,
			v.ACL.Version, v.Hash(), v.Raw, v.Sig); err != nil {
			return nil, err
		}
	}
	var headVersion uint64
	var headHash any
	if head != nil {
		headVersion, headHash = head.ACL.Version, head.Hash()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sync_club SET acl_version = ?, acl_hash = ?, status = ? WHERE id = 1`,
		headVersion, headHash, status); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	c.chain, c.status = chain, status

	if head != nil {
		if err := c.syncKeys(ctx, head, res); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// checkRemoteGenesis refuses a club.json in the sync root that differs from
// the pinned one (§7). A missing file is tolerated: the pinned copy is used.
func (c *Club) checkRemoteGenesis(ctx context.Context) error {
	raw, err := c.tr.Get(ctx, layout.Genesis)
	if errors.Is(err, transport.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	sig, err := c.tr.Get(ctx, layout.GenesisSig)
	if err != nil && !errors.Is(err, transport.ErrNotExist) {
		return err
	}
	g, err := acl.ParseGenesis(raw, sig)
	if err != nil {
		// Unsigned, forged or still being synced: never trusted, and the
		// pinned copy stays in force. Reported so the UI can warn.
		return err
	}
	return g.Matches(c.genesis.ClubID, c.genesis.RootSignPub)
}

// remoteCandidates lists acl/ and pairs every ACL file with its signature.
func (c *Club) remoteCandidates(ctx context.Context) ([]acl.Candidate, error) {
	files, err := c.tr.List(ctx, layout.ACLDir)
	if err != nil {
		return nil, err
	}
	sigs := map[string]bool{}
	for _, f := range files {
		if e, ok := layout.Parse(f.Path); ok && e.Kind == layout.KindACLSig {
			sigs[f.Path] = true
		}
	}
	var out []acl.Candidate
	for _, f := range files {
		e, ok := layout.Parse(f.Path)
		if !ok || e.Kind != layout.KindACL || !sigs[layout.SigPath(f.Path)] {
			// Unknown names are ignored (§6); an ACL file without its
			// signature is probably still being synced.
			continue
		}
		raw, err := c.tr.Get(ctx, f.Path)
		if errors.Is(err, transport.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sig, err := c.tr.Get(ctx, layout.SigPath(f.Path))
		if errors.Is(err, transport.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, acl.Candidate{Path: f.Path, Raw: raw, Sig: sig})
	}
	return out, nil
}

// syncKeys makes sure this device holds, for every scope it may read, the
// keys of all epochs up to the current one, each verified against the key
// check in the ACL. Keys of scopes it can no longer read are kept: they
// decrypt history the device has already seen.
func (c *Club) syncKeys(ctx context.Context, head *acl.Version, res *RefreshResult) error {
	if err := c.settleDrafts(ctx, head); err != nil {
		return err
	}
	for _, s := range db.Scopes {
		readable := head.Perm(c.dev.ID, s).CanRead()
		for e := uint64(1); e <= head.Epochs[s]; e++ {
			check := head.KeyChecks[s][e-1]
			wrapped, err := c.cachedKey(ctx, s, e)
			if err != nil {
				return err
			}
			if wrapped != nil {
				if c.verifyWrapped(wrapped, s, e, check) == nil {
					continue
				}
				// Cached key does not match this chain (e.g. after a reorg).
				if _, err := c.db.ExecContext(ctx, `DELETE FROM sync_keys WHERE scope = ? AND epoch = ?`, s, e); err != nil {
					return err
				}
			}
			if !readable {
				continue
			}
			wrapped, err = c.tr.Get(ctx, layout.KeyPath(s, e, c.dev.ID))
			if errors.Is(err, transport.ErrNotExist) {
				res.MissingKeys = append(res.MissingKeys, KeyRef{s, e})
				continue
			}
			if err != nil {
				return err
			}
			if c.verifyWrapped(wrapped, s, e, check) != nil {
				res.BadKeys = append(res.BadKeys, KeyRef{s, e})
				continue
			}
			if _, err := c.db.ExecContext(ctx, `INSERT INTO sync_keys (scope, epoch, wrapped) VALUES (?, ?, ?)`, s, e, wrapped); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Club) cachedKey(ctx context.Context, s db.Scope, e uint64) ([]byte, error) {
	var w []byte
	err := c.db.QueryRowContext(ctx, `SELECT wrapped FROM sync_keys WHERE scope = ? AND epoch = ?`, s, e).Scan(&w)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

func (c *Club) verifyWrapped(wrapped []byte, s db.Scope, e uint64, check string) error {
	k, err := keys.Unwrap(wrapped, c.dev.Age)
	if err != nil {
		return err
	}
	defer clear(k)
	return keys.Verify(s, e, k, check)
}

// settleDrafts drops epoch key drafts once the accepted chain contains their
// epoch, with this key or with a concurrent admin's one.
func (c *Club) settleDrafts(ctx context.Context, head *acl.Version) error {
	for _, s := range db.Scopes {
		if _, err := c.db.ExecContext(ctx, `DELETE FROM sync_key_drafts WHERE scope = ? AND epoch <= ?`,
			s, head.Epochs[s]); err != nil {
			return err
		}
	}
	return nil
}

// ScopeKey returns K(scope, epoch), verified against the accepted ACL.
func (c *Club) ScopeKey(ctx context.Context, s db.Scope, epoch uint64) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scopeKey(ctx, s, epoch, nil)
}

// scopeKey returns K(scope, epoch) from this device's keys or, if root is
// given, from the root's key file.
func (c *Club) scopeKey(ctx context.Context, s db.Scope, epoch uint64, root age.Identity) ([]byte, error) {
	head := c.head()
	if head == nil || epoch < 1 || epoch > head.Epochs[s] {
		return nil, fmt.Errorf("%w: %s epoch %d", ErrMissingKey, s, epoch)
	}
	check := head.KeyChecks[s][epoch-1]
	wrapped, err := c.cachedKey(ctx, s, epoch)
	if err != nil {
		return nil, err
	}
	if wrapped != nil {
		if k, err := keys.Unwrap(wrapped, c.dev.Age); err == nil && keys.Verify(s, epoch, k, check) == nil {
			return k, nil
		}
	}
	if root != nil {
		w, err := c.tr.Get(ctx, layout.KeyPath(s, epoch, layout.RootRecipient))
		if err == nil {
			if k, err := keys.Unwrap(w, root); err == nil && keys.Verify(s, epoch, k, check) == nil {
				return k, nil
			}
		} else if !errors.Is(err, transport.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%w: %s epoch %d", ErrMissingKey, s, epoch)
}
