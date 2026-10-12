package store

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Qualification statuses.
const (
	QualValid    = "valid"
	QualExpiring = "expiring"
	QualExpired  = "expired"
)

// QualificationType is a kind of qualification (trainer licence, first aid, ...).
type QualificationType struct {
	ID             string
	Name           string
	ValidityMonths *int
	InspectionOnly bool
}

// QualificationTypeInput is the writable part of a type.
type QualificationTypeInput struct {
	Name           string
	ValidityMonths *int
	InspectionOnly bool
}

// Qualification is a qualification held by a member.
type Qualification struct {
	ID         string
	Member     MemberRef
	Type       QualificationType
	IssuedDate time.Time
	ExpiryDate *time.Time
	Note       *string
	// Status is QualValid, QualExpiring or QualExpired relative to the query.
	Status string
}

// QualificationInput is the writable part of a member qualification.
type QualificationInput struct {
	TypeID     string
	IssuedDate time.Time
	// ExpiryDate defaults to IssuedDate plus the validity of the type.
	ExpiryDate *time.Time
	Note       *string
}

// ListQualificationTypes returns all live types ordered by name.
func (s *Store) ListQualificationTypes(ctx context.Context) ([]QualificationType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, validity_months, inspection_only FROM qualification_types WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []QualificationType{}
	for rows.Next() {
		t, err := scanQualType(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func scanQualType(r scanner) (QualificationType, error) {
	var t QualificationType
	var months sql.NullInt64
	var inspection int
	if err := r.Scan(&t.ID, &t.Name, &months, &inspection); err != nil {
		return t, mapErr(err)
	}
	if months.Valid {
		v := int(months.Int64)
		t.ValidityMonths = &v
	}
	t.InspectionOnly = inspection != 0
	return t, nil
}

func (s *Store) getQualType(ctx context.Context, id string) (QualificationType, error) {
	return scanQualType(s.db.QueryRowContext(ctx,
		`SELECT id, name, validity_months, inspection_only FROM qualification_types WHERE id = ? AND deleted_at IS NULL`, id))
}

func qualTypeColumns(in QualificationTypeInput) row {
	inspection := 0
	if in.InspectionOnly {
		inspection = 1
	}
	return row{"name": in.Name, "validity_months": intArg(in.ValidityMonths), "inspection_only": inspection}
}

// CreateQualificationType inserts a type; names are unique.
func (s *Store) CreateQualificationType(ctx context.Context, in QualificationTypeInput) (QualificationType, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := w.requireUnique("qualification_types", "name", in.Name, "id", id); err != nil {
			return err
		}
		values := qualTypeColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("qualification_types", values)
	})
	if err != nil {
		return QualificationType{}, err
	}
	return s.getQualType(ctx, id)
}

// UpdateQualificationType replaces the fields of a live type.
func (s *Store) UpdateQualificationType(ctx context.Context, id string, in QualificationTypeInput) (QualificationType, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update("qualification_types", `id = ? AND deleted_at IS NULL`, []any{id}, qualTypeColumns(in))); err != nil {
			return err
		}
		return w.requireUnique("qualification_types", "name", in.Name, "id", id)
	})
	if err != nil {
		return QualificationType{}, err
	}
	return s.getQualType(ctx, id)
}

// DeleteQualificationType tombstones a type together with the qualifications of it.
func (s *Store) DeleteQualificationType(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update("qualification_types", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		_, err := w.update("member_qualifications", `type_id = ? AND deleted_at IS NULL`, []any{id}, tombstone)
		return err
	})
}

// CreateQualification records a qualification of a live member. Inspection-only
// types never keep a note.
func (s *Store) CreateQualification(ctx context.Context, memberID string, in QualificationInput) (string, error) {
	typ, err := s.getQualType(ctx, in.TypeID)
	if err == ErrNotFound {
		return "", ErrInvalidReference
	}
	if err != nil {
		return "", err
	}
	expiry := in.ExpiryDate
	if expiry == nil && typ.ValidityMonths != nil {
		e := in.IssuedDate.AddDate(0, *typ.ValidityMonths, 0)
		expiry = &e
	}
	note := in.Note
	if typ.InspectionOnly {
		note = nil
	}
	id := db.NewID()
	err = s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			return err
		}
		return w.insert("member_qualifications", row{
			"id": id, "member_id": memberID, "type_id": in.TypeID, "issued_date": db.FormatTime(in.IssuedDate),
			"expiry_date": timeArg(expiry), "note": strArg(note), "created_at": db.Now(),
		})
	})
	return id, err
}

