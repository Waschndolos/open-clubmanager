package collab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

// ErrClubExists is returned by Create when the target folder already holds a
// club; the user probably wants to join it instead (UX F2 step 1).
var ErrClubExists = errors.New("collab: the folder already contains a club")

// ClubFolderName is the subfolder Create makes inside the chosen folder.
func ClubFolderName(clubName string) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '-'
		}
		return r
	}, strings.TrimSpace(clubName))
	name = strings.Trim(name, ". ")
	if name == "" {
		name = "Verein"
	}
	return "Open ClubManager – " + name
}

// device returns this installation's identity, creating it with the given
// password on first use. Callers hold s.mu.
func (s *Service) device(ctx context.Context, password string) (*identity.Device, error) {
	if s.dev != nil {
		return s.dev, nil
	}
	dev, err := identity.Ensure(ctx, s.conn, s.keyStore(password))
	if err != nil {
		return nil, err
	}
	s.dev = dev
	return dev, nil
}

// CreateResult is returned by Create.
type CreateResult struct {
	Path        string
	RecoveryKit string
}

// Create sets up a new club in a subfolder of folder (UX F2). The recovery
// kit is returned once and not kept.
func (s *Service) Create(ctx context.Context, folder, clubName, person, deviceName, password string) (*CreateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(); err != nil {
		return nil, err
	}
	info, err := CheckFolder(folder)
	if err != nil {
		return nil, err
	}
	if !info.Writable {
		return nil, fmt.Errorf("%w: %s", ErrNotWritable, folder)
	}
	if info.Club != nil {
		return nil, ErrClubExists
	}
	dir := filepath.Join(info.Path, ClubFolderName(clubName))
	if sub, err := CheckFolder(dir); err == nil && sub.Club != nil {
		return nil, ErrClubExists
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	dev, err := s.device(ctx, password)
	if err != nil {
		return nil, err
	}
	tr, err := transport.OpenLocalFolder(dir)
	if err != nil {
		return nil, err
	}
	c, kit, err := club.Create(ctx, s.conn, dev, tr, strings.TrimSpace(clubName), strings.TrimSpace(person), strings.TrimSpace(deviceName))
	if err != nil {
		_ = tr.Close()
		return nil, err
	}
	text := kit.Text()
	kit.Wipe()
	if err := s.adopt(ctx, c, tr, dir); err != nil {
		return nil, err
	}
	s.lastSync = s.Now()
	return &CreateResult{Path: dir, RecoveryKit: text}, nil
}

// Join sends a join request to the club in folder (UX F4) and returns the
// safety code to read out to the admin. If replaceLocalData is set, synced
// tables that still hold data are backed up next to the database and cleared
// first (spec §17.1); otherwise such data makes Join fail with
// club.ErrLocalData.
func (s *Service) Join(ctx context.Context, folder, code, person, deviceName, password string, replaceLocalData bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(); err != nil {
		return "", err
	}
	if _, _, err := club.ParseInviteCode(code); err != nil {
		return "", err
	}
	dir, err := clubDir(folder)
	if err != nil {
		return "", err
	}
	if replaceLocalData {
		if err := s.backupAndClear(ctx); err != nil {
			return "", err
		}
	}
	dev, err := s.device(ctx, password)
	if err != nil {
		return "", err
	}
	tr, err := transport.OpenLocalFolder(dir)
	if err != nil {
		return "", err
	}
	c, safety, err := club.Join(ctx, s.conn, dev, tr, code, strings.TrimSpace(person), strings.TrimSpace(deviceName))
	if err != nil {
		_ = tr.Close()
		return "", err
	}
	if err := s.adopt(ctx, c, tr, dir); err != nil {
		return "", err
	}
	return safety, nil
}

// SafetyCode returns the safety code of this device, shown while a join
// request waits for approval.
func (s *Service) SafetyCode() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dev == nil {
		return "", ErrLocked
	}
	return identity.SafetyCode(s.dev.SignPub(), s.dev.AgeRecipient()), nil
}

