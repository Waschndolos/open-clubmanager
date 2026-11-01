package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// minPasswordLength is the password policy for every login: the setup admin,
// users created in the user management, password changes and resets.
const minPasswordLength = 8

// checkPasswordPolicy writes a 400 response with the code PASSWORD_TOO_SHORT
// and returns false when password does not meet the policy. field names the
// request property in the error message.
func checkPasswordPolicy(w http.ResponseWriter, password, field string) bool {
	if utf8.RuneCountInString(password) >= minPasswordLength {
		return true
	}
	code := "PASSWORD_TOO_SHORT"
	writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Code: &code, Error: fmt.Sprintf("%s must be at least %d characters", field, minPasswordLength)})
	return false
}

// isAdminEmail reports whether email is the config-file administrator's
// address, ignoring case.
func (a *API) isAdminEmail(email string) bool {
	return a.cfg.AdminEmail != "" && strings.EqualFold(strings.TrimSpace(email), a.cfg.AdminEmail)
}

// writeEmailInUse writes the 409 response for an e-mail address that already
// belongs to the administrator or another user.
func writeEmailInUse(w http.ResponseWriter) {
	code := "EMAIL_IN_USE"
	writeJSON(w, http.StatusConflict, openapi.ErrorResponse{Code: &code, Error: "email already in use"})
}
