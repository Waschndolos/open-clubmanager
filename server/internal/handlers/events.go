package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/ical"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
	"github.com/Waschndolos/open-clubmanager/server/internal/store"
)

const (
	maxEventRangeDays  = 400
	maxExportRangeDays = 1100
	// Default export range: from 30 days ago up to one year ahead.
	exportPastDays   = 30
	exportFutureDays = 365
)

func toAPIEvent(e store.Event) openapi.Event {
	out := openapi.Event{
		Id: e.ID, Title: e.Title, Description: e.Description, Location: e.Location,
		Category: openapi.EventCategory(e.Category), Start: e.Start, End: e.End, AllDay: e.AllDay,
		TimeZone: e.TimeZone, Recurrence: openapi.EventRecurrence(e.Recurrence),
		Exceptions: make([]openapi_types.Date, 0, len(e.Exceptions)),
		Groups:     toAPINamedList(e.Groups), Sections: toAPINamedList(e.Sections), Roles: toAPINamedList(e.Roles),
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
	if e.RecurrenceUntil != nil {
		out.RecurrenceUntil = &openapi_types.Date{Time: *e.RecurrenceUntil}
	}
	for _, d := range e.Exceptions {
		if t, err := time.Parse(store.DateLayout, d); err == nil {
			out.Exceptions = append(out.Exceptions, openapi_types.Date{Time: t})
		}
	}
	return out
}

func toAPIOccurrence(o store.Occurrence) openapi.EventOccurrence {
	e := o.Event
	date, _ := time.Parse(store.DateLayout, o.Date)
	return openapi.EventOccurrence{
		EventId: e.ID, OccurrenceDate: openapi_types.Date{Time: date}, Title: e.Title, Description: e.Description,
		Location: e.Location, Category: openapi.EventCategory(e.Category), Start: o.Start, End: o.End,
		AllDay: e.AllDay, Recurring: e.Recurrence != store.RecurNone,
		Groups: toAPINamedList(e.Groups), Sections: toAPINamedList(e.Sections), Roles: toAPINamedList(e.Roles),
	}
}

// decodeEvent reads and validates an EventInput.
func decodeEvent(w http.ResponseWriter, r *http.Request) (store.EventInput, bool) {
	var req openapi.EventInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		badRequest(w, "invalid request body")
		return store.EventInput{}, false
	}
	in := store.EventInput{
		Title: strings.TrimSpace(req.Title), Category: string(req.Category), Start: req.Start, End: req.End,
		Recurrence: store.RecurNone, TimeZone: "UTC",
	}
	var valid bool
	if in.Description, valid = trimmed(req.Description, 5000); !valid {
		badRequest(w, "description is too long")
		return in, false
	}
	if in.Location, valid = trimmed(req.Location, 200); !valid {
		badRequest(w, "location is too long")
		return in, false
	}
	if req.AllDay != nil {
		in.AllDay = *req.AllDay
	}
	if req.Recurrence != nil {
		in.Recurrence = string(*req.Recurrence)
	}
	if req.TimeZone != nil && *req.TimeZone != "" {
		in.TimeZone = *req.TimeZone
	}
	if req.RecurrenceUntil != nil {
		u := req.RecurrenceUntil.Time
		in.RecurrenceUntil = &u
	}
	for _, l := range []struct {
		dst *[]string
		src *[]string
	}{{&in.GroupIDs, req.GroupIds}, {&in.SectionIDs, req.SectionIds}, {&in.RoleIDs, req.RoleIds}} {
		if l.src != nil {
			*l.dst = *l.src
		}
	}

	switch {
	case in.Title == "" || len(in.Title) > 200:
		badRequest(w, "title is required (at most 200 characters)")
	case !store.ValidEventCategory(in.Category):
		badRequest(w, "category must be training, festival, meeting, competition or other")
	case !store.ValidEventRecurrence(in.Recurrence):
		badRequest(w, "recurrence must be none, weekly, biweekly or monthly")
	case in.Start.IsZero() || in.End.Before(in.Start):
		badRequest(w, "end must not be before start")
	case in.End.Sub(in.Start) > 366*24*time.Hour:
		badRequest(w, "an event lasts at most one year")
	case in.RecurrenceUntil != nil && in.RecurrenceUntil.Before(dateOf(in.Start)):
		badRequest(w, "recurrenceUntil must not be before the start date")
	default:
		if _, err := time.LoadLocation(in.TimeZone); err != nil {
			badRequest(w, "timeZone must be an IANA time zone such as Europe/Berlin")
			return in, false
		}
		return in, true
	}
	return in, false
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func audienceFilter(group, section, role *string) store.AudienceFilter {
	var f store.AudienceFilter
	if group != nil {
		f.GroupID = *group
	}
	if section != nil {
		f.SectionID = *section
	}
	if role != nil {
		f.RoleID = *role
	}
	return f
}

// ListEvents: GET /api/v2/events.
func (a *API) ListEvents(w http.ResponseWriter, r *http.Request, params openapi.ListEventsParams) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "events", db.ScopeMembersCore) {
		return
	}
	from, to := params.From.Time, params.To.Time
	if to.Before(from) || to.Sub(from) > maxEventRangeDays*24*time.Hour {
		badRequest(w, "to must be after from and at most 400 days later")
		return
	}
	events, err := st.ListEvents(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list events")
		return
	}
	occ := store.Occurrences(events, from, to, audienceFilter(params.GroupId, params.SectionId, params.RoleId))
	out := make([]openapi.EventOccurrence, 0, len(occ))
	for _, o := range occ {
		out = append(out, toAPIOccurrence(o))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateEvent: POST /api/v2/events.
func (a *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "events", db.ScopeMembersCore) {
		return
	}
	in, ok := decodeEvent(w, r)
	if !ok {
		return
	}
	e, err := st.CreateEvent(r.Context(), in)
	if err != nil {
		writeStoreError(w, err, "", "failed to create event")
		return
	}
	st.Audit(r.Context(), "CREATE", "Event", e.ID, currentUserID(r))
	writeJSON(w, http.StatusCreated, toAPIEvent(e))
}

