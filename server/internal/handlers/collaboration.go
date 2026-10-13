package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/collab"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/acl"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/club"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/identity"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// Collab returns the collaboration service, e.g. to start its background
// refresh.
func (a *API) Collab() *collab.Service { return a.collab }

func newCollab(a *API) *collab.Service {
	return collab.New(
		func() *sql.DB {
			a.dbMu.RLock()
			defer a.dbMu.RUnlock()
			return a.conn
		},
		func() string { return a.cfg.DatabasePath },
	)
}

// collabError maps errors of the collaboration layer to responses. code is a
// stable identifier the frontend uses to pick a message.
func collabError(w http.ResponseWriter, err error) {
	type mapping struct {
		target error
		status int
		code   string
	}
	for _, m := range []mapping{
		{collab.ErrNoDatabase, http.StatusServiceUnavailable, "NO_DATABASE"},
		{collab.ErrLocked, http.StatusForbidden, "LOCKED"},
		{club.ErrNoClub, http.StatusConflict, "NO_CLUB"},
		{club.ErrAlreadyInClub, http.StatusConflict, "ALREADY_IN_CLUB"},
		{club.ErrLocalData, http.StatusConflict, "LOCAL_DATA"},
		{collab.ErrClubExists, http.StatusConflict, "CLUB_EXISTS"},
		{collab.ErrFolderMissing, http.StatusNotFound, "FOLDER_MISSING"},
		{collab.ErrNotWritable, http.StatusBadRequest, "FOLDER_NOT_WRITABLE"},
		{collab.ErrNoClubInFolder, http.StatusBadRequest, "NO_CLUB_IN_FOLDER"},
		{collab.ErrStorageMissing, http.StatusConflict, "STORAGE_MISSING"},
		{club.ErrBadInviteCode, http.StatusBadRequest, "BAD_INVITE_CODE"},
		{identity.ErrBadRecoveryKit, http.StatusBadRequest, "BAD_RECOVERY_KIT"},
		{club.ErrKitMismatch, http.StatusBadRequest, "KIT_MISMATCH"},
		{club.ErrNotAdmin, http.StatusForbidden, "NOT_ADMIN"},
		{club.ErrUnknownRole, http.StatusBadRequest, "UNKNOWN_ROLE"},
		{club.ErrNoInvite, http.StatusNotFound, "NO_INVITE"},
		{club.ErrNoRequest, http.StatusNotFound, "NO_REQUEST"},
		{club.ErrInviteUnknown, http.StatusConflict, "INVITE_UNKNOWN"},
		{club.ErrInviteExpired, http.StatusConflict, "INVITE_EXPIRED"},
		{club.ErrInviteUsed, http.StatusConflict, "INVITE_USED"},
		{club.ErrContested, http.StatusConflict, "INVITE_CONTESTED"},
		{club.ErrBadProof, http.StatusConflict, "BAD_PROOF"},
		{club.ErrBadRequest, http.StatusConflict, "BAD_REQUEST"},
		{collab.ErrWrongSafetyCode, http.StatusUnprocessableEntity, "WRONG_SAFETY_CODE"},
		{collab.ErrTooManyAttempts, http.StatusUnprocessableEntity, "TOO_MANY_ATTEMPTS"},
		{collab.ErrOwnDevice, http.StatusConflict, "OWN_DEVICE"},
		{club.ErrLastAdmin, http.StatusConflict, "LAST_ADMIN"},
		{club.ErrUnknownDevice, http.StatusNotFound, "UNKNOWN_DEVICE"},
		{club.ErrDeviceRevoked, http.StatusConflict, "DEVICE_REVOKED"},
		{club.ErrNotMember, http.StatusConflict, "NOT_MEMBER"},
	} {
		if errors.Is(err, m.target) {
			code := m.code
			writeJSON(w, m.status, openapi.ErrorResponse{Code: &code, Error: err.Error()})
			return
		}
	}
	log.Printf("collaboration: %v", err)
	code := "INTERNAL_ERROR"
	writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Code: &code, Error: "collaboration request failed"})
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return false
	}
	return true
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// checkPassword verifies the password of the logged-in user. Creating or
// joining a club encrypts the device keys with it, so the keys unlock on
// every later login.
func (a *API) checkPassword(r *http.Request, password string) bool {
	email := currentUserID(r)
	if email == a.cfg.AdminEmail && a.cfg.AdminPassword != "" {
		return bcrypt.CompareHashAndPassword([]byte(a.cfg.AdminPassword), []byte(password)) == nil
	}
	st := a.store()
	if st == nil {
		return false
	}
	u, err := st.GetUserByEmail(r.Context(), email)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func (a *API) requirePassword(w http.ResponseWriter, r *http.Request, password string) bool {
	if a.checkPassword(r, password) {
		return true
	}
	code := "WRONG_PASSWORD"
	writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "password is not correct"})
	return false
}

