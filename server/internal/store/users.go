package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// User is an application login stored in the database (scope admin). The
// administrator from config.json is not a row here.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	AppRole      string
	CreatedAt    time.Time
}

const userSelect = `SELECT id, email, password, app_role, created_at FROM users WHERE deleted_at IS NULL`

func scanUser(row scanner) (User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.AppRole, &createdAt); err != nil {
		return u, mapErr(err)
	}
	u.CreatedAt = parseTime(createdAt)
	return u, nil
}

// ListUsers returns all live users in creation order.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, userSelect+` ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// GetUser returns a live user by id.
func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx, userSelect+` AND id = ?`, id))
}

// GetUserByEmail returns a live user by e-mail address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx, userSelect+` AND email = ?`, email))
}

// CreateUser inserts a user; passwordHash must already be hashed.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, appRole string) (User, error) {
	id := db.NewID()
	now := db.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password, app_role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, email, passwordHash, appRole, now, now)
	if err != nil {
		return User{}, mapErr(err)
	}
	return s.GetUser(ctx, id)
}

// UpdateUser changes the e-mail and/or role of a live user; nil leaves a field
// unchanged.
func (s *Store) UpdateUser(ctx context.Context, id string, email, appRole *string) (User, error) {
	err := requireAffected(s.db.ExecContext(ctx,
		`UPDATE users SET email = COALESCE(?, email), app_role = COALESCE(?, app_role), updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		strArg(email), strArg(appRole), db.Now(), id))
	if err != nil {
		return User{}, err
	}
	return s.GetUser(ctx, id)
}

// SetUserPassword stores a new password hash for the live user with the given
// e-mail address.
func (s *Store) SetUserPassword(ctx context.Context, email, passwordHash string) error {
	return requireAffected(s.db.ExecContext(ctx,
		`UPDATE users SET password = ?, updated_at = ? WHERE email = ? AND deleted_at IS NULL`, passwordHash, db.Now(), email))
}

// DeleteUser tombstones a user and drops its device-local session tokens.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE users SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id))
		if err != nil {
			return err
		}
		for _, table := range []string{"refresh_tokens", "password_reset_tokens"} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = ?`, id); err != nil {
				return err
			}
		}
		return nil
	})
}
