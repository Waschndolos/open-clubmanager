// Package collab connects the sync identity layer (internal/sync/club) to the
// HTTP API: it holds the unlocked device keys, the open club and its
// transport, refreshes the club in the background and turns the club's
// state into what the UI shows (docs/ux/collaboration-flows.md).
package collab

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/engine"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/transport"
)

// RefreshInterval is how often the club is refreshed in the background
// (spec §18.2: every 30 s while active).
const RefreshInterval = 30 * time.Second

// MaxApproveAttempts is the number of wrong safety codes after which a join
// request is rejected (UX F5).
const MaxApproveAttempts = 3

var (
	// ErrLocked means the device keys exist but are not unlocked yet: the
	// user who set up collaboration has to log in.
	ErrLocked = errors.New("collab: device keys are locked")
	// ErrNoDatabase means no database is open.
	ErrNoDatabase = errors.New("collab: no database")
	// ErrWrongSafetyCode is returned by Approve for a code that does not match.
	ErrWrongSafetyCode = errors.New("collab: safety code does not match")
	// ErrTooManyAttempts is returned by Approve after MaxApproveAttempts
	// wrong codes; the request has been rejected.
	ErrTooManyAttempts = errors.New("collab: too many wrong safety codes, request rejected")
	// ErrOwnDevice is returned when trying to revoke or demote this device.
	ErrOwnDevice = errors.New("collab: this device cannot be changed here")
	// ErrStorageMissing means the club's storage location is not reachable.
	ErrStorageMissing = errors.New("collab: storage location not reachable")
)

// Service is the collaboration state of this installation.
type Service struct {
	// DB returns the currently open database, or nil.
	DB func() *sql.DB
	// DBPath returns the path of the open database; device keys are stored
	// next to it.
	DBPath func() string
	// WorkFactor is the scrypt work factor for the key file (0: default).
	WorkFactor int
	// Now is the wall clock, replaceable in tests.
	Now func() time.Time

	mu       sync.Mutex
	conn     *sql.DB // database the state below belongs to
	dev      *identity.Device
	club     *club.Club
	engine   *engine.Engine
	tr       *transport.LocalFolder
	path     string
	lastSync time.Time
	lastErr  string
	requests []club.JoinRequest
	lostOwn  int
	badKeys  int
	attempts map[string]int
	syncing  bool
	// incomplete is the number of readable scopes without a starting
	// snapshot after the last sync.
	incomplete int
}

// New returns a service. Call Run to start background refreshes.
func New(dbFn func() *sql.DB, pathFn func() string) *Service {
	return &Service{DB: dbFn, DBPath: pathFn, Now: time.Now, attempts: map[string]int{}}
}

// Run refreshes the club every RefreshInterval until ctx ends.
func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(RefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := s.Sync(ctx); err != nil && !errors.Is(err, club.ErrNoClub) && !errors.Is(err, ErrLocked) {
				log.Printf("collab: background refresh: %v", err)
			}
		}
	}
}

// keyStore returns the key store for the open database, protected with the
// given passphrase (the login password of the user who set up collaboration).
func (s *Service) keyStore(passphrase string) *identity.PassphraseFileStore {
	return &identity.PassphraseFileStore{
		Dir:        filepath.Join(filepath.Dir(db.CleanPath(s.DBPath())), "keys"),
		Passphrase: passphrase,
		WorkFactor: s.WorkFactor,
	}
}

// current returns the open database and resets the in-memory state when the
// database was switched since the last call. Callers hold s.mu.
func (s *Service) current() (*sql.DB, error) {
	conn := s.DB()
	if conn == nil {
		return nil, ErrNoDatabase
	}
	if conn != s.conn {
		s.closeLocked()
		s.conn = conn
	}
	return conn, nil
}

func (s *Service) closeLocked() {
	if s.tr != nil {
		_ = s.tr.Close()
	}
	s.dev, s.club, s.engine, s.tr, s.path = nil, nil, nil, nil, ""
	s.requests, s.lostOwn, s.badKeys, s.lastErr, s.incomplete = nil, 0, 0, "", 0
	s.lastSync = time.Time{}
	s.attempts = map[string]int{}
}