func permission(p string) openapi.AreaPermission { return openapi.AreaPermission(p) }

func areaPermissions(get func(dbpkg.Scope) string) openapi.AreaPermissions {
	return openapi.AreaPermissions{
		MembersCore:    permission(get(dbpkg.ScopeMembersCore)),
		MembersContact: permission(get(dbpkg.ScopeMembersContact)),
		MembersBank:    permission(get(dbpkg.ScopeMembersBank)),
		Finance:        permission(get(dbpkg.ScopeFinance)),
		Inventory:      permission(get(dbpkg.ScopeInventory)),
		Admin:          permission(get(dbpkg.ScopeAdmin)),
	}
}

func toStatus(st *collab.Status) openapi.CollaborationStatus {
	out := openapi.CollaborationStatus{
		Mode:            openapi.CollaborationStatusMode(st.Mode),
		Locked:          st.Locked,
		ClubName:        strPtr(st.ClubName),
		ClubId:          strPtr(st.ClubID),
		StoragePath:     strPtr(st.StoragePath),
		DeviceId:        strPtr(st.DeviceID),
		Person:          strPtr(st.Person),
		DeviceName:      strPtr(st.DeviceName),
		Role:            strPtr(st.Role),
		IsAdmin:         st.IsAdmin,
		Permissions:     areaPermissions(func(s dbpkg.Scope) string { return st.Permissions[s] }),
		LastSync:        st.LastSync,
		LastError:       strPtr(st.LastError),
		Syncing:         st.Syncing,
		PendingChanges:  st.PendingChanges,
		PendingRequests: st.PendingRequests,
		Conflicts:       st.Conflicts,
		LostChanges:     st.LostChanges,
		BadKeys:         st.BadKeys,

		WaitingStreams:     st.WaitingStreams,
		QuarantinedStreams: st.QuarantinedStreams,
		WaitingChanges:     st.WaitingChanges,
		IncompleteAreas:    st.IncompleteAreas,
		ClockWarnings:      st.ClockWarnings,
	}
	if st.StorageError {
		t := true
		out.StorageError = &t
	}
	return out
}

func (a *API) GetCollaborationStatus(w http.ResponseWriter, r *http.Request) {
	st, err := a.collab.Status(r.Context())
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatus(st))
}

func (a *API) SyncCollaboration(w http.ResponseWriter, r *http.Request) {
	st, err := a.collab.Sync(r.Context())
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatus(st))
}

func toFolder(f *collab.FolderInfo) openapi.FolderInfo {
	out := openapi.FolderInfo{
		Path: f.Path, Writable: f.Writable, Kind: openapi.FolderInfoKind(f.Kind),
		Provider: strPtr(f.Provider), ClubsInside: []openapi.FolderClub{},
	}
	if f.Club != nil {
		out.Club = &openapi.FolderClub{Path: f.Club.Path, Name: f.Club.Name}
	}
	for _, c := range f.ClubsInside {
		out.ClubsInside = append(out.ClubsInside, openapi.FolderClub{Path: c.Path, Name: c.Name})
	}
	return out
}

func (a *API) ListCollaborationFolders(w http.ResponseWriter, _ *http.Request) {
	out := []openapi.FolderInfo{}
	for _, f := range collab.DetectFolders() {
		out = append(out, toFolder(&f))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CheckCollaborationFolder(w http.ResponseWriter, r *http.Request) {
	var req openapi.CheckFolderRequest
	if !decodeBody(w, r, &req) {
		return
	}
	info, err := collab.CheckFolder(req.Path)
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFolder(info))
}

func (a *API) CreateCollaborationClub(w http.ResponseWriter, r *http.Request) {
	var req openapi.CreateClubRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ClubName) == "" || strings.TrimSpace(req.Person) == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "club name and person are required"})
		return
	}
	if !a.requirePassword(w, r, req.Password) {
		return
	}
	res, err := a.collab.Create(r.Context(), req.Folder, req.ClubName, req.Person, req.DeviceName, req.Password)
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, openapi.CreateClubResponse{Path: res.Path, RecoveryKit: res.RecoveryKit})
}

