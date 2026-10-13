package store

import (
	"context"

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
	// audience is the event_audiences.kind of this entity.
	audience string
}

var (
	Roles    = NamedKind{table: "roles", linkTable: "member_roles", linkCol: "role_id", audience: AudienceRole}
	Groups   = NamedKind{table: "groups", linkTable: "member_groups", linkCol: "group_id", audience: AudienceGroup}
	Sections = NamedKind{table: "sections", linkTable: "member_sections", linkCol: "section_id", audience: AudienceSection}
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
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := w.requireUnique(k.table, "name", name, "id", e.ID); err != nil {
			return err
		}
		return w.insert(k.table, row{"id": e.ID, "name": name, "created_at": db.Now()})
	})
	return e, err
}

// RenameNamed changes the name of a live entity.
func (s *Store) RenameNamed(ctx context.Context, k NamedKind, id, name string) (NamedEntity, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update(k.table, `id = ? AND deleted_at IS NULL`, []any{id}, row{"name": name})); err != nil {
			return err
		}
		return w.requireUnique(k.table, "name", name, "id", id)
	})
	return NamedEntity{ID: id, Name: name}, err
}

// DeleteNamed tombstones an entity with its member links and event audiences.
func (s *Store) DeleteNamed(ctx context.Context, k NamedKind, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update(k.table, `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		if _, err := w.update(k.linkTable, k.linkCol+` = ? AND deleted_at IS NULL`, []any{id}, tombstone); err != nil {
			return err
		}
		_, err := w.update("event_audiences", `kind = ? AND target_id = ? AND deleted_at IS NULL`, []any{k.audience, id}, tombstone)
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
	return s.inWriteTx(ctx, func(w *writer) error {
		var exists int
		err := w.QueryRowContext(ctx, `SELECT 1 FROM `+k.table+` WHERE id = ? AND deleted_at IS NULL`, id).Scan(&exists)
		if err != nil {
			return mapErr(err)
		}
		return setLinks(w, k.linkTable, k.linkCol, id, "member_id", memberIDs, db.Now())
	})
}
