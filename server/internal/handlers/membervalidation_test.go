package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// POST and PUT /members refuse invalid members with 400 VALIDATION_FAILED and
// the problem of each field (#193).
func TestMemberValidation(t *testing.T) {
	h := newTestRouter(t)

	send := func(method, path string, body any) (int, openapi.ErrorResponse) {
		t.Helper()
		var buf bytes.Buffer
		if s, ok := body.(string); ok {
			buf.WriteString(s)
		} else if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, &buf))
		var resp openapi.ErrorResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec.Code, resp
	}
	fieldCodes := func(resp openapi.ErrorResponse) string {
		if resp.Fields == nil {
			return ""
		}
		parts := []string{}
		for _, f := range *resp.Fields {
			parts = append(parts, f.Field+":"+f.Code)
		}
		return strings.Join(parts, ",")
	}
	valid := func(extra map[string]any) map[string]any {
		body := map[string]any{"number": 1, "firstName": "Ada", "lastName": "Lovelace"}
		for k, v := range extra {
			body[k] = v
		}
		return body
	}

	tests := []struct {
		name string
		body any
		want string
	}{
		{"empty body", "{}", "number:not_positive,firstName:required,lastName:required"},
		{"whitespace names", valid(map[string]any{"firstName": "  ", "lastName": " "}), "firstName:required,lastName:required"},
		{"negative number", valid(map[string]any{"number": -5}), "number:not_positive"},
		{"invalid IBAN", valid(map[string]any{"iban": "XX00 not an iban"}), "iban:invalid"},
		{"IBAN check digits", valid(map[string]any{"iban": "DE89 3704 0044 0532 0130 01"}), "iban:checksum"},
		{"malformed BIC", valid(map[string]any{"bic": "NOPE"}), "bic:invalid"},
		{"exit before entry", valid(map[string]any{"entryDate": "2020-02-01T00:00:00.000Z", "exitDate": "2020-01-31T00:00:00.000Z"}), "exitDate:before_entry"},
		{"birthday in the future", valid(map[string]any{"birthday": "2099-01-01T00:00:00.000Z"}), "birthday:in_future"},
		{"long first name", valid(map[string]any{"firstName": strings.Repeat("x", 5000)}), "firstName:too_long"},
		{"malformed e-mail", valid(map[string]any{"email": "not-an-email"}), "email:invalid"},
		{"number as text", `{"number":"7","firstName":"Ada","lastName":"Lovelace"}`, "number:invalid"},
	}
	for _, tt := range tests {
		code, resp := send(http.MethodPost, "/api/v2/members", tt.body)
		if code != http.StatusBadRequest || resp.Code == nil || *resp.Code != "VALIDATION_FAILED" || fieldCodes(resp) != tt.want {
			t.Errorf("%s: %d %+v (%s), want 400 VALIDATION_FAILED %s", tt.name, code, resp, fieldCodes(resp), tt.want)
		}
	}
	if _, resp := send(http.MethodPost, "/api/v2/members", valid(map[string]any{"firstName": strings.Repeat("x", 101)})); resp.Fields == nil ||
		(*resp.Fields)[0].Limit == nil || *(*resp.Fields)[0].Limit != 100 {
		t.Errorf("too_long without limit: %+v", resp)
	}

	var list openapi.MembersListResponse
	if doJSON(t, h, http.MethodGet, "/api/v2/members", nil, &list); list.Total != 0 {
		t.Fatalf("invalid members were stored: %+v", list.Items)
	}

	var created openapi.Member
	if code := doJSON(t, h, http.MethodPost, "/api/v2/members", valid(map[string]any{
		"firstName": "  Ada ", "iban": "de89 3704 0044 0532 0130 00", "bic": "cobadeff xxx", "email": "ada@example.org",
		"birthday": "1990-05-17T00:00:00.000Z", "entryDate": "2020-01-01T00:00:00.000Z", "exitDate": "2020-01-01T00:00:00.000Z",
	}), &created); code != http.StatusCreated {
		t.Fatalf("create valid member: %d", code)
	}
	if created.FirstName != "Ada" || *created.Iban != "DE89370400440532013000" || *created.Bic != "COBADEFFXXX" {
		t.Errorf("not normalized: %q %q %q", created.FirstName, *created.Iban, *created.Bic)
	}

	path := "/api/v2/members/" + created.Id
	if code, resp := send(http.MethodPut, path, valid(map[string]any{"lastName": "", "iban": "DE89370400440532013001"})); code != http.StatusBadRequest ||
		fieldCodes(resp) != "lastName:required,iban:checksum" {
		t.Errorf("update: %d %s", code, fieldCodes(resp))
	}
	var got openapi.Member
	if doJSON(t, h, http.MethodGet, path, nil, &got); got.LastName != "Lovelace" || *got.Iban != "DE89370400440532013000" {
		t.Errorf("rejected update was stored: %+v", got)
	}
	// A valid update of a missing member is still a 404.
	if code := doJSON(t, h, http.MethodPut, "/api/v2/members/missing", valid(nil), nil); code != http.StatusNotFound {
		t.Errorf("update missing member: %d", code)
	}
}
