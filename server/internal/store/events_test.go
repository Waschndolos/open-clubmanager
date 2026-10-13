package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func occDates(list []Occurrence) []string {
	out := make([]string, len(list))
	for i, o := range list {
		out[i] = o.Date
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestExpandRecurrence(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	day := func(s string) time.Time { return *date(s) }
	// Tuesday training 18:00–19:30 Berlin time, starting in summer time.
	training := Event{ID: "t", Title: "Training", TimeZone: "Europe/Berlin", Recurrence: RecurWeekly,
		Start: time.Date(2026, 10, 13, 18, 0, 0, 0, berlin), End: time.Date(2026, 10, 13, 19, 30, 0, 0, berlin),
		Exceptions: []string{"2026-10-27"}}

	got := expand(&training, day("2026-10-14"), day("2026-11-10"))
	if want := []string{"2026-10-20", "2026-11-03", "2026-11-10"}; !equal(occDates(got), want) {
		t.Fatalf("weekly = %v, want %v (27th is excluded)", occDates(got), want)
	}
	// The wall-clock time stays 18:00 across the end of daylight saving (25 Oct).
	for _, o := range got {
		if l := o.Start.In(berlin); l.Hour() != 18 || o.End.Sub(o.Start) != 90*time.Minute {
			t.Errorf("%s starts %v, lasts %v", o.Date, l, o.End.Sub(o.Start))
		}
	}
	if got[0].Start.UTC().Hour() != 16 || got[1].Start.UTC().Hour() != 17 {
		t.Errorf("UTC hours = %d, %d; want 16 (CEST) and 17 (CET)", got[0].Start.UTC().Hour(), got[1].Start.UTC().Hour())
	}

	// Far in the future the expansion starts near the window, not at the first occurrence.
	if got := expand(&training, day("2029-01-01"), day("2029-01-07")); !equal(occDates(got), []string{"2029-01-02"}) {
		t.Errorf("weekly 2029 = %v", occDates(got))
	}
	// Nothing before the first occurrence.
	if got := expand(&training, day("2026-01-01"), day("2026-10-12")); len(got) != 0 {
		t.Errorf("before start = %v", occDates(got))
	}

	biweekly := training
	biweekly.Recurrence, biweekly.Exceptions = RecurBiweekly, nil
	until := day("2026-11-24")
	biweekly.RecurrenceUntil = &until
	if got := expand(&biweekly, day("2026-10-01"), day("2026-12-31")); !equal(occDates(got),
		[]string{"2026-10-13", "2026-10-27", "2026-11-10", "2026-11-24"}) {
		t.Errorf("biweekly until = %v", occDates(got))
	}

	// Monthly on the 31st skips months without that day.
	monthly := Event{ID: "m", Title: "Vorstand", TimeZone: "Europe/Berlin", Recurrence: RecurMonthly,
		Start: time.Date(2026, 1, 31, 19, 0, 0, 0, berlin), End: time.Date(2026, 1, 31, 21, 0, 0, 0, berlin)}
	if got := expand(&monthly, day("2026-01-01"), day("2026-06-30")); !equal(occDates(got),
		[]string{"2026-01-31", "2026-03-31", "2026-05-31"}) {
		t.Errorf("monthly = %v", occDates(got))
	}

	// A multi-day all-day event overlapping the window start is included; end is exclusive.
	camp := Event{ID: "c", Title: "Camp", TimeZone: "Europe/Berlin", Recurrence: RecurNone, AllDay: true,
		Start: time.Date(2026, 7, 10, 0, 0, 0, 0, berlin), End: time.Date(2026, 7, 13, 0, 0, 0, 0, berlin)}
	got = expand(&camp, day("2026-07-12"), day("2026-07-20"))
	if len(got) != 1 || got[0].Date != "2026-07-10" || got[0].End.In(berlin).Day() != 13 {
		t.Errorf("camp = %+v", got)
	}
	if got := expand(&camp, day("2026-07-13"), day("2026-07-20")); len(got) != 0 {
		t.Errorf("camp after its end = %v", occDates(got))
	}

	if o, ok := OccurrenceOn(&training, day("2026-11-03")); !ok || o.Start.In(berlin).Hour() != 18 {
		t.Errorf("OccurrenceOn = %+v, %v", o, ok)
	}
	if _, ok := OccurrenceOn(&training, day("2026-10-27")); ok {
		t.Error("excluded occurrence found")
	}
}

func TestAudienceFilter(t *testing.T) {
	all := Event{ID: "all"}
	youth := Event{ID: "youth", Groups: []NamedEntity{{ID: "g1"}}}
	board := Event{ID: "board", Roles: []NamedEntity{{ID: "r1"}}}
	cases := []struct {
		f    AudienceFilter
		want map[string]bool
	}{
		{AudienceFilter{}, map[string]bool{"all": true, "youth": true, "board": true}},
		{AudienceFilter{GroupID: "g1"}, map[string]bool{"all": true, "youth": true}},
		{AudienceFilter{RoleID: "r1"}, map[string]bool{"all": true, "board": true}},
		{AudienceFilter{GroupID: "g2"}, map[string]bool{"all": true}},
	}
	for _, c := range cases {
		for _, e := range []*Event{&all, &youth, &board} {
			if got := c.f.Matches(e); got != c.want[e.ID] {
				t.Errorf("%+v matches %s = %v", c.f, e.ID, got)
			}
		}
	}
}

func TestEventCRUD(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	group, _ := s.CreateNamed(ctx, Groups, "Jugend")
	role, _ := s.CreateNamed(ctx, Roles, "Trainer")
	start := time.Date(2026, 10, 13, 16, 0, 0, 0, time.UTC)

	e, err := s.CreateEvent(ctx, EventInput{Title: "Jugendtraining", Category: EventTraining, Start: start,
		End: start.Add(time.Hour), TimeZone: "Europe/Berlin", Recurrence: RecurWeekly,
		GroupIDs: []string{group.ID}, RoleIDs: []string{role.ID}, Location: ptr("Strand")})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if len(e.Groups) != 1 || e.Groups[0].Name != "Jugend" || len(e.Roles) != 1 || *e.Location != "Strand" {
		t.Errorf("created = %+v", e)
	}
	if _, err := s.CreateEvent(ctx, EventInput{Title: "X", Category: EventOther, Start: start, End: start,
		Recurrence: RecurNone, GroupIDs: []string{"nope"}}); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("unknown group: %v", err)
	}

	e, err = s.SetEventException(ctx, e.ID, *date("2026-10-20"), true)
	if err != nil || len(e.Exceptions) != 1 || e.Exceptions[0] != "2026-10-20" {
		t.Fatalf("exception = %v, %v", e.Exceptions, err)
	}

	// Updating the series keeps exceptions and replaces the audience.
	e, err = s.UpdateEvent(ctx, e.ID, EventInput{Title: "Training", Category: EventTraining, Start: start,
		End: start.Add(2 * time.Hour), TimeZone: "Europe/Berlin", Recurrence: RecurWeekly, RoleIDs: []string{role.ID}})
	if err != nil || len(e.Groups) != 0 || len(e.Roles) != 1 || len(e.Exceptions) != 1 || e.Location != nil {
		t.Fatalf("updated = %+v, %v", e, err)
	}
	e, _ = s.SetEventException(ctx, e.ID, *date("2026-10-20"), false)
	if len(e.Exceptions) != 0 {
		t.Errorf("restored exception kept: %v", e.Exceptions)
	}

	// Deleting a role removes it from the audience.
	if err := s.DeleteNamed(ctx, Roles, role.ID); err != nil {
		t.Fatal(err)
	}
	if e, _ = s.GetEvent(ctx, e.ID); len(e.Roles) != 0 {
		t.Errorf("roles after deleting the role = %v", e.Roles)
	}

	list, _ := s.ListEvents(ctx)
	if occ := Occurrences(list, *date("2026-10-13"), *date("2026-10-27"), AudienceFilter{}); len(occ) != 3 {
		t.Errorf("occurrences = %v", occDates(occ))
	}
	if err := s.DeleteEvent(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEvent(ctx, e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted event: %v", err)
	}
}
