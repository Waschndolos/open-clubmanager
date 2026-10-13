package store

import (
	"context"
	"database/sql"
	"sort"
	"time"
	// Recurring events are expanded in IANA time zones; embed the database so
	// this also works where the OS has none (e.g. the Windows desktop build).
	_ "time/tzdata"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Event categories.
const (
	EventTraining    = "training"
	EventFestival    = "festival"
	EventMeeting     = "meeting"
	EventCompetition = "competition"
	EventOther       = "other"
)

// RecurBiweekly repeats an event every 14 days; events also use RecurNone,
// RecurWeekly and RecurMonthly of tasks.
const RecurBiweekly = "biweekly"

// Audience kinds of an event.
const (
	AudienceGroup   = "group"
	AudienceSection = "section"
	AudienceRole    = "role"
)

// DateLayout is the layout of local dates (occurrence dates, exceptions).
const DateLayout = "2006-01-02"

// Event is a club event, possibly a recurring series.
type Event struct {
	ID          string
	Title       string
	Description *string
	Location    *string
	Category    string
	// Start and End are the first occurrence. For all-day events only their
	// local dates count and End is the midnight after the last day
	// (exclusive, as in iCalendar).
	Start, End time.Time
	AllDay     bool
	TimeZone   string
	Recurrence string
	// RecurrenceUntil is the last local date an occurrence may start on.
	RecurrenceUntil *time.Time
	// Exceptions are the local dates (DateLayout) of excluded occurrences.
	Exceptions []string
	Groups     []NamedEntity
	Sections   []NamedEntity
	Roles      []NamedEntity
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// EventInput is the writable part of an event.
type EventInput struct {
	Title           string
	Description     *string
	Location        *string
	Category        string
	Start, End      time.Time
	AllDay          bool
	TimeZone        string
	Recurrence      string
	RecurrenceUntil *time.Time
	GroupIDs        []string
	SectionIDs      []string
	RoleIDs         []string
}

// Occurrence is one occurrence of an event.
type Occurrence struct {
	Event *Event
	// Date is the local start date (DateLayout); it identifies the occurrence.
	Date       string
	Start, End time.Time
}

// AudienceFilter selects events for a group, section or role. Events for the
// whole club always match; an empty filter matches everything.
type AudienceFilter struct {
	GroupID, SectionID, RoleID string
}

// ValidEventCategory reports whether c is a known category.
func ValidEventCategory(c string) bool {
	switch c {
	case EventTraining, EventFestival, EventMeeting, EventCompetition, EventOther:
		return true
	}
	return false
}

// ValidEventRecurrence reports whether r is a known event recurrence.
func ValidEventRecurrence(r string) bool {
	switch r {
	case RecurNone, RecurWeekly, RecurBiweekly, RecurMonthly:
		return true
	}
	return false
}

const eventSelect = `SELECT id, title, description, location, category, starts_at, ends_at, all_day, time_zone,
       recurrence, recurrence_until, created_at, updated_at
FROM events WHERE deleted_at IS NULL`

// ListEvents returns all live events with audiences and exceptions.
func (s *Store) ListEvents(ctx context.Context) ([]Event, error) {
	return s.queryEvents(ctx, "")
}

// GetEvent returns a live event.
func (s *Store) GetEvent(ctx context.Context, id string) (Event, error) {
	list, err := s.queryEvents(ctx, id)
	if err != nil {
		return Event{}, err
	}
	if len(list) == 0 {
		return Event{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) queryEvents(ctx context.Context, id string) ([]Event, error) {
	query, args := eventSelect, []any(nil)
	if id != "" {
		query += ` AND id = ?`
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY starts_at, id`, args...)
	if err != nil {
		return nil, err
	}
	result := []Event{}
	index := map[string]int{}
	for rows.Next() {
		var e Event
		var desc, loc, until sql.NullString
		var start, end, created, updated string
		var allDay int
		if err := rows.Scan(&e.ID, &e.Title, &desc, &loc, &e.Category, &start, &end, &allDay, &e.TimeZone,
			&e.Recurrence, &until, &created, &updated); err != nil {
			rows.Close()
			return nil, err
		}
		e.Description, e.Location, e.AllDay = nullStr(desc), nullStr(loc), allDay != 0
		e.Start, e.End, e.RecurrenceUntil = parseTime(start), parseTime(end), nullTime(until)
		e.CreatedAt, e.UpdatedAt = parseTime(created), parseTime(updated)
		e.Exceptions, e.Groups, e.Sections, e.Roles = []string{}, []NamedEntity{}, []NamedEntity{}, []NamedEntity{}
		index[e.ID] = len(result)
		result = append(result, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(result) == 0 {
		return result, err
	}

	filter, fargs := "", []any(nil)
	if id != "" {
		filter, fargs = ` AND a.event_id = ?`, []any{id}
	}
	rows, err = s.db.QueryContext(ctx, `SELECT a.event_id, a.kind, n.id, n.name FROM event_audiences a
		JOIN (SELECT 'group' AS kind, id, name FROM groups WHERE deleted_at IS NULL
		      UNION ALL SELECT 'section', id, name FROM sections WHERE deleted_at IS NULL
		      UNION ALL SELECT 'role', id, name FROM roles WHERE deleted_at IS NULL) n
		  ON n.kind = a.kind AND n.id = a.target_id
		WHERE a.deleted_at IS NULL`+filter+` ORDER BY n.name`, fargs...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var eventID, kind string
		var n NamedEntity
		if err := rows.Scan(&eventID, &kind, &n.ID, &n.Name); err != nil {
			rows.Close()
			return nil, err
		}
		i, ok := index[eventID]
		if !ok {
			continue
		}
		e := &result[i]
		switch kind {
		case AudienceGroup:
			e.Groups = append(e.Groups, n)
		case AudienceSection:
			e.Sections = append(e.Sections, n)
		case AudienceRole:
			e.Roles = append(e.Roles, n)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	filter = ""
	if id != "" {
		filter = ` AND event_id = ?`
	}
	rows, err = s.db.QueryContext(ctx, `SELECT event_id, occurrence_date FROM event_exceptions
		WHERE deleted_at IS NULL`+filter+` ORDER BY occurrence_date`, fargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID, date string
		if err := rows.Scan(&eventID, &date); err != nil {
			return nil, err
		}
		if i, ok := index[eventID]; ok {
			result[i].Exceptions = append(result[i].Exceptions, date)
		}
	}
	return result, rows.Err()
}

// CreateEvent inserts an event.
func (s *Store) CreateEvent(ctx context.Context, in EventInput) (Event, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		values := eventColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		if err := w.insert("events", values); err != nil {
			return err
		}
		return writeAudiences(w, id, in)
	})
	if err != nil {
		return Event{}, err
	}
	return s.GetEvent(ctx, id)
}

// UpdateEvent replaces a live event (the whole series). Exceptions are kept.
func (s *Store) UpdateEvent(ctx context.Context, id string, in EventInput) (Event, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update("events", `id = ? AND deleted_at IS NULL`, []any{id}, eventColumns(in))); err != nil {
			return err
		}
		return writeAudiences(w, id, in)
	})
	if err != nil {
		return Event{}, err
	}
	return s.GetEvent(ctx, id)
}

func eventColumns(in EventInput) row {
	allDay := 0
	if in.AllDay {
		allDay = 1
	}
	tz := in.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	var until any
	if in.RecurrenceUntil != nil && in.Recurrence != RecurNone {
		until = in.RecurrenceUntil.Format(DateLayout)
	}
	return row{
		"title": in.Title, "description": strArg(in.Description), "location": strArg(in.Location),
		"category": in.Category, "starts_at": db.FormatTime(in.Start), "ends_at": db.FormatTime(in.End),
		"all_day": allDay, "time_zone": tz, "recurrence": in.Recurrence, "recurrence_until": until,
	}
}

// writeAudiences makes the audience rows of an event exactly the given ids;
// unknown ids are rejected.
func writeAudiences(w *writer, eventID string, in EventInput) error {
	type target struct{ kind, id string }
	var want []target
	for _, l := range []struct {
		kind NamedKind
		ids  []string
	}{{Groups, in.GroupIDs}, {Sections, in.SectionIDs}, {Roles, in.RoleIDs}} {
		for _, id := range l.ids {
			var one int
			err := w.QueryRowContext(w.ctx, `SELECT 1 FROM `+l.kind.table+` WHERE id = ? AND deleted_at IS NULL`, id).Scan(&one)
			if err == sql.ErrNoRows {
				return ErrInvalidReference
			}
			if err != nil {
				return err
			}
			want = append(want, target{l.kind.audience, id})
		}
	}

	rows, err := w.QueryContext(w.ctx, `SELECT kind, target_id FROM event_audiences WHERE event_id = ? AND deleted_at IS NULL`, eventID)
	if err != nil {
		return err
	}
	var current []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.kind, &t.id); err != nil {
			rows.Close()
			return err
		}
		current = append(current, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	keep := map[target]bool{}
	for _, t := range want {
		keep[t] = true
	}
	now := db.Now()
	for _, t := range current {
		if keep[t] {
			continue
		}
		if _, err := w.update("event_audiences", `event_id = ? AND kind = ? AND target_id = ?`,
			[]any{eventID, t.kind, t.id}, row{"deleted_at": now}); err != nil {
			return err
		}
	}
	for _, t := range want {
		if err := w.upsert("event_audiences", row{"event_id": eventID, "kind": t.kind, "target_id": t.id},
			row{"deleted_at": nil}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteEvent tombstones an event with its audiences, exceptions and attendance.
func (s *Store) DeleteEvent(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update("events", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		for _, table := range []string{"event_audiences", "event_exceptions", "event_attendance"} {
			if _, err := w.update(table, `event_id = ? AND deleted_at IS NULL`, []any{id}, tombstone); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetEventException excludes (or, with excluded false, restores) the
// occurrence of a live event on the given local date.
func (s *Store) SetEventException(ctx context.Context, id string, date time.Time, excluded bool) (Event, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		var one int
		err := w.QueryRowContext(ctx, `SELECT 1 FROM events WHERE id = ? AND deleted_at IS NULL`, id).Scan(&one)
		if err != nil {
			return mapErr(err)
		}
		pk := row{"event_id": id, "occurrence_date": date.Format(DateLayout)}
		if excluded {
			return w.upsert("event_exceptions", pk, row{"deleted_at": nil})
		}
		_, err = w.update("event_exceptions", `event_id = ? AND occurrence_date = ? AND deleted_at IS NULL`,
			[]any{id, date.Format(DateLayout)}, row{"deleted_at": db.Now()})
		return err
	})
	if err != nil {
		return Event{}, err
	}
	return s.GetEvent(ctx, id)
}

// Matches reports whether the event is for the audience of the filter.
func (f AudienceFilter) Matches(e *Event) bool {
	if f == (AudienceFilter{}) || len(e.Groups)+len(e.Sections)+len(e.Roles) == 0 {
		return true
	}
	has := func(list []NamedEntity, id string) bool {
		for _, n := range list {
			if id != "" && n.ID == id {
				return true
			}
		}
		return false
	}
	return has(e.Groups, f.GroupID) || has(e.Sections, f.SectionID) || has(e.Roles, f.RoleID)
}

// maxOccurrencesPerEvent bounds the expansion of one series.
const maxOccurrencesPerEvent = 2000

// Occurrences expands the events into the occurrences that overlap the local
// days from..to (inclusive, both DateLayout dates in each event's time zone),
// ordered by start.
func Occurrences(events []Event, from, to time.Time, f AudienceFilter) []Occurrence {
	result := []Occurrence{}
	for i := range events {
		e := &events[i]
		if !f.Matches(e) {
			continue
		}
		result = append(result, expand(e, from, to)...)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].Start.Equal(result[j].Start) {
			return result[i].Start.Before(result[j].Start)
		}
		return result[i].Event.Title < result[j].Event.Title
	})
	return result
}

func eventLocation(e *Event) *time.Location {
	loc, err := time.LoadLocation(e.TimeZone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// expand returns the occurrences of one event overlapping the local days
// from..to. Each occurrence keeps the local wall-clock times of the first one.
func expand(e *Event, from, to time.Time) []Occurrence {
	loc := eventLocation(e)
	first, last := e.Start.In(loc), e.End.In(loc)
	if e.AllDay {
		// Only the dates count; the end is exclusive and at least one day.
		first = time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
		last = time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, loc)
		if !last.After(first) {
			last = first.AddDate(0, 0, 1)
		}
	}
	spanDays := daysBetween(first, last)
	windowStart := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	windowEnd := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	var until time.Time
	if e.RecurrenceUntil != nil {
		u := e.RecurrenceUntil.UTC()
		until = time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	}
	excluded := map[string]bool{}
	for _, d := range e.Exceptions {
		excluded[d] = true
	}

	at := func(k int) (time.Time, bool) {
		switch e.Recurrence {
		case RecurWeekly, RecurBiweekly:
			step := 7
			if e.Recurrence == RecurBiweekly {
				step = 14
			}
			return time.Date(first.Year(), first.Month(), first.Day()+k*step, first.Hour(), first.Minute(), first.Second(), 0, loc), true
		case RecurMonthly:
			d := time.Date(first.Year(), first.Month()+time.Month(k), first.Day(), first.Hour(), first.Minute(), first.Second(), 0, loc)
			// Months without that day (e.g. the 31st) are skipped.
			return d, d.Day() == first.Day()
		}
		return first, k == 0
	}

	// Start a little before the window so that occurrences reaching into it
	// are included.
	k := 0
	if e.Recurrence != RecurNone && windowStart.After(first) {
		switch e.Recurrence {
		case RecurWeekly:
			k = daysBetween(first, windowStart)/7 - spanDays/7 - 1
		case RecurBiweekly:
			k = daysBetween(first, windowStart)/14 - spanDays/14 - 1
		case RecurMonthly:
			k = (windowStart.Year()-first.Year())*12 + int(windowStart.Month()-first.Month()) - spanDays/28 - 1
		}
		if k < 0 {
			k = 0
		}
	}

	var result []Occurrence
	for n := 0; n < maxOccurrencesPerEvent; n, k = n+1, k+1 {
		start, ok := at(k)
		if !start.Before(windowEnd) || (!until.IsZero() && !start.Before(until)) {
			break
		}
		if !ok {
			continue
		}
		end := time.Date(start.Year(), start.Month(), start.Day()+spanDays, last.Hour(), last.Minute(), last.Second(), 0, loc)
		date := start.Format(DateLayout)
		if end.After(windowStart) && !excluded[date] {
			result = append(result, Occurrence{Event: e, Date: date, Start: start, End: end})
		}
		if e.Recurrence == RecurNone {
			break
		}
	}
	return result
}

// daysBetween returns the number of calendar days from a to b (same location).
func daysBetween(a, b time.Time) int {
	dayA := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	dayB := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(dayB.Sub(dayA).Hours() / 24)
}

// OccurrenceOn returns the occurrence of the event that starts on the given
// local date, if there is one.
func OccurrenceOn(e *Event, date time.Time) (Occurrence, bool) {
	for _, o := range expand(e, date, date) {
		if o.Date == date.Format(DateLayout) {
			return o, true
		}
	}
	return Occurrence{}, false
}
