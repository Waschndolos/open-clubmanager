package handlers

import (
	"encoding/json"
	"net/http"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// preferenceUserID is the user all preferences are stored under. The desktop
// deployment currently shares one set of preferences between all logins, so
// they are kept under the config-file admin's id.
const preferenceUserID = dbpkg.AdminUserID

// ── User Preferences ─────────────────────────────────────────────────────────

// ListPreferences returns all preferences stored for the current (admin) user.
func (a *API) ListPreferences(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	prefs, err := st.ListPreferences(r.Context(), preferenceUserID)
	if err != nil {
		writeStoreError(w, err, "", "failed to list preferences")
		return
	}
	result := make([]openapi.UserPreferenceRecord, len(prefs))
	for i, p := range prefs {
		result[i] = openapi.UserPreferenceRecord{Id: p.ID, UserId: p.UserID, Key: p.Key, Value: p.Value, UpdatedAt: p.UpdatedAt}
	}
	writeJSON(w, http.StatusOK, result)
}

// UpsertPreference creates or updates a preference for the current user.
func (a *API) UpsertPreference(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok {
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

	if err := st.SetPreference(r.Context(), preferenceUserID, req.Key, valueStr); err != nil {
		writeStoreError(w, err, "", "failed to upsert preference")
		return
	}
	writeJSON(w, http.StatusOK, openapi.SuccessResponse{Success: true})
}

// GetPreferenceByKey returns a single user preference by key.
func (a *API) GetPreferenceByKey(w http.ResponseWriter, r *http.Request, identifier string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	p, err := st.GetPreference(r.Context(), preferenceUserID, identifier)
	if err != nil {
		writeStoreError(w, err, "preference not found", "failed to load preference")
		return
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(p.Value), &parsed); err != nil {
		parsed = p.Value // fall back to raw string
	}

	writeJSON(w, http.StatusOK, openapi.UserPreferenceValueResponse{
		Key:       identifier,
		Value:     parsed,
		UpdatedAt: p.UpdatedAt,
	})
}

// UpdatePreferenceById updates a user preference by key (identifier).
func (a *API) UpdatePreferenceById(w http.ResponseWriter, r *http.Request, identifier string) {
	st, ok := a.requireStore(w)
	if !ok {
		return
	}
	var req openapi.UpdateUserPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if err := st.SetPreference(r.Context(), preferenceUserID, identifier, req.Value); err != nil {
		writeStoreError(w, err, "", "failed to update preference")
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
