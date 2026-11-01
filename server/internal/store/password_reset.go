package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Password reset tokens are local to this device. Only the SHA-256 hash of a
// token is stored; the token itself exists only in the reset link.

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func deleteExpiredResetTokens(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE expires_at <= ?`, db.Now())
	return err
}

// CreatePasswordResetToken stores a reset token for userID (a users.id or
// db.AdminUserID) that is valid until expiresAt. Older tokens of the same
// account are dropped, so only the most recent link works, and expired tokens
// of all accounts are cleaned up.
func (s *Store) CreatePasswordResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := deleteExpiredResetTokens(ctx, tx); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE user_id = ?`, userID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO password_reset_tokens (id, token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
			db.NewID(), hashResetToken(token), userID, db.Now(), db.FormatTime(expiresAt))
		return mapErr(err)
	})
}

// ConsumePasswordResetToken redeems a reset token: it returns the account the
// token was issued for and deletes all reset tokens of that account, so a
// token works only once. Unknown and expired tokens return ErrNotFound.
func (s *Store) ConsumePasswordResetToken(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := deleteExpiredResetTokens(ctx, tx); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx,
			`SELECT user_id FROM password_reset_tokens WHERE token_hash = ?`, hashResetToken(token)).Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			// Commit the cleanup anyway.
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE user_id = ?`, userID)
		return err
	})
	if err == nil && userID == "" {
		err = ErrNotFound
	}
	return userID, err
}