// Unlock loads the device keys with the login password and opens the club.
// It is called after every successful login. A wrong password (another local
// user) leaves the keys locked and is not an error.
func (s *Service) Unlock(ctx context.Context, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, err := s.current()
	if err != nil {
		return err
	}
	if s.dev != nil {
		return nil
	}
	dev, err := identity.Load(ctx, conn, s.keyStore(password))
	if errors.Is(err, identity.ErrNoIdentity) || errors.Is(err, identity.ErrWrongPassphrase) {
		return nil
	}
	if err != nil {
		return err
	}
	s.dev = dev
	if err := s.openClubLocked(ctx); err != nil && !errors.Is(err, club.ErrNoClub) {
		s.lastErr = err.Error()
		return nil
	}
	return nil
}

// storage is the JSON in sync_club.transport.
type storage struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

func (s *Service) openClubLocked(ctx context.Context) error {
	var raw string
	err := s.conn.QueryRowContext(ctx, `SELECT transport FROM sync_club WHERE id = 1`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return club.ErrNoClub
	}
	if err != nil {
		return err
	}
	var st storage
	if err := json.Unmarshal([]byte(raw), &st); err != nil || st.Path == "" {
		return fmt.Errorf("collab: no storage location recorded: %w", ErrStorageMissing)
	}
	tr, err := transport.OpenLocalFolder(st.Path)
	if err != nil {
		s.path = st.Path
		return fmt.Errorf("%w: %v", ErrStorageMissing, err)
	}
	c, err := club.Open(ctx, s.conn, s.dev, tr)
	if err != nil {
		_ = tr.Close()
		return err
	}
	s.tr, s.club, s.path = tr, c, st.Path
	s.engine = engine.New(s.conn, c, tr)
	return nil
}

// adopt records a freshly created or joined club. Callers hold s.mu.
func (s *Service) adopt(ctx context.Context, c *club.Club, tr *transport.LocalFolder, path string) error {
	raw, _ := json.Marshal(storage{Kind: "folder", Path: path})
	if _, err := s.conn.ExecContext(ctx, `UPDATE sync_club SET transport = ? WHERE id = 1`, string(raw)); err != nil {
		return err
	}
	if s.tr != nil && s.tr != tr {
		_ = s.tr.Close()
	}
	s.club, s.tr, s.path = c, tr, path
	s.engine = engine.New(s.conn, c, tr)
	return nil
}