// Recover uses the recovery kit (UX F12): on a device without a club it
// joins the club in folder as admin; on a member device it makes this
// device an admin device again.
func (s *Service) Recover(ctx context.Context, folder, kitText, person, deviceName, password string, replaceLocalData bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(); err != nil {
		return err
	}
	if s.club != nil {
		_, err := s.club.Recover(ctx, kitText, strings.TrimSpace(person), strings.TrimSpace(deviceName))
		return err
	}
	dir, err := clubDir(folder)
	if err != nil {
		return err
	}
	if replaceLocalData {
		if err := s.backupAndClear(ctx); err != nil {
			return err
		}
	}
	dev, err := s.device(ctx, password)
	if err != nil {
		return err
	}
	tr, err := transport.OpenLocalFolder(dir)
	if err != nil {
		return err
	}
	c, _, err := club.RecoverNew(ctx, s.conn, dev, tr, kitText, strings.TrimSpace(person), strings.TrimSpace(deviceName))
	if err != nil {
		_ = tr.Close()
		return err
	}
	if _, err := c.Refresh(ctx); err != nil {
		_ = tr.Close()
		return err
	}
	return s.adopt(ctx, c, tr, dir)
}

// clubDir returns folder if it holds a club, or its only subfolder that does.
func clubDir(folder string) (string, error) {
	info, err := CheckFolder(folder)
	if err != nil {
		return "", err
	}
	if info.Club != nil {
		return info.Path, nil
	}
	if len(info.ClubsInside) == 1 {
		return info.ClubsInside[0].Path, nil
	}
	return "", fmt.Errorf("%w: %s", ErrNoClubInFolder, folder)
}

// backupAndClear writes a copy of the database next to it and clears the
// synced tables (spec §17.1 "Joining with existing local data").
func (s *Service) backupAndClear(ctx context.Context) error {
	empty, err := s.syncedTablesEmpty(ctx)
	if err != nil || empty {
		return err
	}
	path := db.CleanPath(s.DBPath())
	backup := strings.TrimSuffix(path, filepath.Ext(path)) + "-vor-beitritt-" + s.Now().Format("20060102-150405") + ".db"
	if _, err := s.conn.ExecContext(ctx, `VACUUM INTO ?`, backup); err != nil {
		return fmt.Errorf("collab: backup before join: %w", err)
	}
	return club.ClearSyncedData(ctx, s.conn)
}

func (s *Service) syncedTablesEmpty(ctx context.Context) (bool, error) {
	for _, t := range db.SyncedTables {
		var n int
		if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+t.Name).Scan(&n); err != nil {
			return false, err
		}
		if n > 0 {
			return false, nil
		}
	}
	return true, nil
}

// HasLocalData reports whether synced tables hold data, so joining would
// replace it.
func (s *Service) HasLocalData(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(); err != nil {
		return false, err
	}
	empty, err := s.syncedTablesEmpty(ctx)
	return !empty, err
}

// Invites lists open and used invites of this admin device.
func (s *Service) Invites(ctx context.Context) ([]club.Invite, error) {
	c, err := s.openClub()
	if err != nil {
		return nil, err
	}
	return c.Invites(ctx)
}

// CreateInvite creates an invite and returns it with its code (UX F3).
func (s *Service) CreateInvite(ctx context.Context, role, person string) (*club.Invite, error) {
	c, err := s.openClub()
	if err != nil {
		return nil, err
	}
	return c.CreateInvite(ctx, role, strings.TrimSpace(person), 0)
}

// WithdrawInvite deletes an unused invite.
func (s *Service) WithdrawInvite(ctx context.Context, inviteID string) error {
	c, err := s.openClub()
	if err != nil {
		return err
	}
	return c.WithdrawInvite(ctx, inviteID)
}

// Requests returns the pending join requests found by the last refresh,
// without their safety codes: the admin must type the code the new member
// reads out (UX F5).
func (s *Service) Requests() []club.JoinRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]club.JoinRequest, 0, len(s.requests))
	for _, r := range s.requests {
		r.SafetyCode = ""
		out = append(out, r)
	}
	return out
}

