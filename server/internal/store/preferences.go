package store

import (
	"context"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Preference is a per-user UI setting kept on this device only.
type Preference struct {
	ID        string
	UserID    string
	Key       string
	Value     string
	UpdatedAt time.Time
}

// ListPreferences returns all preferences of a user ordered by key.
func (s *Store) ListPreferences(ctx context.Context, userID string) ([]Preference, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, key, value, updated_at FROM user_preferences WHERE user_id = ? ORDER BY key`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Preference{}
	for rows.Next() {
		var p Preference
		var updatedAt string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Key, &p.Value, &updatedAt); err != nil {
			return nil, err
		}
		p.UpdatedAt = parseTime(updatedAt)
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetPreference returns one preference of a user.
func (s *Store) GetPreference(ctx context.Context, userID, key string) (Preference, error) {
	var p Preference
	var updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, key, value, updated_at FROM user_preferences WHERE user_id = ? AND key = ?`, userID, key,
	).Scan(&p.ID, &p.UserID, &p.Key, &p.Value, &updatedAt)
	if err != nil {
		return p, mapErr(err)
	}
	p.UpdatedAt = parseTime(updatedAt)
	return p, nil
}

// SetPreference creates or replaces a preference of a user.
func (s *Store) SetPreference(ctx context.Context, userID, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_preferences (id, user_id, key, value, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		db.NewID(), userID, key, value, db.Now())
	return mapErr(err)
}
