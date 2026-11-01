package club

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// This file lets a joining device learn the outcome of its request without
// waiting forever (issue #194):
//
//   - Invite markers: the admin device publishes invites/<marker>.json for
//     every invite it created, where marker is a hash of the complete invite
//     code. A joining device refuses a code whose marker it cannot find, so a
//     mistyped code is reported at once. The marker reveals neither the
//     invite id nor the secret: finding a code for it means a 120-bit
//     preimage search. It is unsigned and only a hint; the admin's checks of
//     §17.2 step 4 stay authoritative.
//   - Rejection notices: rejecting a request writes
//     requests/<device_id>.rejected.json, signed by the rejecting admin
//     device and bound to the club, the device and the invite id. The
//     joining device verifies it against the ACL and leaves the club.
//   - CancelJoin withdraws a pending request.

var (
	// ErrJoinRejected is returned by Refresh on a joining device whose
	// request was rejected by an admin. The local club state has been
	// removed; the device is no longer part of a club.
	ErrJoinRejected = errors.New("club: the join request was rejected")
	// ErrNotJoining is returned by CancelJoin when no join request is
	// pending (the device is a member already, or not in a club).
	ErrNotJoining = errors.New("club: no join request is pending")
)

const (
	inviteMarkerType = "ocm.invite"
	rejectionType    = "ocm.join.rejected"
)

// inviteMarker is invites/<marker>.json.
type inviteMarker struct {
	Type      string `json:"type"`
	ClubID    string `json:"club_id"`
	ExpiresAt string `json:"expires_at"`
	Used      bool   `json:"used,omitempty"`
}

// InviteMarker returns the name of the marker file of an invite: the
// lowercase hex SHA-256 of "ocm.invite-marker.v1\0" || invite_id ||
// invite_secret.
func InviteMarker(id, secret []byte) string {
	h := sha256.New()
	h.Write([]byte("ocm.invite-marker.v1\x00"))
	h.Write(id)
	h.Write(secret)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Club) markerBytes(inv *Invite) []byte {
	raw, _ := json.Marshal(inviteMarker{
		Type: inviteMarkerType, ClubID: c.genesis.ClubID,
		ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339), Used: inv.UsedAt != nil,
	})
	return raw
}

// publishInviteMarker makes the marker of inv match the invite: present while
// it is open or used but not expired, absent once it has expired.
func (c *Club) publishInviteMarker(ctx context.Context, inv *Invite) error {
	path := layout.InvitePath(InviteMarker(inv.ID, inv.Secret))
	if !c.Now().Before(inv.ExpiresAt) {
		return c.tr.Delete(ctx, path)
	}
	want := c.markerBytes(inv)
	have, err := c.tr.Get(ctx, path)
	if err == nil && bytes.Equal(have, want) {
		return nil
	}
	if err != nil && !errors.Is(err, transport.ErrNotExist) {
		return err
	}
	if err == nil {
		if err := c.tr.Delete(ctx, path); err != nil {
			return err
		}
	}
	if err := c.tr.Create(ctx, path, want); err != nil && !errors.Is(err, transport.ErrExists) {
		return fmt.Errorf("club: write invite marker: %w", err)
	}
	return nil
}

