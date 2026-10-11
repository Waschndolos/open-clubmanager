package store

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// AuditEntry is one row of the append-only audit log.
type AuditEntry struct {
	ID        string
	Action    string // "CREATE" | "UPDATE" | "DELETE"
	Entity    string
	EntityID  string
	UserID    string
	Data      *string
	CreatedAt time.Time
}

// Audit appends an entry to the audit log. Failures are logged, never
// returned, so auditing cannot break the operation being audited.
func (s *Store) Audit(ctx context.Context, action, entity, entityID, userID string) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, action, entity, entity_id, user_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		db.NewID(), action, entity, entityID, userID, db.Now())
	if err != nil {
		log.Printf("audit %s %s %s: %v", action, entity, entityID, err)
	}
}

// ListAudit returns all audit entries, newest first.
func (s *Store) ListAudit(ctx context.Context) ([]AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, action, entity, entity_id, user_id, data, created_at FROM audit_log ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var data sql.NullString
		var createdAt string
		if err := rows.Scan(&e.ID, &e.Action, &e.Entity, &e.EntityID, &e.UserID, &data, &createdAt); err != nil {
			return nil, err
		}
		e.Data, e.CreatedAt = nullStr(data), parseTime(createdAt)
		result = append(result, e)
	}
	return result, rows.Err()
}
