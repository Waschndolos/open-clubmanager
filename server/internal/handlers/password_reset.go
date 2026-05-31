package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"sync"
	"time"

	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

const resetTokenTTL = time.Hour

// tokenEntry holds a pending password reset token.
type tokenEntry struct {
	email     string
	expiresAt time.Time
}

// resetTokenStore is an in-memory store for password reset tokens.
var (
	resetTokens   = make(map[string]tokenEntry)
	resetTokensMu sync.Mutex
)

// ForgotPassword handles POST /auth/forgot-password.
// It always returns HTTP 200 to prevent email enumeration.
func (a *API) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req openapi.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: "If the address is registered, a reset link has been sent."})
		return
	}

	if string(req.Email) == "" || string(req.Email) != a.cfg.AdminEmail {
		// Always return 200 – do not reveal whether the email exists.
		writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: "If the address is registered, a reset link has been sent."})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to generate reset token"})
		return
	}

	resetTokensMu.Lock()
	resetTokens[token] = tokenEntry{email: a.cfg.AdminEmail, expiresAt: time.Now().Add(resetTokenTTL)}
	resetTokensMu.Unlock()

	if err := a.sendResetEmail(a.cfg.AdminEmail, token); err != nil {
		// Log but do not expose the error to the caller.
		log.Printf("password reset: failed to send email: %v", err)
	}

	writeJSON(w, http.StatusOK, openapi.MessageResponse{Message: "If the address is registered, a reset link has been sent."})
}

// ResetPassword handles POST /auth/reset-password.
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

	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "password must be at least 8 characters"})
		return
	}

	resetTokensMu.Lock()
	entry, ok := resetTokens[req.Token]
	if ok {
		delete(resetTokens, req.Token)
	}
	resetTokensMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		writeJSON(w, http.StatusBadRequest, openapi.ErrorResponse{Error: "invalid or expired reset token"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, openapi.ErrorResponse{Error: "failed to hash password"})
		return
	}

	// Update the in-memory admin password for the running process.
	a.cfg.AdminPassword = string(hash)

	log.Printf("password reset: admin password updated. New hash: %s", string(hash))

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
