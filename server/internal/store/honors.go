package store

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Honor kinds.
const (
	HonorMembership = "membership"
	HonorBirthday   = "birthday"
	HonorOther      = "other"
)

// Honor is an honor (Ehrung) given to a member. A membership or birthday
// honor with a Milestone marks that stage as done.
type Honor struct {
	ID        string
	Member    MemberRef
	Kind      string
	Milestone *int
	Title     string
	Date      time.Time
	Note      *string
	CreatedAt time.Time
}

// MemberRef identifies a member in lists.
type MemberRef struct {
	ID        string
	Number    int
	FirstName string
	LastName  string
}

// HonorInput is the writable part of an honor.
type HonorInput struct {
	Kind      string
	Milestone *int
	Title     string
	Date      time.Time
	Note      *string
}

// Jubilee is an upcoming anniversary or round birthday of a member.
type Jubilee struct {
	Member MemberRef
	Kind   string // HonorMembership or HonorBirthday
	// Milestone is the number of membership years or the age reached.
	Milestone int
	Date      time.Time
}

// JubileeQuery selects the jubilees in the closed date range [From, To].
type JubileeQuery struct {
	From, To        time.Time
	MembershipYears []int
	Ages            []int
}

const honorSelect = `SELECT h.id, h.kind, h.milestone, h.title, h.honor_date, h.note, h.created_at,
       m.id, m.number, m.first_name, m.last_name
FROM member_honors h
JOIN members m ON m.id = h.member_id
WHERE h.deleted_at IS NULL AND m.deleted_at IS NULL`

func scanHonor(r scanner) (Honor, error) {
	var h Honor
	var milestone sql.NullInt64
	var date, createdAt string
	var note sql.NullString
	err := r.Scan(&h.ID, &h.Kind, &milestone, &h.Title, &date, &note, &createdAt,
		&h.Member.ID, &h.Member.Number, &h.Member.FirstName, &h.Member.LastName)
	if err != nil {
		return h, mapErr(err)
	}
	if milestone.Valid {
		v := int(milestone.Int64)
		h.Milestone = &v
	}
	h.Date, h.CreatedAt, h.Note = parseTime(date), parseTime(createdAt), nullStr(note)
	return h, nil
}

// ListHonors returns the honors of one member, or of all members when
// memberID is empty, newest first.
func (s *Store) ListHonors(ctx context.Context, memberID string) ([]Honor, error) {
	query, args := honorSelect, []any(nil)
	if memberID != "" {
		query += ` AND h.member_id = ?`
		args = append(args, memberID)
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY h.honor_date DESC, h.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Honor{}
	for rows.Next() {
		h, err := scanHonor(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

// CreateHonor records an honor for a live member.
func (s *Store) CreateHonor(ctx context.Context, memberID string, in HonorInput) (Honor, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			return err
		}
		return w.insert("member_honors", row{
			"id": id, "member_id": memberID, "kind": in.Kind, "milestone": intArg(in.Milestone),
			"title": in.Title, "honor_date": db.FormatTime(in.Date), "note": strArg(in.Note), "created_at": db.Now(),
		})
	})
	if err != nil {
		return Honor{}, err
	}
	return scanHonor(s.db.QueryRowContext(ctx, honorSelect+` AND h.id = ?`, id))
}

// DeleteHonor tombstones an honor.
func (s *Store) DeleteHonor(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("member_honors", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

// MembershipAdjust returns the manual correction of a member's membership years.
func (s *Store) MembershipAdjust(ctx context.Context, memberID string) (int, error) {
	var one int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM members WHERE id = ? AND deleted_at IS NULL`, memberID).Scan(&one); err != nil {
		return 0, mapErr(err)
	}
	var years int
	err := s.db.QueryRowContext(ctx,
		`SELECT years_adjust FROM member_membership WHERE member_id = ? AND deleted_at IS NULL`, memberID).Scan(&years)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return years, err
}

// SetMembershipAdjust sets the manual correction (in years, may be negative)
// that is added to the membership years computed from the entry date.
func (s *Store) SetMembershipAdjust(ctx context.Context, memberID string, years int) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			return err
		}
		return w.upsert("member_membership", row{"member_id": memberID},
			row{"years_adjust": years, "deleted_at": nil, "created_at": db.Now()})
	})
}

func intArg(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

// Jubilees returns the anniversaries and round birthdays of current members
// in the query range that have not been honored yet, ordered by date.
func (s *Store) Jubilees(ctx context.Context, q JubileeQuery) ([]Jubilee, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, m.number, m.first_name, m.last_name, m.birthday, m.entry_date, COALESCE(ms.years_adjust, 0)
		FROM members m LEFT JOIN member_membership ms ON ms.member_id = m.id AND ms.deleted_at IS NULL
		WHERE m.deleted_at IS NULL AND (m.exit_date IS NULL OR m.exit_date > ?)`, db.FormatTime(q.From))
	if err != nil {
		return nil, err
	}
	var members []jubileeMember
	for rows.Next() {
		var m jubileeMember
		var birthday, entry sql.NullString
		if err := rows.Scan(&m.ref.ID, &m.ref.Number, &m.ref.FirstName, &m.ref.LastName, &birthday, &entry, &m.adjust); err != nil {
			rows.Close()
			return nil, err
		}
		m.birthday, m.entry = nullTime(birthday), nullTime(entry)
		members = append(members, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	honored := map[honorKey]bool{}
	hrows, err := s.db.QueryContext(ctx, `SELECT member_id, kind, milestone FROM member_honors
		WHERE deleted_at IS NULL AND milestone IS NOT NULL AND kind IN ('membership', 'birthday')`)
	if err != nil {
		return nil, err
	}
	defer hrows.Close()
	for hrows.Next() {
		var k honorKey
		if err := hrows.Scan(&k.memberID, &k.kind, &k.milestone); err != nil {
			return nil, err
		}
		honored[k] = true
	}
	if err := hrows.Err(); err != nil {
		return nil, err
	}
	return computeJubilees(members, honored, q), nil
}

type honorKey struct {
	memberID  string
	kind      string
	milestone int
}

type jubileeMember struct {
	ref      MemberRef
	birthday *time.Time
	entry    *time.Time
	adjust   int
}

// computeJubilees is the date arithmetic behind Jubilees. Membership years
// count from the entry date plus the manual correction; a milestone date that
// does not exist (29 February) falls on 1 March.
func computeJubilees(members []jubileeMember, honored map[honorKey]bool, q JubileeQuery) []Jubilee {
	from, to := dateOnly(q.From), dateOnly(q.To)
	result := []Jubilee{}
	add := func(m jubileeMember, kind string, base time.Time, milestone, offset int) {
		d := time.Date(base.Year()+milestone-offset, base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
		if d.Before(from) || d.After(to) || honored[honorKey{m.ref.ID, kind, milestone}] {
			return
		}
		result = append(result, Jubilee{Member: m.ref, Kind: kind, Milestone: milestone, Date: d})
	}
	for _, m := range members {
		if m.entry != nil {
			for _, y := range q.MembershipYears {
				add(m, HonorMembership, *m.entry, y, m.adjust)
			}
		}
		if m.birthday != nil {
			for _, a := range q.Ages {
				add(m, HonorBirthday, *m.birthday, a, 0)
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].Date.Equal(result[j].Date) {
			return result[i].Date.Before(result[j].Date)
		}
		return result[i].Member.Number < result[j].Member.Number
	})
	return result
}

func dateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
