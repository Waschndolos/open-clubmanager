package store

import (
	"context"
	"database/sql"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// NamedEntity is a role, group or section.
type NamedEntity struct {
	ID   string
	Name string
}

// NamedKind selects one of the named-entity tables and its member link table.
type NamedKind struct {
	table     string
	linkTable string
	linkCol   string
}

var (
	Roles    = NamedKind{table: "roles", linkTable: "member_roles", linkCol: "role_id"}
	Groups   = NamedKind{table: "groups", linkTable: "member_groups", linkCol: "group_id"}
	Sections = NamedKind{table: "sections", linkTable: "member_sections", linkCol: "section_id"}
)

// ListNamed returns all live entities of a kind ordered by name.
func (s *Store) ListNamed(ctx context.Context, k NamedKind) ([]NamedEntity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM `+k.table+` WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []NamedEntity{}
	for rows.Next() {
		var e NamedEntity
		if err := rows.Scan(&e.ID, &e.Name); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// CreateNamed inserts a new entity. Names are unique among live entities.
func (s *Store) CreateNamed(ctx context.Context, k NamedKind, name string) (NamedEntity, error) {
	e := NamedEntity{ID: db.NewID(), Name: name}
	now := db.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO `+k.table+` (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, e.ID, name, now, now)
	return e, mapErr(err)
}

// RenameNamed changes the name of a live entity.
func (s *Store) RenameNamed(ctx context.Context, k NamedKind, id, name string) (NamedEntity, error) {
	err := requireAffected(s.db.ExecContext(ctx,
		`UPDATE `+k.table+` SET name = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, name, db.Now(), id))
	return NamedEntity{ID: id, Name: name}, err
}

// DeleteNamed tombstones an entity and its member links.
func (s *Store) DeleteNamed(ctx context.Context, k NamedKind, id string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE `+k.table+` SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE `+k.linkTable+` SET deleted_at = ?, updated_at = ? WHERE `+k.linkCol+` = ? AND deleted_at IS NULL`, now, now, id)
		return err
	})
}

// NamedMembers returns the live members linked to an entity.
func (s *Store) NamedMembers(ctx context.Context, k NamedKind, id string) ([]MemberSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.number, m.first_name, m.last_name FROM members m
		 JOIN `+k.linkTable+` l ON l.member_id = m.id AND l.deleted_at IS NULL
		 WHERE l.`+k.linkCol+` = ? AND m.deleted_at IS NULL
		 ORDER BY m.last_name, m.first_name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []MemberSummary{}
	for rows.Next() {
		var m MemberSummary
		if err := rows.Scan(&m.ID, &m.Number, &m.FirstName, &m.LastName); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// SetNamedMembers makes memberIDs the exact set of members linked to a live
// entity.
func (s *Store) SetNamedMembers(ctx context.Context, k NamedKind, id string, memberIDs []string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM `+k.table+` WHERE id = ? AND deleted_at IS NULL`, id).Scan(&exists)
		if err != nil {
			return mapErr(err)
		}
		return setLinks(ctx, tx, k.linkTable, k.linkCol, id, "member_id", memberIDs, db.Now())
	})
}
