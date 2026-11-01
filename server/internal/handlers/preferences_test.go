package handlers

import (
	"net/http"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// A preference that was never saved is not an error: the frontend asks for
// memberTableColumns on every page load and falls back to its defaults.
func TestGetPreferenceByKeyWithoutStoredValue(t *testing.T) {
	h := newTestRouter(t)

	if code := doJSON(t, h, http.MethodGet, "/api/v2/preferences/memberTableColumns", nil, nil); code != http.StatusNoContent {
		t.Fatalf("missing preference: %d, want 204", code)
	}

	body := map[string]any{"key": "memberTableColumns", "value": map[string]bool{"id": false}}
	if code := doJSON(t, h, http.MethodPost, "/api/v2/preferences", body, nil); code != http.StatusOK {
		t.Fatalf("upsert: %d", code)
	}
	var got openapi.UserPreferenceValueResponse
	if code := doJSON(t, h, http.MethodGet, "/api/v2/preferences/memberTableColumns", nil, &got); code != http.StatusOK {
		t.Fatalf("stored preference: %d, want 200", code)
	}
	if v, ok := got.Value.(map[string]any); !ok || v["id"] != false {
		t.Errorf("value = %#v, want {id: false}", got.Value)
	}
}
