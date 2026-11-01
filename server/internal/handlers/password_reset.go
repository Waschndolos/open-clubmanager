package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"time"

	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

const resetTokenTTL = time.Hour

const forgotPasswordMessage = "If the address is registered, a reset link has been sent."

// ForgotPassword handles POST /auth/forgot-password. It works for the
// config-file admin and for the users of the database. Tokens are stored in
// the database (only their hash), so they survive a restart; a new request
// replaces an older token of the same account.
// It always returns HTTP 200 to prevent email enumeration.
func (a *API) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req openapi.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: forgotPasswordMessage})
		return
	}
	st := a.store()
	email := string(req.Email)
	if email == "" || st == nil {
		// Always return 200 – do not reveal whether the email exists.
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: forgotPasswordMessage})
		return
	}

	var userID, to string
	if a.isAdminEmail(email) {
		userID, to = dbpkg.AdminUserID, a.cfg.AdminEmail
	} else if u, err := st.GetUserByEmail(r.Context(), email); err == nil {
		userID, to = u.ID, u.Email
	} else {
		if !errors.Is(err, store.ErrNotFound) {
			log.Printf("password reset: look up user: %v", err)
		}
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: forgotPasswordMessage})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to generate reset token"})
		return
	}
	if err := st.CreatePasswordResetToken(r.Context(), userID, token, time.Now().Add(resetTokenTTL)); err != nil {
		log.Printf("password reset: store token: %v", err)
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to store reset token"})
		return
	}

	if err := a.sendResetEmail(to, token); err != nil {
		// Log but do not expose the error to the caller.
		log.Printf("password reset: failed to send email: %v", err)
	}

	writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: forgotPasswordMessage})
}

// ResetPassword handles POST /auth/reset-password. A token works once and
// within resetTokenTTL. The new password ends all sessions of the account
// that started before the reset, as a password change does.
func (a *API) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req openapi.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "token and newPassword are required"})
		return
	}

	if !checkPasswordPolicy(w, req.NewPassword, "newPassword") {
		return
	}
	st, ok := a.requireStore(w)
	if !ok {
		return
	}

	// Redeeming deletes the token, so a request that fails afterwards needs
	// a new reset link.
	userID, err := st.ConsumePasswordResetToken(r.Context(), req.Token)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid or expired reset token"})
		return
	}
	if err != nil {
		writeStoreError(w, err, "", "failed to redeem reset token")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	if userID != dbpkg.AdminUserID {
		// SetUserPassword moves updated_at, which revokes older sessions.
		u, err := st.GetUser(r.Context(), userID)
		if err == nil {
			err = st.SetUserPassword(r.Context(), u.Email, string(hash))
		}
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid or expired reset token"})
			return
		}
		if err != nil {
			writeStoreError(w, err, "", "failed to save new password")
			return
		}
		log.Printf("password reset: password of user %s updated", userID)
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: "Password reset successful."})
		return
	}

	if a.cfg.AdminEmail == "" {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid or expired reset token"})
		return
	}
	a.cfg.AdminPassword = string(hash)
	a.cfg.AdminTokensValidAfter = time.Now().Unix()
	if err := saveConfig(a); err != nil {
		log.Printf("password reset: failed to save config: %v", err)
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to save new password"})
		return
	}
	a.revokeAdminRefreshTokens(r.Context())
	log.Printf("password reset: admin password updated")

	writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: "Password reset successful."})
}

// generateToken creates a cryptographically secure random hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sendResetEmail sends the password reset link via SMTP.
// If SMTP is not configured it falls back to logging the link to the console.
func (a *API) sendResetEmail(to, token string) error {
	baseURL := a.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1420"
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)

	smtpCfg := a.cfg.SMTP
	if smtpCfg.Host == "" {
		log.Printf("SMTP not configured – password reset link for %s: %s", to, resetURL)
		return nil
	}

	from := smtpCfg.From
	if from == "" {
		from = smtpCfg.User
	}

	subject := "Password Reset – Open ClubManager"
	body := fmt.Sprintf("To reset your password, click the link below (valid for 1 hour):\n\n%s\n\nIf you did not request a reset, ignore this email.", resetURL)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body))

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	var auth smtp.Auth
	if smtpCfg.User != "" {
		auth = smtp.PlainAuth("", smtpCfg.User, smtpCfg.Password, smtpCfg.Host)
	}
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