// GetEvent: GET /api/v2/events/{id}.
func (a *API) GetEvent(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "events", db.ScopeMembersCore) {
		return
	}
	e, err := st.GetEvent(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to load event")
		return
	}
	writeJSON(w, http.StatusOK, toAPIEvent(e))
}

// UpdateEvent: PUT /api/v2/events/{id}.
func (a *API) UpdateEvent(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "events", db.ScopeMembersCore) {
		return
	}
	in, ok := decodeEvent(w, r)
	if !ok {
		return
	}
	e, err := st.UpdateEvent(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to update event")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Event", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIEvent(e))
}

// DeleteEvent: DELETE /api/v2/events/{id}.
func (a *API) DeleteEvent(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "events", db.ScopeMembersCore) {
		return
	}
	if err := st.DeleteEvent(r.Context(), id); err != nil {
		writeStoreError(w, err, "event not found", "failed to delete event")
		return
	}
	st.Audit(r.Context(), "DELETE", "Event", id, currentUserID(r))
	w.WriteHeader(http.StatusNoContent)
}

// AddEventException: POST /api/v2/events/{id}/exceptions.
func (a *API) AddEventException(w http.ResponseWriter, r *http.Request, id string) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "events", db.ScopeMembersCore) {
		return
	}
	var req openapi.EventExceptionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil || req.Date.IsZero() {
		badRequest(w, "date (YYYY-MM-DD) is required")
		return
	}
	e, err := st.SetEventException(r.Context(), id, req.Date.Time, true)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to exclude occurrence")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Event", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIEvent(e))
}

// RemoveEventException: DELETE /api/v2/events/{id}/exceptions/{date}.
func (a *API) RemoveEventException(w http.ResponseWriter, r *http.Request, id string, date openapi_types.Date) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, true, "events", db.ScopeMembersCore) {
		return
	}
	e, err := st.SetEventException(r.Context(), id, date.Time, false)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to restore occurrence")
		return
	}
	st.Audit(r.Context(), "UPDATE", "Event", id, currentUserID(r))
	writeJSON(w, http.StatusOK, toAPIEvent(e))
}

