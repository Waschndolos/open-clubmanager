package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// adminUserID is the fixed user-id used for the single-admin config-based
// deployment. The UserPreference table expects an integer user id.
const adminUserID = 1

// ── User Preferences ─────────────────────────────────────────────────────────

// ListPreferences returns all preferences stored for the current (admin) user.
func (a *API) ListPreferences(w http.ResponseWriter, _ *http.Request) {
	if !a.requireDB(w) {
		return
	}
	rows, err := a.getDB().Query(
		`SELECT "id","userId","key","value","updatedAt" FROM "UserPreference" WHERE "userId"=? ORDER BY "key"`,
		adminUserID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to list preferences"})
		return
	}
	defer rows.Close()

	result := []openapi.UserPreferenceRecord{}
	for rows.Next() {
		var p openapi.UserPreferenceRecord
		var updatedAtStr string
		if err := rows.Scan(&p.Id, &p.UserId, &p.Key, &p.Value, &updatedAtStr); err == nil {
			if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
				p.UpdatedAt = t
			}
			result = append(result, p)
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// UpsertPreference creates or updates a preference for the current user.
func (a *API) UpsertPreference(w http.ResponseWriter, r *http.Request) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.UpsertUserPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	valueStr, err := marshalPreferenceValue(req.Value)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "value must be JSON-serializable"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.getDB().Exec(
		`INSERT INTO "UserPreference" ("userId","key","value","updatedAt") VALUES (?,?,?,?)
		 ON CONFLICT("userId","key") DO UPDATE SET "value"=excluded."value","updatedAt"=excluded."updatedAt"`,
		adminUserID, req.Key, valueStr, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to upsert preference"})
		return
	}
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// GetPreferenceByKey returns a single user preference by key.
func (a *API) GetPreferenceByKey(w http.ResponseWriter, _ *http.Request, identifier string) {
	if !a.requireDB(w) {
		return
	}
	var value, updatedAtStr string
	err := a.getDB().QueryRow(
		`SELECT "value","updatedAt" FROM "UserPreference" WHERE "userId"=? AND "key"=?`,
		adminUserID, identifier,
	).Scan(&value, &updatedAtStr)
	if err != nil {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "preference not found"})
		return
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		parsed = value // fall back to raw string
	}

	var updatedAt time.Time
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		updatedAt = t
	}

	writeJSON(w, http.StatusOK, openapi.UserPreferenceValueResponse{
		Key:       identifier,
		Value:     parsed,
		UpdatedAt: updatedAt,
	})
}

// UpdatePreferenceById updates a user preference by key (identifier).
func (a *API) UpdatePreferenceById(w http.ResponseWriter, r *http.Request, identifier string) {
	if !a.requireDB(w) {
		return
	}
	var req openapi.UpdateUserPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := a.getDB().Exec(
		`INSERT INTO "UserPreference" ("userId","key","value","updatedAt") VALUES (?,?,?,?)
		 ON CONFLICT("userId","key") DO UPDATE SET "value"=excluded."value","updatedAt"=excluded."updatedAt"`,
		adminUserID, identifier, req.Value, now,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to update preference"})
		return
	}
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// ── App Preferences ───────────────────────────────────────────────────────────
// App preferences are stored in the server config to avoid a DB dependency
// during early setup and for application-wide settings.

// GetAppPreferences returns all app-level preferences.
func (a *API) GetAppPreferences(w http.ResponseWriter, _ *http.Request) {
	prefs := a.cfg.AppPreferences
	if prefs == nil {
		prefs = map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, prefs)
}

// GetAppPreferenceByKey returns a single app preference.
func (a *API) GetAppPreferenceByKey(w http.ResponseWriter, _ *http.Request, key string) {
	if a.cfg.AppPreferences == nil {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "preference not found"})
		return
	}
	value, ok := a.cfg.AppPreferences[key]
	if !ok {
		writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "preference not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{key: value})
}

// UpdateAppPreferenceByKey sets a single app preference and persists the config.
func (a *API) UpdateAppPreferenceByKey(w http.ResponseWriter, r *http.Request, key string) {
	var req openapi.UpdateAppPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if a.cfg.AppPreferences == nil {
		a.cfg.AppPreferences = map[string]interface{}{}
	}
	a.cfg.AppPreferences[key] = req.Value

	// Persist so that the preference survives a server restart.
	_ = saveConfig(a)

	writeJSON(w, http.StatusOK, map[string]interface{}{key: req.Value})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func marshalPreferenceValue(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

