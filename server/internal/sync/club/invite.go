package club

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// DefaultInviteTTL is how long an invite stays valid (§17.2).
const DefaultInviteTTL = 7 * 24 * time.Hour

var (
	ErrBadInviteCode = errors.New("club: malformed invite code")
	ErrUnknownRole   = errors.New("club: unknown role")
	ErrInviteUnknown = errors.New("club: request refers to an unknown invite")
	ErrInviteExpired = errors.New("club: invite has expired")
	ErrInviteUsed    = errors.New("club: invite has already been used")
	ErrBadProof      = errors.New("club: invite proof does not verify")
	ErrBadRequest    = errors.New("club: malformed or forged join request")
	ErrAlreadyListed = errors.New("club: device is already in the ACL")
	// ErrContested is set on every request when two or more requests use the
	// same invite. Someone else holds the invite code; the admin should
	// create a new invite (see the PR notes on safety code grinding).
	ErrContested = errors.New("club: several join requests use the same invite")
	ErrNoRequest = errors.New("club: no such join request")
)

// Invite is an invite created by this admin device.
type Invite struct {
	ID        []byte
	Secret    []byte
	Role      string
	Person    string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    string
	// Code is the text to hand to the new member (QR later). Only set on
	// the invite returned by CreateInvite.
	Code string
}

// Sizes of the invite parts (§17.2 step 1). Together they are 15 bytes,
// exactly 24 base32 characters.
const (
	inviteIDSize     = 5
	inviteSecretSize = 10
)

var inviteEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// FormatInviteCode returns the invite code of §17.2: invite_id ||
// invite_secret as 24 characters of uppercase base32 (RFC 4648 alphabet
// A–Z, 2–7) in groups of four, e.g. "AB2C-DEF3-…". The code names neither
// the club nor the storage: the new device reads club.json from the storage
// location the user selects.
func FormatInviteCode(id, secret []byte) string {
	s := inviteEncoding.EncodeToString(append(slices.Clone(id), secret...))
	var groups []string
	for i := 0; i < len(s); i += 4 {
		groups = append(groups, s[i:min(i+4, len(s))])
	}
	return strings.Join(groups, "-")
}

// ParseInviteCode returns invite_id and invite_secret. Case, dashes and
// white space are ignored, so a code read out over the phone or copied with
// line breaks still works.
func ParseInviteCode(code string) (id, secret []byte, err error) {
	s := strings.ToUpper(strings.Join(strings.FieldsFunc(code, func(r rune) bool {
		return r == '-' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}), ""))
	if len(s) != 24 {
		return nil, nil, ErrBadInviteCode
	}
	b, err := inviteEncoding.DecodeString(s)
	if err != nil || len(b) != inviteIDSize+inviteSecretSize {
		return nil, nil, ErrBadInviteCode
	}
	return b[:inviteIDSize], b[inviteIDSize:], nil
}