func normalizeCode(code string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, code)
}

// Approve admits a requesting device if safetyCode matches. After
// MaxApproveAttempts wrong codes the request is rejected.
func (s *Service) Approve(ctx context.Context, deviceID, safetyCode string) error {
	c, err := s.openClub()
	if err != nil {
		return err
	}
	reqs, err := c.JoinRequests(ctx)
	if err != nil {
		return err
	}
	var req *club.JoinRequest
	for i := range reqs {
		if reqs[i].DeviceID == deviceID {
			req = &reqs[i]
		}
	}
	if req == nil {
		return club.ErrNoRequest
	}
	if req.Err != nil {
		return req.Err
	}
	if normalizeCode(safetyCode) != req.SafetyCode {
		s.mu.Lock()
		s.attempts[deviceID]++
		n := s.attempts[deviceID]
		s.mu.Unlock()
		if n >= MaxApproveAttempts {
			if err := c.RejectRequest(ctx, deviceID); err != nil {
				return err
			}
			s.dropRequest(deviceID)
			return ErrTooManyAttempts
		}
		return fmt.Errorf("%w (%d of %d attempts)", ErrWrongSafetyCode, n, MaxApproveAttempts)
	}
	if _, err := c.Approve(ctx, deviceID); err != nil {
		return err
	}
	s.dropRequest(deviceID)
	return nil
}

// Reject deletes a join request.
func (s *Service) Reject(ctx context.Context, deviceID string) error {
	c, err := s.openClub()
	if err != nil {
		return err
	}
	if err := c.RejectRequest(ctx, deviceID); err != nil {
		return err
	}
	s.dropRequest(deviceID)
	return nil
}

func (s *Service) dropRequest(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, deviceID)
	out := s.requests[:0]
	for _, r := range s.requests {
		if r.DeviceID != deviceID {
			out = append(out, r)
		}
	}
	s.requests = out
}

// DeviceInfo is one device of the ACL (UX F6).
type DeviceInfo struct {
	DeviceID   string
	Person     string
	DeviceName string
	Role       string
	Active     bool
	Me         bool
	AddedIn    uint64
}

// Devices lists the devices of the current ACL, active ones first, grouped
// by person.
func (s *Service) Devices() ([]DeviceInfo, error) {
	c, err := s.openClub()
	if err != nil {
		return nil, err
	}
	head := c.Head()
	if head == nil {
		return []DeviceInfo{}, nil
	}
	out := make([]DeviceInfo, 0, len(head.Devices))
	for _, d := range head.Devices {
		out = append(out, DeviceInfo{
			DeviceID: d.DeviceID, Person: d.Person, DeviceName: d.DeviceName, Role: d.Role,
			Active: d.Active(), Me: d.DeviceID == c.Device().ID, AddedIn: d.AddedIn,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Active != out[j].Active {
			return out[i].Active
		}
		if out[i].Person != out[j].Person {
			return out[i].Person < out[j].Person
		}
		return out[i].DeviceName < out[j].DeviceName
	})
	return out, nil
}

// Roles returns the roles of the current ACL, or the default roles before
// the device is a member.
func (s *Service) Roles() map[string]acl.Role {
	if c, err := s.openClub(); err == nil && c.Head() != nil {
		return c.Head().Roles
	}
	return acl.DefaultRoles()
}

// ChangeRole gives another device a new role (UX F6).
func (s *Service) ChangeRole(ctx context.Context, deviceID, role string) error {
	c, err := s.openClub()
	if err != nil {
		return err
	}
	if deviceID == c.Device().ID {
		return ErrOwnDevice
	}
	_, err = c.ChangeRole(ctx, deviceID, role)
	return err
}

// Revoke removes another device (UX F6). The cutoff is computed by the club
// from the streams this device accepted.
func (s *Service) Revoke(ctx context.Context, deviceID string) error {
	c, err := s.openClub()
	if err != nil {
		return err
	}
	if deviceID == c.Device().ID {
		return ErrOwnDevice
	}
	_, err = c.Revoke(ctx, deviceID, nil)
	return err
}
