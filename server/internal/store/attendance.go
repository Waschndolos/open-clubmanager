package store

import (
	"context"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Attendance states.
const (
	AttendancePresent   = "present"
	AttendanceExcused   = "excused"
	AttendanceUnexcused = "unexcused"
)

// ValidAttendanceStatus reports whether s is a known attendance state.
func ValidAttendanceStatus(s string) bool {
	return s == AttendancePresent || s == AttendanceExcused || s == AttendanceUnexcused
}

// AttendanceScope limits which events a login may record and see attendance
// for. Trainers only see events whose audience contains one of their groups
// or sections; everyone else sees all events.
type AttendanceScope struct {
	All        bool
	GroupIDs   map[string]bool
	SectionIDs map[string]bool
}

// Allows reports whether the event is within the scope. Club-wide events
// (without audience) are not a trainer's own groups.
func (sc AttendanceScope) Allows(e *Event) bool {
	if sc.All {
		return true
	}
	for _, g := range e.Groups {
		if sc.GroupIDs[g.ID] {
			return true
		}
	}
	for _, s := range e.Sections {
		if sc.SectionIDs[s.ID] {
			return true
		}
	}
	return false
}

// TrainerScope returns the groups and sections of the member whose e-mail is
// email (the member linked to a login); empty when there is none.
func (s *Store) TrainerScope(ctx context.Context, email string) (AttendanceScope, error) {
	sc := AttendanceScope{GroupIDs: map[string]bool{}, SectionIDs: map[string]bool{}}
	if email == "" {
		return sc, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT 'group', l.group_id FROM member_groups l
		JOIN member_contacts c ON c.member_id = l.member_id AND c.deleted_at IS NULL
		JOIN members m ON m.id = l.member_id AND m.deleted_at IS NULL
		WHERE l.deleted_at IS NULL AND lower(c.email) = lower(?)
		UNION ALL
		SELECT 'section', l.section_id FROM member_sections l
		JOIN member_contacts c ON c.member_id = l.member_id AND c.deleted_at IS NULL
		JOIN members m ON m.id = l.member_id AND m.deleted_at IS NULL
		WHERE l.deleted_at IS NULL AND lower(c.email) = lower(?)`, email, email)
	if err != nil {
		return sc, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err != nil {
			return sc, err
		}
		if kind == "group" {
			sc.GroupIDs[id] = true
		} else {
			sc.SectionIDs[id] = true
		}
	}
	return sc, rows.Err()
}

// AttendanceMember is a member on the attendance sheet of an occurrence.
type AttendanceMember struct {
	MemberRef
	// Status is nil when nothing has been recorded yet.
	Status *string
	// InAudience is false for members recorded although they are not in the
	// event's audience (guests).
	InAudience bool
}

// AttendanceSheet is the attendance of one occurrence.
type AttendanceSheet struct {
	Occurrence Occurrence
	Members    []AttendanceMember
}

// AttendanceEntry sets (or, with a nil Status, clears) one member's attendance.
type AttendanceEntry struct {
	MemberID string
	Status   *string
}

// occurrenceOf returns the live event and its occurrence on date.
func (s *Store) occurrenceOf(ctx context.Context, eventID string, date time.Time) (Event, Occurrence, error) {
	e, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return e, Occurrence{}, err
	}
	o, ok := OccurrenceOn(&e, date)
	if !ok {
		return e, Occurrence{}, ErrNotFound
	}
	return e, o, nil
}

// AttendanceSheet lists the members expected at an occurrence (the event's
// audience, or all members for club-wide events, who belong to the club on
// that day) and everyone already recorded, with their status.
func (s *Store) AttendanceSheet(ctx context.Context, eventID string, date time.Time) (AttendanceSheet, error) {
	e, o, err := s.occurrenceOf(ctx, eventID, date)
	if err != nil {
		return AttendanceSheet{}, err
	}
	sheet := AttendanceSheet{Occurrence: o, Members: []AttendanceMember{}}
	day := o.Date

	active := `m.deleted_at IS NULL AND (m.exit_date IS NULL OR substr(m.exit_date, 1, 10) >= ?)
		AND (m.entry_date IS NULL OR substr(m.entry_date, 1, 10) <= ?)`
	var query string
	args := []any{}
	if len(e.Groups)+len(e.Sections)+len(e.Roles) == 0 {
		query = `SELECT m.id FROM members m WHERE ` + active
		args = append(args, day, day)
	} else {
		query = `SELECT m.id FROM members m WHERE ` + active + ` AND m.id IN (
			SELECT member_id FROM member_groups WHERE deleted_at IS NULL AND group_id IN (SELECT target_id FROM event_audiences WHERE event_id = ? AND kind = 'group' AND deleted_at IS NULL)
			UNION SELECT member_id FROM member_sections WHERE deleted_at IS NULL AND section_id IN (SELECT target_id FROM event_audiences WHERE event_id = ? AND kind = 'section' AND deleted_at IS NULL)
			UNION SELECT member_id FROM member_roles WHERE deleted_at IS NULL AND role_id IN (SELECT target_id FROM event_audiences WHERE event_id = ? AND kind = 'role' AND deleted_at IS NULL))`
		args = append(args, day, day, eventID, eventID, eventID)
	}
	audience := map[string]bool{}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return sheet, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return sheet, err
		}
		audience[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sheet, err
	}

	recorded := map[string]string{}
	rows, err = s.db.QueryContext(ctx, `SELECT member_id, status FROM event_attendance
		WHERE event_id = ? AND occurrence_date = ? AND deleted_at IS NULL`, eventID, day)
	if err != nil {
		return sheet, err
	}
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			rows.Close()
			return sheet, err
		}
		recorded[id] = status
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sheet, err
	}

	ids := make([]string, 0, len(audience)+len(recorded))
	for id := range audience {
		ids = append(ids, id)
	}
	for id := range recorded {
		if !audience[id] {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return sheet, nil
	}
	rows, err = s.db.QueryContext(ctx, `SELECT id, number, first_name, last_name FROM members
		WHERE deleted_at IS NULL AND id IN (`+placeholders(len(ids))+`)`, stringArgs(ids)...)
	if err != nil {
		return sheet, err
	}
	defer rows.Close()
	for rows.Next() {
		var m AttendanceMember
		if err := rows.Scan(&m.ID, &m.Number, &m.FirstName, &m.LastName); err != nil {
			return sheet, err
		}
		if st, ok := recorded[m.ID]; ok {
			st := st
			m.Status = &st
		}
		m.InAudience = audience[m.ID]
		sheet.Members = append(sheet.Members, m)
	}
	if err := rows.Err(); err != nil {
		return sheet, err
	}
	sort.Slice(sheet.Members, func(i, j int) bool {
		a, b := sheet.Members[i], sheet.Members[j]
		if a.LastName != b.LastName {
			return a.LastName < b.LastName
		}
		if a.FirstName != b.FirstName {
			return a.FirstName < b.FirstName
		}
		return a.Number < b.Number
	})
	return sheet, nil
}

// SetAttendance records the given entries for an occurrence in one
// transaction. Entries are idempotent, so a client may resend them (e.g. after
// working offline).
func (s *Store) SetAttendance(ctx context.Context, eventID string, date time.Time, entries []AttendanceEntry) error {
	_, o, err := s.occurrenceOf(ctx, eventID, date)
	if err != nil {
		return err
	}
	return s.inWriteTx(ctx, func(w *writer) error {
		now := db.Now()
		for _, en := range entries {
			if err := requireLiveMember(ctx, w.Tx, en.MemberID); err != nil {
				return err
			}
			pk := row{"event_id": eventID, "occurrence_date": o.Date, "member_id": en.MemberID}
			if en.Status == nil {
				if _, err := w.update("event_attendance", `event_id = ? AND occurrence_date = ? AND member_id = ? AND deleted_at IS NULL`,
					[]any{eventID, o.Date, en.MemberID}, row{"deleted_at": now}); err != nil {
					return err
				}
				continue
			}
			if err := w.upsert("event_attendance", pk, row{"status": *en.Status, "deleted_at": nil}); err != nil {
				return err
			}
		}
		return nil
	})
}

// AttendanceRecord is one recorded attendance.
type AttendanceRecord struct {
	Occurrence Occurrence
	Member     MemberRef
	Status     string
}

// AttendanceCount summarises the records of an occurrence.
type AttendanceCount struct {
	Occurrence                  Occurrence
	Present, Excused, Unexcused int
}

// AttendanceOccurrences returns the occurrences in the local days from..to
// within the scope (and, if set, for the group or section), with the number
// of recorded states, ordered by start.
func (s *Store) AttendanceOccurrences(ctx context.Context, from, to time.Time, sc AttendanceScope, f AttendanceFilter) ([]AttendanceCount, error) {
	occ, err := s.scopedOccurrences(ctx, from, to, sc, f)
	if err != nil {
		return nil, err
	}
	records, err := s.recordsFor(ctx, occ, from, to)
	if err != nil {
		return nil, err
	}
	counts := make([]AttendanceCount, len(occ))
	index := map[string]int{}
	for i, o := range occ {
		counts[i] = AttendanceCount{Occurrence: o}
		index[o.Event.ID+"|"+o.Date] = i
	}
	for _, r := range records {
		c := &counts[index[r.Occurrence.Event.ID+"|"+r.Occurrence.Date]]
		switch r.Status {
		case AttendancePresent:
			c.Present++
		case AttendanceExcused:
			c.Excused++
		case AttendanceUnexcused:
			c.Unexcused++
		}
	}
	return counts, nil
}

// AttendanceFilter restricts statistics to events for one group or section
// (club-wide events do not count as a group's events).
type AttendanceFilter struct {
	GroupID, SectionID string
}

func (f AttendanceFilter) matches(e *Event) bool {
	if f.GroupID == "" && f.SectionID == "" {
		return true
	}
	for _, g := range e.Groups {
		if g.ID == f.GroupID {
			return true
		}
	}
	for _, s := range e.Sections {
		if s.ID == f.SectionID {
			return true
		}
	}
	return false
}

func (s *Store) scopedOccurrences(ctx context.Context, from, to time.Time, sc AttendanceScope, f AttendanceFilter) ([]Occurrence, error) {
	events, err := s.ListEvents(ctx)
	if err != nil {
		return nil, err
	}
	var allowed []Event
	for i := range events {
		if sc.Allows(&events[i]) && f.matches(&events[i]) {
			allowed = append(allowed, events[i])
		}
	}
	return Occurrences(allowed, from, to, AudienceFilter{}), nil
}

// recordsFor returns the recorded attendance of the given occurrences.
func (s *Store) recordsFor(ctx context.Context, occ []Occurrence, from, to time.Time) ([]AttendanceRecord, error) {
	byKey := map[string]Occurrence{}
	for _, o := range occ {
		byKey[o.Event.ID+"|"+o.Date] = o
	}
	if len(byKey) == 0 {
		return []AttendanceRecord{}, nil
	}
	// Occurrences may start a day before "from" in their own time zone.
	rows, err := s.db.QueryContext(ctx, `SELECT a.event_id, a.occurrence_date, a.status, m.id, m.number, m.first_name, m.last_name
		FROM event_attendance a JOIN members m ON m.id = a.member_id AND m.deleted_at IS NULL
		WHERE a.deleted_at IS NULL AND a.occurrence_date >= ? AND a.occurrence_date <= ?`,
		from.AddDate(0, 0, -1).Format(DateLayout), to.AddDate(0, 0, 1).Format(DateLayout))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AttendanceRecord{}
	for rows.Next() {
		var eventID, date string
		var r AttendanceRecord
		if err := rows.Scan(&eventID, &date, &r.Status, &r.Member.ID, &r.Member.Number, &r.Member.FirstName, &r.Member.LastName); err != nil {
			return nil, err
		}
		o, ok := byKey[eventID+"|"+date]
		if !ok {
			continue // outside the scope or range, or an excluded occurrence
		}
		r.Occurrence = o
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if !a.Occurrence.Start.Equal(b.Occurrence.Start) {
			return a.Occurrence.Start.Before(b.Occurrence.Start)
		}
		if a.Occurrence.Event.ID != b.Occurrence.Event.ID {
			return a.Occurrence.Event.ID < b.Occurrence.Event.ID
		}
		if a.Member.LastName != b.Member.LastName {
			return a.Member.LastName < b.Member.LastName
		}
		return a.Member.FirstName < b.Member.FirstName
	})
	return result, nil
}

// AttendanceRecords returns the recorded attendance in the local days
// from..to within the scope and filter, ordered by occurrence and name.
func (s *Store) AttendanceRecords(ctx context.Context, from, to time.Time, sc AttendanceScope, f AttendanceFilter) ([]AttendanceRecord, error) {
	occ, err := s.scopedOccurrences(ctx, from, to, sc, f)
	if err != nil {
		return nil, err
	}
	return s.recordsFor(ctx, occ, from, to)
}

// MemberAttendance summarises the records of one member.
type MemberAttendance struct {
	Member                      MemberRef
	Present, Excused, Unexcused int
}

// Rate is the share of recorded occurrences the member attended.
func (m MemberAttendance) Rate() float64 {
	total := m.Present + m.Excused + m.Unexcused
	if total == 0 {
		return 0
	}
	return float64(m.Present) / float64(total)
}

// SummarizeByMember counts the records per member, ordered by name.
func SummarizeByMember(records []AttendanceRecord) []MemberAttendance {
	index := map[string]int{}
	var out []MemberAttendance
	for _, r := range records {
		i, ok := index[r.Member.ID]
		if !ok {
			i = len(out)
			index[r.Member.ID] = i
			out = append(out, MemberAttendance{Member: r.Member})
		}
		switch r.Status {
		case AttendancePresent:
			out[i].Present++
		case AttendanceExcused:
			out[i].Excused++
		case AttendanceUnexcused:
			out[i].Unexcused++
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Member, out[j].Member
		if a.LastName != b.LastName {
			return a.LastName < b.LastName
		}
		if a.FirstName != b.FirstName {
			return a.FirstName < b.FirstName
		}
		return a.Number < b.Number
	})
	if out == nil {
		out = []MemberAttendance{}
	}
	return out
}