// publishInviteMarkerByID updates the marker of the invite with the b64 id.
func (c *Club) publishInviteMarkerByID(ctx context.Context, inviteID string) error {
	inv, err := scanInvite(c.db.QueryRowContext(ctx,
		`SELECT invite_id, secret, role, person, created_at, expires_at, used_at, used_by FROM sync_invites WHERE invite_id = ?`,
		inviteID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.publishInviteMarker(ctx, inv)
}

// syncInviteMarkers brings the markers of all invites of this device up to
// date. It also publishes markers of invites created before markers existed.
func (c *Club) syncInviteMarkers(ctx context.Context) error {
	rows, err := c.db.QueryContext(ctx,
		`SELECT invite_id, secret, role, person, created_at, expires_at, used_at, used_by FROM sync_invites`)
	if err != nil {
		return err
	}
	var invs []*Invite
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			rows.Close()
			return err
		}
		invs = append(invs, inv)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, inv := range invs {
		if err := c.publishInviteMarker(ctx, inv); err != nil {
			return err
		}
	}
	return nil
}

// checkInviteMarker is the joining device's check of an invite code against
// the markers in the club's storage.
func checkInviteMarker(ctx context.Context, tr transport.Transport, g *acl.Genesis, id, secret []byte, now time.Time) error {
	raw, err := tr.Get(ctx, layout.InvitePath(InviteMarker(id, secret)))
	if errors.Is(err, transport.ErrNotExist) {
		return ErrInviteUnknown
	}
	if err != nil {
		return err
	}
	var m inviteMarker
	if err := json.Unmarshal(raw, &m); err != nil || m.Type != inviteMarkerType || m.ClubID != g.ClubID {
		return ErrInviteUnknown
	}
	if m.Used {
		return ErrInviteUsed
	}
	if exp, err := time.Parse(time.RFC3339, m.ExpiresAt); err == nil && !now.Before(exp) {
		return ErrInviteExpired
	}
	return nil
}

// joinRejection is requests/<device_id>.rejected.json.
type joinRejection struct {
	Type     string `json:"type"`
	ClubID   string `json:"club_id"`
	DeviceID string `json:"device_id"`
	InviteID string `json:"invite_id"`
	By       string `json:"by"`
}

// writeRejection tells the device of a rejected request about the rejection.
// Callers hold c.mu.
func (c *Club) writeRejection(ctx context.Context, deviceID string) error {
	// The request is read leniently: a malformed request is rejected too,
	// and its device learns about it if it can be matched at all.
	reqRaw, err := c.tr.Get(ctx, layout.RequestPath(deviceID))
	if errors.Is(err, transport.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var req joinRequest
	_ = json.Unmarshal(reqRaw, &req)
	raw, err := json.MarshalIndent(joinRejection{
		Type: rejectionType, ClubID: c.genesis.ClubID, DeviceID: deviceID, InviteID: req.InviteID, By: c.dev.ID,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := c.deleteRejection(ctx, deviceID); err != nil {
		return err
	}
	path := layout.RejectionPath(deviceID)
	if err := c.tr.Create(ctx, layout.SigPath(path), wire.Sign(c.dev.Sign, raw)); err != nil {
		return fmt.Errorf("club: write rejection: %w", err)
	}
	if err := c.tr.Create(ctx, path, raw); err != nil {
		return fmt.Errorf("club: write rejection: %w", err)
	}
	return nil
}

func (c *Club) deleteRejection(ctx context.Context, deviceID string) error {
	path := layout.RejectionPath(deviceID)
	if err := c.tr.Delete(ctx, path); err != nil {
		return err
	}
	return c.tr.Delete(ctx, layout.SigPath(path))
}

// rejected reports whether a valid rejection of this device's pending request
// is in the storage: signed by an active admin of head, for this club, this
// device and the invite it used.
func (c *Club) rejected(ctx context.Context, head *acl.Version) (bool, error) {
	if head == nil {
		return false, nil
	}
	path := layout.RejectionPath(c.dev.ID)
	raw, err := c.tr.Get(ctx, path)
	if errors.Is(err, transport.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	sig, err := c.tr.Get(ctx, layout.SigPath(path))
	if errors.Is(err, transport.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var r joinRejection
	if err := wire.DecodeStrict(raw, &r); err != nil || r.Type != rejectionType ||
		r.ClubID != c.genesis.ClubID || r.DeviceID != c.dev.ID {
		return false, nil
	}
	var inviteID sql.NullString
	if err := c.db.QueryRowContext(ctx, `SELECT join_invite_id FROM sync_club WHERE id = 1`).Scan(&inviteID); err != nil {
		return false, err
	}
	if r.InviteID != inviteID.String {
		return false, nil
	}
	admin := head.Device(r.By)
	if admin == nil || !admin.Active() || !head.IsAdmin(r.By) {
		return false, nil
	}
	pub, err := wire.ParsePublicKey(admin.SignPub)
	if err != nil || wire.Verify(pub, raw, sig) != nil {
		return false, nil
	}
	return true, nil
}

// joinStateTables are the local tables a joining device may have written
// since its request. Synced data tables are not touched: a joining device
// starts empty and gets no write access before it is admitted.
var joinStateTables = []string{
	"sync_acl", "sync_keys", "sync_key_drafts", "sync_streams", "sync_scopes", "sync_pending",
	"sync_history", "sync_uploads", "sync_clock_epochs", "sync_clock_warnings", "sync_field_clock",
	"sync_conflicts", "sync_outbox", "sync_club",
}

// AbandonJoin removes the local state of a pending join request, so the
// device can join again. It does not touch the storage location; use
// CancelJoin when the club is open.
func AbandonJoin(ctx context.Context, conn *sql.DB) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM sync_club WHERE id = 1`).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && status != StatusJoining) {
		return ErrNotJoining
	}
	if err != nil {
		return err
	}
	for _, t := range joinStateTables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// leave removes this joining device's files from the storage and its local
// club state. Callers hold c.mu.
func (c *Club) leave(ctx context.Context) error {
	// An unreachable storage must not keep the device stuck: the local state
	// is removed anyway. A request file left behind can still be rejected
	// by an admin.
	_ = c.deleteRequest(ctx, c.dev.ID)
	_ = c.deleteRejection(ctx, c.dev.ID)
	if err := AbandonJoin(ctx, c.db); err != nil {
		return err
	}
	c.chain, c.status = nil, ""
	return nil
}

// CancelJoin withdraws this device's pending join request: the request file
// is deleted and the local club state removed, so the device can join again
// with another code. If the request was approved in the meantime, the device
// stays a member and ErrNotJoining is returned.
func (c *Club) CancelJoin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status != StatusJoining {
		return ErrNotJoining
	}
	// Pick up an approval that is already there. A storage that cannot be
	// read does not prevent cancelling.
	if _, err := c.refresh(ctx); errors.Is(err, ErrJoinRejected) {
		return nil
	}
	if c.status != StatusJoining {
		return ErrNotJoining
	}
	return c.leave(ctx)
}