func (a *API) JoinCollaborationClub(w http.ResponseWriter, r *http.Request) {
	var req openapi.JoinClubRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Person) == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "person is required"})
		return
	}
	if !a.requirePassword(w, r, req.Password) {
		return
	}
	replace := req.ReplaceLocalData != nil && *req.ReplaceLocalData
	safety, err := a.collab.Join(r.Context(), req.Folder, req.Code, req.Person, req.DeviceName, req.Password, replace)
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, openapi.SafetyCodeResponse{SafetyCode: safety})
}

func (a *API) GetCollaborationSafetyCode(w http.ResponseWriter, _ *http.Request) {
	code, err := a.collab.SafetyCode()
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, openapi.SafetyCodeResponse{SafetyCode: code})
}

func (a *API) RecoverCollaborationClub(w http.ResponseWriter, r *http.Request) {
	var req openapi.RecoverRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if !a.requirePassword(w, r, req.Password) {
		return
	}
	folder := ""
	if req.Folder != nil {
		folder = *req.Folder
	}
	replace := req.ReplaceLocalData != nil && *req.ReplaceLocalData
	if err := a.collab.Recover(r.Context(), folder, req.RecoveryKit, req.Person, req.DeviceName, req.Password, replace); err != nil {
		collabError(w, err)
		return
	}
	a.SyncCollaboration(w, r)
}

func (a *API) GetCollaborationLocalData(w http.ResponseWriter, r *http.Request) {
	has, err := a.collab.HasLocalData(r.Context())
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, openapi.LocalDataResponse{HasLocalData: has})
}

func toInvite(inv *club.Invite) openapi.ClubInvite {
	return openapi.ClubInvite{
		Id: wire.EncodeB64(inv.ID), Person: inv.Person, Role: inv.Role,
		CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt, UsedAt: inv.UsedAt, Code: strPtr(inv.Code),
	}
}

