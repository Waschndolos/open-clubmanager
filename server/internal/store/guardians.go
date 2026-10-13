package store

import (
	"context"
	"database/sql"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Guardian is a guardian (Erziehungsberechtigte/r) of a member, either another
// member or a person given by name.
type Guardian struct {
	ID       string
	MemberID string
	Member   *MemberRef // the guardian, when it is a member
	Name     *string
	Relation *string
	Email    *string
	Phone    *string
}

// GuardianInput is the writable part of a guardian; either GuardianMemberID
// or Name is set.
type GuardianInput struct {
	GuardianMemberID *string
	Name             *string
	Relation         *string
	Email            *string
	Phone            *string
}

// ListGuardians returns the guardians of a member.
func (s *Store) ListGuardians(ctx context.Context, memberID string) ([]Guardian, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT g.id, g.member_id, g.name, g.relation, g.email, g.phone,
		m.id, m.number, m.first_name, m.last_name
		FROM member_guardians g
		LEFT JOIN members m ON m.id = g.guardian_member_id AND m.deleted_at IS NULL
		WHERE g.member_id = ? AND g.deleted_at IS NULL
		ORDER BY g.created_at, g.id`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Guardian{}
	for rows.Next() {
		var g Guardian
		var name, relation, email, phone, mid, first, last sql.NullString
		var number sql.NullInt64
		if err := rows.Scan(&g.ID, &g.MemberID, &name, &relation, &email, &phone, &mid, &number, &first, &last); err != nil {
			return nil, err
		}
		g.Name, g.Relation, g.Email, g.Phone = nullStr(name), nullStr(relation), nullStr(email), nullStr(phone)
		if mid.Valid {
			g.Member = &MemberRef{ID: mid.String, Number: int(number.Int64), FirstName: first.String, LastName: last.String}
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// CreateGuardian adds a guardian to a live member.
func (s *Store) CreateGuardian(ctx context.Context, memberID string, in GuardianInput) (string, error) {
	if in.GuardianMemberID != nil && *in.GuardianMemberID == memberID {
		return "", ErrInvalidReference
	}
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			return err
		}
		if in.GuardianMemberID != nil {
			if err := requireLiveMember(ctx, w.Tx, *in.GuardianMemberID); err != nil {
				return err
			}
		}
		return w.insert("member_guardians", row{
			"id": id, "member_id": memberID, "guardian_member_id": strArg(in.GuardianMemberID), "name": strArg(in.Name),
			"relation": strArg(in.Relation), "email": strArg(in.Email), "phone": strArg(in.Phone), "created_at": db.Now(),
		})
	})
	return id, err
}

// DeleteGuardian tombstones a guardian entry.
func (s *Store) DeleteGuardian(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("member_guardians", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}