// CreateInvite creates an invite for a role (§17.2 step 1). The invite is
// stored only locally. ttl <= 0 means DefaultInviteTTL.
func (c *Club) CreateInvite(ctx context.Context, role, person string, ttl time.Duration) (*Invite, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	head := c.head()
	if head == nil || !head.IsAdmin(c.dev.ID) {
		return nil, ErrNotAdmin
	}
	if _, ok := head.Roles[role]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownRole, role)
	}
	if ttl <= 0 {
		ttl = DefaultInviteTTL
	}
	inv := &Invite{ID: make([]byte, inviteIDSize), Secret: make([]byte, inviteSecretSize), Role: role, Person: person}
	if _, err := rand.Read(inv.ID); err != nil {
		return nil, err
	}
	if _, err := rand.Read(inv.Secret); err != nil {
		return nil, err
	}
	inv.CreatedAt = c.Now().UTC()
	inv.ExpiresAt = inv.CreatedAt.Add(ttl)
	if _, err := c.db.ExecContext(ctx,
		`INSERT INTO sync_invites (invite_id, secret, role, person, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		wire.EncodeB64(inv.ID), inv.Secret, role, person, db.FormatTime(inv.CreatedAt), db.FormatTime(inv.ExpiresAt)); err != nil {
		return nil, err
	}
	inv.Code = FormatInviteCode(inv.ID, inv.Secret)
	return inv, nil
}

// Invites lists the invites this device created, newest first.
func (c *Club) Invites(ctx context.Context) ([]Invite, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT invite_id, secret, role, person, created_at, expires_at, used_at, used_by FROM sync_invites ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invite
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanInvite(r scanner) (*Invite, error) {
	var id, created, expires string
	var usedAt, usedBy sql.NullString
	inv := &Invite{}
	if err := r.Scan(&id, &inv.Secret, &inv.Role, &inv.Person, &created, &expires, &usedAt, &usedBy); err != nil {
		return nil, err
	}
	var err error
	if inv.ID, err = wire.DecodeB64(id); err != nil {
		return nil, err
	}
	if inv.CreatedAt, err = parseTime(created); err != nil {
		return nil, err
	}
	if inv.ExpiresAt, err = parseTime(expires); err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t, err := parseTime(usedAt.String)
		if err != nil {
			return nil, err
		}
		inv.UsedAt = &t
	}
	inv.UsedBy = usedBy.String
	return inv, nil
}

func parseTime(s string) (time.Time, error) {
	t, ok := db.ParseTime(s)
	if !ok {
		return time.Time{}, fmt.Errorf("club: malformed time %q", s)
	}
	return t, nil
}

// joinRequest is requests/<device_id>.json (§17.2 step 3).
type joinRequest struct {
	Type         string `json:"type"`
	ClubID       string `json:"club_id"`
	InviteID     string `json:"invite_id"`
	DeviceID     string `json:"device_id"`
	SignPub      string `json:"sign_pub"`
	AgeRecipient string `json:"age_recipient"`
	Person       string `json:"person"`
	Proof        string `json:"proof"`
}

const joinType = "ocm.join"

// Proof computes HMAC-SHA-256(invite_secret, invite_id || sign_pub ||
// age_recipient) over the raw invite id, the raw 32-byte public key and the
// UTF-8 recipient string. Both leading fields have fixed lengths, so the
// concatenation is unambiguous.
func Proof(secret, inviteID []byte, signPub ed25519.PublicKey, ageRecipient string) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write(inviteID)
	m.Write(signPub)
	m.Write([]byte(ageRecipient))
	return m.Sum(nil)
}

// Join sends a join request with an invite code (§17.2 step 3) to the club
// whose files are in tr, the storage location the user selected. It pins
// that club.json (trust on first use: the code does not name the club) and
// returns the club in status joining plus the safety code to
// compare with the admin. Refresh picks up the approval.
func Join(ctx context.Context, conn *sql.DB, dev *identity.Device, tr transport.Transport, code, person string) (*Club, string, error) {
	inviteID, secret, err := ParseInviteCode(code)
	if err != nil {
		return nil, "", err
	}
	// One club per device (v1), and a joining device starts empty (§15.4).
	if err := requireNoClub(ctx, conn); err != nil {
		return nil, "", err
	}
	if err := requireEmptySyncedData(ctx, conn); err != nil {
		return nil, "", err
	}
	g, gRaw, gSig, err := fetchGenesis(ctx, tr)
	if err != nil {
		return nil, "", err
	}
	req := joinRequest{
		Type: joinType, ClubID: g.ClubID, InviteID: wire.EncodeB64(inviteID), DeviceID: dev.ID,
		SignPub: dev.SignPubB64(), AgeRecipient: dev.AgeRecipient(), Person: person,
		Proof: wire.EncodeB64(Proof(secret, inviteID, dev.SignPub(), dev.AgeRecipient())),
	}
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, "", err
	}
	path := layout.RequestPath(dev.ID)
	if err := tr.Create(ctx, layout.SigPath(path), wire.Sign(dev.Sign, raw)); err != nil {
		return nil, "", fmt.Errorf("club: write join request: %w", err)
	}
	if err := tr.Create(ctx, path, raw); err != nil {
		return nil, "", fmt.Errorf("club: write join request: %w", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO sync_club (id, club_id, name, root_sign_pub, root_age_recipient, genesis, genesis_sig, status, join_invite_id)
		 VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ClubID, g.Name, g.RootSignPub, g.RootAgeRecipient, gRaw, gSig, StatusJoining, req.InviteID); err != nil {
		return nil, "", err
	}
	c := &Club{db: conn, dev: dev, tr: tr, genesis: g, status: StatusJoining, Now: time.Now}
	return c, identity.SafetyCode(dev.SignPub(), dev.AgeRecipient()), nil
}

func fetchGenesis(ctx context.Context, tr transport.Transport) (*acl.Genesis, []byte, []byte, error) {
	raw, err := tr.Get(ctx, layout.Genesis)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("club: read %s: %w", layout.Genesis, err)
	}
	sig, err := tr.Get(ctx, layout.GenesisSig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("club: read %s: %w", layout.GenesisSig, err)
	}
	g, err := acl.ParseGenesis(raw, sig)
	if err != nil {
		return nil, nil, nil, err
	}
	return g, raw, sig, nil
}

// JoinRequest is a pending join request as seen by an admin device.
type JoinRequest struct {
	DeviceID     string
	SignPub      string
	AgeRecipient string
	Person       string
	InviteID     string
	// Role is the role of the invite, if the invite is known.
	Role string
	// SafetyCode is the 6-digit code to compare with the new member.
	SafetyCode string
	// Err is nil if the request can be approved; otherwise it says why not.
	Err error
}

// JoinRequests lists and checks the pending join requests (§17.2 step 4).
// Requests whose file is not yet complete are skipped.
func (c *Club) JoinRequests(ctx context.Context) ([]JoinRequest, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.joinRequests(ctx)
}

func (c *Club) joinRequests(ctx context.Context) ([]JoinRequest, error) {
	head := c.head()
	if head == nil || !head.IsAdmin(c.dev.ID) {
		return nil, ErrNotAdmin
	}
	files, err := c.tr.List(ctx, layout.RequestsDir)
	if err != nil {
		return nil, err
	}
	var out []JoinRequest
	byInvite := map[string][]int{}
	for _, f := range files {
		e, ok := layout.Parse(f.Path)
		if !ok || e.Kind != layout.KindRequest {
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
		r := c.checkRequest(ctx, head, e.DeviceID, raw, sig)
		out = append(out, r)
		if r.Err == nil {
			byInvite[r.InviteID] = append(byInvite[r.InviteID], len(out)-1)
		}
	}
	for _, idx := range byInvite {
		if len(idx) > 1 {
			for _, i := range idx {
				out[i].Err = ErrContested
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeviceID < out[j].DeviceID })
	return out, nil
}

// checkRequest verifies one request: format, file name, self-signature,
// invite (known, unexpired, unused) and HMAC proof.
func (c *Club) checkRequest(ctx context.Context, head *acl.Version, fileDeviceID string, raw, sig []byte) JoinRequest {
	out := JoinRequest{DeviceID: fileDeviceID}
	var req joinRequest
	if err := wire.DecodeStrict(raw, &req); err != nil || req.Type != joinType {
		out.Err = ErrBadRequest
		return out
	}
	pub, err := wire.ParsePublicKey(req.SignPub)
	if err != nil || req.DeviceID != fileDeviceID || identity.DeviceID(pub) != req.DeviceID ||
		req.ClubID != c.genesis.ClubID {
		out.Err = ErrBadRequest
		return out
	}
	if _, err := age.ParseX25519Recipient(req.AgeRecipient); err != nil {
		out.Err = ErrBadRequest
		return out
	}
	if wire.Verify(pub, raw, sig) != nil {
		out.Err = ErrBadRequest
		return out
	}
	out.SignPub, out.AgeRecipient, out.Person, out.InviteID = req.SignPub, req.AgeRecipient, req.Person, req.InviteID
	out.SafetyCode = identity.SafetyCode(pub, req.AgeRecipient)
	if head.Device(req.DeviceID) != nil {
		out.Err = ErrAlreadyListed
		return out
	}

	inv, err := scanInvite(c.db.QueryRowContext(ctx,
		`SELECT invite_id, secret, role, person, created_at, expires_at, used_at, used_by FROM sync_invites WHERE invite_id = ?`,
		req.InviteID))
	if errors.Is(err, sql.ErrNoRows) {
		out.Err = ErrInviteUnknown
		return out
	}
	if err != nil {
		out.Err = err
		return out
	}
	out.Role = inv.Role
	proof, err := wire.DecodeB64(req.Proof)
	if err != nil || !hmac.Equal(proof, Proof(inv.Secret, inv.ID, pub, req.AgeRecipient)) {
		out.Err = ErrBadProof
		return out
	}
	switch {
	case inv.UsedAt != nil:
		out.Err = ErrInviteUsed
	case !c.Now().Before(inv.ExpiresAt):
		out.Err = ErrInviteExpired
	}
	return out
}

// Approve admits the device of a valid join request (§17.2 step 5): a new ACL
// version adds it with the invite's role, the scope keys of all epochs it
// may read are wrapped for it, the invite is marked used and the request
// file is deleted. The admin must have compared the safety code first.
func (c *Club) Approve(ctx context.Context, deviceID string) (*acl.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.refresh(ctx); err != nil {
		return nil, err
	}
	reqs, err := c.joinRequests(ctx)
	if err != nil {
		return nil, err
	}
	var req *JoinRequest
	for i := range reqs {
		if reqs[i].DeviceID == deviceID {
			req = &reqs[i]
		}
	}
	if req == nil {
		return nil, ErrNoRequest
	}
	if req.Err != nil {
		return nil, req.Err
	}
	v, err := c.mutate(ctx, nil, func(next *acl.ACL) error {
		if _, ok := next.Roles[req.Role]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownRole, req.Role)
		}
		next.Devices = append(next.Devices, acl.Device{
			DeviceID: req.DeviceID, SignPub: req.SignPub, AgeRecipient: req.AgeRecipient,
			Person: req.Person, Role: req.Role, Status: acl.StatusActive, AddedIn: next.Version,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, err := c.db.ExecContext(ctx, `UPDATE sync_invites SET used_at = ?, used_by = ? WHERE invite_id = ?`,
		db.FormatTime(c.Now().UTC()), req.DeviceID, req.InviteID); err != nil {
		return nil, err
	}
	return v, c.deleteRequest(ctx, req.DeviceID)
}

// RejectRequest deletes a join request without admitting the device.
func (c *Club) RejectRequest(ctx context.Context, deviceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	head := c.head()
	if head == nil || !head.IsAdmin(c.dev.ID) {
		return ErrNotAdmin
	}
	if !layout.ValidDeviceID(deviceID) {
		return ErrNoRequest
	}
	return c.deleteRequest(ctx, deviceID)
}

func (c *Club) deleteRequest(ctx context.Context, deviceID string) error {
	path := layout.RequestPath(deviceID)
	if err := c.tr.Delete(ctx, path); err != nil {
		return err
	}
	return c.tr.Delete(ctx, layout.SigPath(path))
}