// DeleteQualification tombstones a member qualification.
func (s *Store) DeleteQualification(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("member_qualifications", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

// Qualifications returns the current qualification per member and type (the
// one valid longest) of current members, with its status as of now: expiring
// when it ends within warnDays. Expired and expiring ones come first.
func (s *Store) Qualifications(ctx context.Context, now time.Time, warnDays int) ([]Qualification, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT q.id, q.issued_date, q.expiry_date, q.note,
		m.id, m.number, m.first_name, m.last_name, t.id, t.name, t.validity_months, t.inspection_only
		FROM member_qualifications q
		JOIN members m ON m.id = q.member_id AND m.deleted_at IS NULL AND (m.exit_date IS NULL OR m.exit_date > ?)
		JOIN qualification_types t ON t.id = q.type_id AND t.deleted_at IS NULL
		WHERE q.deleted_at IS NULL`, db.FormatTime(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []Qualification
	for rows.Next() {
		var q Qualification
		var issued string
		var expiry, note sql.NullString
		var months sql.NullInt64
		var inspection int
		if err := rows.Scan(&q.ID, &issued, &expiry, &note, &q.Member.ID, &q.Member.Number, &q.Member.FirstName,
			&q.Member.LastName, &q.Type.ID, &q.Type.Name, &months, &inspection); err != nil {
			return nil, err
		}
		if months.Valid {
			v := int(months.Int64)
			q.Type.ValidityMonths = &v
		}
		q.Type.InspectionOnly = inspection != 0
		q.IssuedDate, q.ExpiryDate, q.Note = parseTime(issued), nullTime(expiry), nullStr(note)
		all = append(all, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return currentQualifications(all, now, warnDays), nil
}

// currentQualifications keeps the latest qualification per member and type and
// sets the status.
func currentQualifications(all []Qualification, now time.Time, warnDays int) []Qualification {
	type key struct{ member, typ string }
	best := map[key]Qualification{}
	for _, q := range all {
		k := key{q.Member.ID, q.Type.ID}
		if cur, ok := best[k]; !ok || endsLater(q, cur) {
			best[k] = q
		}
	}
	today := dateOnly(now)
	result := make([]Qualification, 0, len(best))
	for _, q := range best {
		q.Status = QualValid
		if q.ExpiryDate != nil {
			exp := dateOnly(*q.ExpiryDate)
			switch {
			case exp.Before(today):
				q.Status = QualExpired
			case !exp.After(today.AddDate(0, 0, warnDays)):
				q.Status = QualExpiring
			}
		}
		result = append(result, q)
	}
	rank := map[string]int{QualExpired: 0, QualExpiring: 1, QualValid: 2}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if rank[a.Status] != rank[b.Status] {
			return rank[a.Status] < rank[b.Status]
		}
		// Entries without an expiry date come last within their status.
		if (a.ExpiryDate == nil) != (b.ExpiryDate == nil) {
			return b.ExpiryDate == nil
		}
		if a.ExpiryDate != nil && !a.ExpiryDate.Equal(*b.ExpiryDate) {
			return a.ExpiryDate.Before(*b.ExpiryDate)
		}
		if a.Member.Number != b.Member.Number {
			return a.Member.Number < b.Member.Number
		}
		if a.Type.Name != b.Type.Name {
			return a.Type.Name < b.Type.Name
		}
		return a.ID < b.ID
	})
	return result
}

// endsLater reports whether a is valid longer than b (no expiry is longest).
func endsLater(a, b Qualification) bool {
	switch {
	case a.ExpiryDate == nil && b.ExpiryDate == nil:
		return a.IssuedDate.After(b.IssuedDate)
	case a.ExpiryDate == nil:
		return true
	case b.ExpiryDate == nil:
		return false
	}
	return a.ExpiryDate.After(*b.ExpiryDate)
}
