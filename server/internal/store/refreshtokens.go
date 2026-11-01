package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// RefreshToken is a stored refresh token. Only the hash of the token is kept.
// Tokens of one login share a family; a refresh replaces the used token by a
// new one of the same family. refresh_tokens is local to this device.
type RefreshToken struct {
	ID       string
	FamilyID string
	Email    string
	AppRole  string
	// CreatedAt is when this token was issued; like the issue time of an
	// access token it is checked against the account's last change.
	CreatedAt time.Time
	ExpiresAt time.Time
	// ReplacedAt is set once the token was used for a refresh.
	ReplacedAt *time.Time
}

// CreateRefreshToken stores a new token. userID is empty for the
// administrator from config.json. Expired tokens are pruned on the way.
func (s *Store) CreateRefreshToken(ctx context.Context, tokenHash, familyID, email, appRole, userID string, createdAt, expiresAt time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE expires_at < ?`, db.FormatTime(createdAt)); err != nil {
			return err
		}
		var uid any
		if userID != "" {
			uid = userID
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO refresh_tokens (id, token_hash, family_id, email, app_role, user_id, created_at, expires_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			db.NewID(), tokenHash, familyID, email, appRole, uid, db.FormatTime(createdAt), db.FormatTime(expiresAt))
		return mapErr(err)
	})
}

// GetRefreshToken returns the token with the given hash, replaced or not.
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error) {
	var t RefreshToken
	var createdAt, expiresAt string
	var replacedAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, family_id, email, app_role, created_at, expires_at, replaced_at FROM refresh_tokens WHERE token_hash = ?`,
		tokenHash).Scan(&t.ID, &t.FamilyID, &t.Email, &t.AppRole, &createdAt, &expiresAt, &replacedAt)
	if err != nil {
		return t, mapErr(err)
	}
	t.CreatedAt = parseTime(createdAt)
	t.ExpiresAt = parseTime(expiresAt)
	if replacedAt.Valid {
		r := parseTime(replacedAt.String)
		t.ReplacedAt = &r
	}
	return t, nil
}

// RotateRefreshToken marks old as replaced and stores a new token of the same
// family in its place. It returns ErrNotFound when old was replaced or
// revoked in the meantime, so only one of two concurrent refreshes rotates.
func (s *Store) RotateRefreshToken(ctx context.Context, old RefreshToken, newHash string, now, expiresAt time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE refresh_tokens SET replaced_at = ? WHERE id = ? AND replaced_at IS NULL`, db.FormatTime(now), old.ID))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO refresh_tokens (id, token_hash, family_id, email, app_role, user_id, created_at, expires_at)
			 SELECT ?, ?, family_id, email, app_role, user_id, ?, ? FROM refresh_tokens WHERE id = ?`,
			db.NewID(), newHash, db.FormatTime(now), db.FormatTime(expiresAt), old.ID)
		return mapErr(err)
	})
}

// RevokeRefreshTokenFamily deletes all tokens of one login.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE family_id = ?`, familyID)
	return err
}

// RevokeRefreshTokensForEmail deletes all tokens issued to an e-mail address.
func (s *Store) RevokeRefreshTokensForEmail(ctx context.Context, email string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE email = ?`, email)
	return err
}
