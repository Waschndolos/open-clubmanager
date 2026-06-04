package db

import "database/sql"

// LogAudit writes an entry to the AuditLog table. Errors are silently ignored
// so audit failures never interrupt the actual operation.
func LogAudit(db *sql.DB, action, entity string, entityID int, userID, data string) {
	_, _ = db.Exec(
		`INSERT INTO "AuditLog" ("action","entity","entityId","userId","data") VALUES (?,?,?,?,?)`,
		action, entity, entityID, userID, nullableString(data),
	)
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