// ExportEventIcs: GET /api/v2/events/{id}/ics.
func (a *API) ExportEventIcs(w http.ResponseWriter, r *http.Request, id string, params openapi.ExportEventIcsParams) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "events", db.ScopeMembersCore) {
		return
	}
	e, err := st.GetEvent(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "event not found", "failed to load event")
		return
	}
	var occ []store.Occurrence
	if params.Date != nil {
		o, found := store.OccurrenceOn(&e, params.Date.Time)
		if !found {
			writeJSON(w, http.StatusNotFound, openapi.ErrorResponse{Error: "the event has no occurrence on this date"})
			return
		}
		occ = []store.Occurrence{o}
	} else {
		from, to := defaultExportRange(time.Now())
		if e.Recurrence == store.RecurNone {
			from, to = e.Start.AddDate(0, 0, -1), e.End.AddDate(0, 0, 1)
		}
		occ = store.Occurrences([]store.Event{e}, from, to, store.AudienceFilter{})
	}
	writeICS(w, r, slug(e.Title), e.Title, occ)
}

// ExportCalendarIcs: GET /api/v2/calendar.ics.
func (a *API) ExportCalendarIcs(w http.ResponseWriter, r *http.Request, params openapi.ExportCalendarIcsParams) {
	st, ok := a.requireStore(w)
	if !ok || !a.scopeAccess(w, r, false, "events", db.ScopeMembersCore) {
		return
	}
	from, to := defaultExportRange(time.Now())
	if params.From != nil {
		from = params.From.Time
	}
	if params.To != nil {
		to = params.To.Time
	}
	if to.Before(from) || to.Sub(from) > maxExportRangeDays*24*time.Hour {
		badRequest(w, "to must be after from and at most three years later")
		return
	}
	events, err := st.ListEvents(r.Context())
	if err != nil {
		writeStoreError(w, err, "", "failed to list events")
		return
	}
	occ := store.Occurrences(events, from, to, audienceFilter(params.GroupId, params.SectionId, params.RoleId))
	writeICS(w, r, "calendar", "Open ClubManager", occ)
}

func defaultExportRange(now time.Time) (time.Time, time.Time) {
	return now.AddDate(0, 0, -exportPastDays), now.AddDate(0, 0, exportFutureDays)
}

// categoryNames are the CATEGORIES written to .ics files, by language.
var categoryNames = map[string]map[string]string{
	"de": {store.EventTraining: "Training", store.EventFestival: "Fest", store.EventMeeting: "Versammlung",
		store.EventCompetition: "Wettkampf", store.EventOther: "Sonstiges"},
	"fr": {store.EventTraining: "Entraînement", store.EventFestival: "Fête", store.EventMeeting: "Assemblée",
		store.EventCompetition: "Compétition", store.EventOther: "Autre"},
	"en": {store.EventTraining: "Training", store.EventFestival: "Festival", store.EventMeeting: "Meeting",
		store.EventCompetition: "Competition", store.EventOther: "Other"},
}

func writeICS(w http.ResponseWriter, r *http.Request, filename, name string, occ []store.Occurrence) {
	lang := "en"
	if al := strings.ToLower(r.Header.Get("Accept-Language")); len(al) >= 2 {
		if _, ok := categoryNames[al[:2]]; ok {
			lang = al[:2]
		}
	}
	events := make([]ical.Event, 0, len(occ))
	for _, o := range occ {
		e := o.Event
		ev := ical.Event{
			UID:     e.ID + "-" + strings.ReplaceAll(o.Date, "-", "") + "@open-clubmanager",
			Summary: e.Title, Start: o.Start, End: o.End, AllDay: e.AllDay,
			Categories: []string{categoryNames[lang][e.Category]},
		}
		if e.Description != nil {
			ev.Description = *e.Description
		}
		if e.Location != nil {
			ev.Location = *e.Location
		}
		events = append(events, ev)
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.ics"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(ical.Calendar(name, events, time.Now())))
}

// slug turns a title into a safe ASCII file name.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case r == 'ä':
			b.WriteString("ae")
		case r == 'ö':
			b.WriteString("oe")
		case r == 'ü':
			b.WriteString("ue")
		case r == 'ß':
			b.WriteString("ss")
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" || len(out) > 60 {
		if len(out) > 60 {
			return strings.Trim(out[:60], "-")
		}
		return "event"
	}
	return out
}