// Sync refreshes the club now ("Sync now", spec §23.2) and returns the new
// status.
func (s *Service) Sync(ctx context.Context) (*Status, error) {
	s.mu.Lock()
	if _, err := s.current(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if s.dev == nil {
		s.mu.Unlock()
		return s.Status(ctx)
	}
	if s.club == nil {
		if err := s.openClubLocked(ctx); err != nil {
			if !errors.Is(err, club.ErrNoClub) {
				s.lastErr = err.Error()
			}
			s.mu.Unlock()
			return s.Status(ctx)
		}
	}
	c, eng := s.club, s.engine
	s.syncing = true
	s.mu.Unlock()

	// Refresh the ACL and keys, then exchange data (spec §13: flush the
	// outbox, upload, pull and apply).
	res, err := c.Refresh(ctx)
	var rep *engine.Report
	if err == nil {
		rep, err = eng.Sync(ctx)
	}
	if err == nil && rep.UploadErr != nil {
		err = rep.UploadErr
	}
	var reqs []club.JoinRequest
	if err == nil && c.Head() != nil && c.Head().IsAdmin(c.Device().ID) {
		reqs, err = c.JoinRequests(ctx)
	}

	s.mu.Lock()
	s.syncing = false
	if s.club == c {
		if err != nil {
			s.lastErr = err.Error()
		} else {
			s.lastErr = ""
			s.lastSync = s.Now()
			s.requests = reqs
			s.lostOwn += len(res.LostOwn)
			s.badKeys = len(res.BadKeys)
			s.incomplete = len(rep.Incomplete)
		}
	}
	s.mu.Unlock()
	return s.Status(ctx)
}

// Area permissions as the UI uses them.
const (
	PermNone  = "none"
	PermRead  = "read"
	PermWrite = "write"
)

// Status is what the UI shows about collaboration (UX F7).
type Status struct {
	// Mode is "local", "joining", "member" or "revoked".
	Mode            string
	Locked          bool
	ClubName        string
	ClubID          string
	StoragePath     string
	StorageError    bool
	DeviceID        string
	Person          string
	DeviceName      string
	Role            string
	IsAdmin         bool
	Permissions     map[db.Scope]string
	LastSync        *time.Time
	LastError       string
	Syncing         bool
	PendingChanges  int
	PendingRequests int
	Conflicts       int
	LostChanges     int
	BadKeys         int
	// Sync core (spec §13): streams of other devices that wait or are
	// quarantined, received changes waiting for a referenced row, readable
	// areas without a starting snapshot, devices with a clock warning.
	WaitingStreams     int
	QuarantinedStreams int
	WaitingChanges     int
	IncompleteAreas    int
	ClockWarnings      int
}

// Status returns the current status. In local mode every area is writable.
func (s *Service) Status(ctx context.Context) (*Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, err := s.current()
	if err != nil {
		return nil, err
	}
	st := &Status{Mode: "local", Permissions: map[db.Scope]string{}}
	for _, sc := range db.Scopes {
		st.Permissions[sc] = PermWrite
	}
	var status, name, clubID string
	err = conn.QueryRowContext(ctx, `SELECT status, name, club_id FROM sync_club WHERE id = 1`).Scan(&status, &name, &clubID)
	if errors.Is(err, sql.ErrNoRows) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	st.Mode, st.ClubName, st.ClubID = status, name, clubID
	_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_outbox`).Scan(&st.PendingChanges)
	if conflicts, err := store.New(conn).ListConflicts(ctx); err == nil {
		st.Conflicts = len(conflicts)
	}
	if sc, waiting, warn, err := engine.Counts(ctx, conn); err == nil {
		st.WaitingStreams, st.QuarantinedStreams = sc.Waiting, sc.Quarantined
		st.WaitingChanges, st.ClockWarnings = waiting, len(warn)
	}
	if s.dev == nil {
		st.Locked = true
		for _, sc := range db.Scopes {
			st.Permissions[sc] = PermNone
		}
		return st, nil
	}
	st.DeviceID = s.dev.ID
	st.StoragePath = s.path
	st.StorageError = s.club == nil
	st.LastError, st.Syncing = s.lastErr, s.syncing
	st.LostChanges, st.BadKeys = s.lostOwn, s.badKeys
	st.IncompleteAreas = s.incomplete
	if !s.lastSync.IsZero() {
		t := s.lastSync
		st.LastSync = &t
	}
	for _, r := range s.requests {
		if r.Err == nil {
			st.PendingRequests++
		}
	}
	var head *acl.Version
	if s.club != nil {
		head = s.club.Head()
		st.Mode = s.club.Status()
	}
	for _, sc := range db.Scopes {
		st.Permissions[sc] = PermNone
	}
	if head == nil {
		return st, nil
	}
	if d := head.Device(s.dev.ID); d != nil {
		st.Person, st.DeviceName, st.Role = d.Person, d.DeviceName, d.Role
	}
	st.IsAdmin = head.IsAdmin(s.dev.ID)
	for _, sc := range db.Scopes {
		p := head.Perm(s.dev.ID, sc)
		switch {
		case p.CanWrite():
			st.Permissions[sc] = PermWrite
		case p.CanRead():
			st.Permissions[sc] = PermRead
		}
	}
	return st, nil
}

// CanWrite reports whether this device may change data of scope sc. In local
// mode everything is writable; in a club the ACL decides. It is used to
// reject local writes that other devices would ignore anyway.
func (s *Service) CanWrite(ctx context.Context, sc db.Scope) bool {
	st, err := s.Status(ctx)
	if err != nil {
		return true
	}
	return st.Permissions[sc] == PermWrite
}

// Club returns the open club, or an error explaining why there is none.
func (s *Service) openClub() (*club.Club, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.current(); err != nil {
		return nil, err
	}
	if s.dev == nil {
		var n int
		if s.conn.QueryRow(`SELECT COUNT(*) FROM sync_club`).Scan(&n); n == 0 {
			return nil, club.ErrNoClub
		}
		return nil, ErrLocked
	}
	if s.club == nil {
		if err := s.openClubLocked(context.Background()); err != nil {
			return nil, err
		}
	}
	return s.club, nil
}