func (a *API) ListCollaborationInvites(w http.ResponseWriter, r *http.Request) {
	invs, err := a.collab.Invites(r.Context())
	if err != nil {
		collabError(w, err)
		return
	}
	out := []openapi.ClubInvite{}
	for i := range invs {
		out = append(out, toInvite(&invs[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateCollaborationInvite(w http.ResponseWriter, r *http.Request) {
	var req openapi.CreateInviteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Person) == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "person is required"})
		return
	}
	inv, err := a.collab.CreateInvite(r.Context(), req.Role, req.Person)
	if err != nil {
		collabError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toInvite(inv))
}

func (a *API) WithdrawCollaborationInvite(w http.ResponseWriter, r *http.Request, inviteId string) {
	if err := a.collab.WithdrawInvite(r.Context(), inviteId); err != nil {
		collabError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requestError turns the reason a request cannot be approved into a stable
// code for the UI.
func requestError(err error) *string {
	if err == nil {
		return nil
	}
	for target, code := range map[error]string{
		club.ErrInviteUnknown: "INVITE_UNKNOWN", club.ErrInviteExpired: "INVITE_EXPIRED",
		club.ErrInviteUsed: "INVITE_USED", club.ErrContested: "INVITE_CONTESTED",
		club.ErrBadProof: "BAD_PROOF", club.ErrBadRequest: "BAD_REQUEST", club.ErrAlreadyListed: "ALREADY_LISTED",
	} {
		if errors.Is(err, target) {
			return &code
		}
	}
	code := "INVALID"
	return &code
}

func (a *API) ListCollaborationRequests(w http.ResponseWriter, _ *http.Request) {
	out := []openapi.ClubJoinRequest{}
	for _, req := range a.collab.Requests() {
		out = append(out, openapi.ClubJoinRequest{
			DeviceId: req.DeviceID, Person: req.Person, DeviceName: strPtr(req.DeviceName),
			Role: strPtr(req.Role), Error: requestError(req.Err),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) ApproveCollaborationRequest(w http.ResponseWriter, r *http.Request, deviceId string) {
	var req openapi.ApproveRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if err := a.collab.Approve(r.Context(), deviceId, req.SafetyCode); err != nil {
		collabError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RejectCollaborationRequest(w http.ResponseWriter, r *http.Request, deviceId string) {
	if err := a.collab.Reject(r.Context(), deviceId); err != nil {
		collabError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListCollaborationDevices(w http.ResponseWriter, _ *http.Request) {
	devs, err := a.collab.Devices()
	if err != nil {
		collabError(w, err)
		return
	}
	out := []openapi.ClubDevice{}
	for _, d := range devs {
		out = append(out, openapi.ClubDevice{
			DeviceId: d.DeviceID, Person: d.Person, DeviceName: strPtr(d.DeviceName),
			Role: d.Role, Active: d.Active, Me: d.Me,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) ChangeCollaborationDeviceRole(w http.ResponseWriter, r *http.Request, deviceId string) {
	var req openapi.ChangeDeviceRoleRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if err := a.collab.ChangeRole(r.Context(), deviceId, req.Role); err != nil {
		collabError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RevokeCollaborationDevice(w http.ResponseWriter, r *http.Request, deviceId string) {
	if err := a.collab.Revoke(r.Context(), deviceId); err != nil {
		collabError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// roleOrder is the display order of the default roles (spec §4.2).
var roleOrder = []string{acl.RoleBoard, acl.RoleTreasurer, acl.RoleSecretary, acl.RoleCoach, acl.RoleEquipmentManager}

func (a *API) ListCollaborationRoles(w http.ResponseWriter, _ *http.Request) {
	roles := a.collab.Roles()
	ids := make([]string, 0, len(roles))
	for id := range roles {
		ids = append(ids, id)
	}
	rank := func(id string) int {
		for i, r := range roleOrder {
			if r == id {
				return i
			}
		}
		return len(roleOrder)
	}
	sort.Slice(ids, func(i, j int) bool {
		if rank(ids[i]) != rank(ids[j]) {
			return rank(ids[i]) < rank(ids[j])
		}
		return ids[i] < ids[j]
	})
	out := make([]openapi.ClubRole, 0, len(ids))
	for _, id := range ids {
		role := roles[id]
		out = append(out, openapi.ClubRole{Id: id, Permissions: areaPermissions(func(s dbpkg.Scope) string {
			switch p := role[s]; {
			case p.CanWrite():
				return collab.PermWrite
			case p.CanRead():
				return collab.PermRead
			}
			return collab.PermNone
		})})
	}
	writeJSON(w, http.StatusOK, out)
}

// pkID returns the id of a row from its encoded primary key.
func pkID(encoded string) string {
	pk, err := op.DecodePK(encoded)
	if err != nil {
		return encoded
	}
	for _, k := range []string{"id", "member_id"} {
		if v, ok := pk[k]; ok {
			return fmt.Sprint(v)
		}
	}
	return encoded
}

func (a *API) ListCollaborationConflicts(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	conflicts, err := st.ListConflicts(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list conflicts")
		return
	}
	out := []openapi.SyncConflict{}
	for _, c := range conflicts {
		out = append(out, openapi.SyncConflict{
			Table: c.Table, RowId: pkID(c.PK), Column: c.Column, Value: fmt.Sprint(c.Value),
			KeptRowId: pkID(c.KeptPK), DetectedAt: c.DetectedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// writeScopes maps API path prefixes to the data scopes a write there
// changes. Member writes carry core, contact and bank data in one payload;
// they are allowed if the device may write any of them, and the UI disables
// the fields of the other areas.
var writeScopes = []struct {
	prefix string
	scopes []dbpkg.Scope
}{
	{"/api/v2/members", []dbpkg.Scope{dbpkg.ScopeMembersCore, dbpkg.ScopeMembersContact, dbpkg.ScopeMembersBank}},
	{"/api/v2/roles", []dbpkg.Scope{dbpkg.ScopeMembersCore}},
	{"/api/v2/groups", []dbpkg.Scope{dbpkg.ScopeMembersCore}},
	{"/api/v2/sections", []dbpkg.Scope{dbpkg.ScopeMembersCore}},
	{"/api/v2/finance", []dbpkg.Scope{dbpkg.ScopeFinance}},
	{"/api/v2/inventory", []dbpkg.Scope{dbpkg.ScopeInventory}},
	{"/api/v2/events", []dbpkg.Scope{dbpkg.ScopeMembersCore}},
}

// ScopeGuard rejects writes to data this device may not change in its club
// (spec §4.2). Other devices would ignore such changes anyway. In local mode
// everything is allowed.
func (a *API) ScopeGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		for _, ws := range writeScopes {
			if !strings.HasPrefix(r.URL.Path, ws.prefix) {
				continue
			}
			allowed := false
			for _, s := range ws.scopes {
				if a.collab.CanWrite(r.Context(), s) {
					allowed = true
				}
			}
			if !allowed {
				code := "READ_ONLY_AREA"
				writeJSON(w, http.StatusForbidden, openapi.ErrorResponse{Code: &code, Error: "your role may not change this data"})
				return
			}
			break
		}
		next.ServeHTTP(w, r)
	})
}
