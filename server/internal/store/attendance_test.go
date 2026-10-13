package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAttendance(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	youth, _ := s.CreateNamed(ctx, Groups, "Jugend")
	adults, _ := s.CreateNamed(ctx, Groups, "Erwachsene")
	member := func(n int, first string, email *string, exit *time.Time, groups ...string) Member {
		m, err := s.CreateMember(ctx, MemberInput{Number: n, FirstName: first, LastName: "Kite", ExitDate: exit,
			Contact: MemberContact{Email: email}, GroupIDs: &groups})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	member(1, "Coach", ptr("coach@example.org"), nil, youth.ID)
	ida := member(2, "Ida", nil, nil, youth.ID)
	ole := member(3, "Ole", nil, nil, youth.ID)
	gone := member(4, "Gone", nil, date("2026-09-30"), youth.ID)
	guest := member(5, "Guest", nil, nil, adults.ID)

	start := time.Date(2026, 10, 13, 15, 30, 0, 0, time.UTC)
	training, err := s.CreateEvent(ctx, EventInput{Title: "Jugendtraining", Category: EventTraining, Start: start,
		End: start.Add(90 * time.Minute), TimeZone: "Europe/Berlin", Recurrence: RecurWeekly, GroupIDs: []string{youth.ID}})
	if err != nil {
		t.Fatal(err)
	}
	party, _ := s.CreateEvent(ctx, EventInput{Title: "Fest", Category: EventFestival, Start: start, End: start.Add(time.Hour),
		TimeZone: "Europe/Berlin", Recurrence: RecurNone})

	sheet, err := s.AttendanceSheet(ctx, training.ID, *date("2026-10-13"))
	if err != nil {
		t.Fatal(err)
	}
	names := func(sh AttendanceSheet) []string {
		var out []string
		for _, m := range sh.Members {
			out = append(out, m.FirstName)
		}
		return out
	}
	// The youth group without the member who left before; sorted by name.
	if got := names(sheet); !equal(got, []string{"Coach", "Ida", "Ole"}) {
		t.Errorf("sheet = %v", got)
	}
	if _, err := s.AttendanceSheet(ctx, training.ID, *date("2026-10-14")); !errors.Is(err, ErrNotFound) {
		t.Errorf("sheet on a day without occurrence: %v", err)
	}

	present, excused := AttendancePresent, AttendanceExcused
	day := *date("2026-10-13")
	if err := s.SetAttendance(ctx, training.ID, day, []AttendanceEntry{
		{MemberID: ida.ID, Status: &present}, {MemberID: ole.ID, Status: &excused}, {MemberID: guest.ID, Status: &present},
	}); err != nil {
		t.Fatal(err)
	}
	// Resending is idempotent; a nil status clears the entry.
	if err := s.SetAttendance(ctx, training.ID, day, []AttendanceEntry{
		{MemberID: ida.ID, Status: &present}, {MemberID: ole.ID, Status: nil},
	}); err != nil {
		t.Fatal(err)
	}
	sheet, _ = s.AttendanceSheet(ctx, training.ID, day)
	if got := names(sheet); !equal(got, []string{"Coach", "Guest", "Ida", "Ole"}) {
		t.Errorf("sheet with guest = %v", got)
	}
	for _, m := range sheet.Members {
		switch m.FirstName {
		case "Guest":
			if m.InAudience || m.Status == nil || *m.Status != present {
				t.Errorf("guest = %+v", m)
			}
		case "Ole":
			if m.Status != nil {
				t.Errorf("cleared entry kept: %+v", m)
			}
		}
	}
	if err := s.SetAttendance(ctx, training.ID, day, []AttendanceEntry{{MemberID: gone.ID, Status: &present}}); err != nil {
		t.Errorf("recording a member that has left: %v", err)
	}
	_ = s.SetAttendance(ctx, training.ID, *date("2026-10-20"), []AttendanceEntry{{MemberID: ida.ID, Status: &excused}})

	// Club-wide events list all current members.
	sheet, _ = s.AttendanceSheet(ctx, party.ID, day)
	if len(sheet.Members) != 4 {
		t.Errorf("club-wide sheet = %v", names(sheet))
	}

	// A trainer's scope covers the youth group's events, not club-wide ones.
	sc, err := s.TrainerScope(ctx, "COACH@example.org")
	if err != nil || !sc.Allows(&training) || sc.Allows(&party) {
		t.Errorf("trainer scope = %+v, %v", sc, err)
	}
	if sc, _ := s.TrainerScope(ctx, "unknown@example.org"); sc.Allows(&training) {
		t.Error("unlinked login sees the youth training")
	}

	from, to := *date("2026-10-01"), *date("2026-10-31")
	records, err := s.AttendanceRecords(ctx, from, to, AttendanceScope{All: true}, AttendanceFilter{GroupID: youth.ID})
	if err != nil || len(records) != 4 {
		t.Fatalf("records = %+v, %v", records, err)
	}
	stats := SummarizeByMember(records)
	if len(stats) != 3 || stats[2].Member.FirstName != "Ida" || stats[2].Present != 1 || stats[2].Excused != 1 || stats[2].Rate() != 0.5 {
		t.Errorf("stats = %+v", stats)
	}
	counts, _ := s.AttendanceOccurrences(ctx, from, to, sc, AttendanceFilter{})
	if len(counts) != 3 || counts[0].Present != 3 {
		t.Errorf("trainer occurrences = %+v", counts)
	}

	// Excluding an occurrence hides its records; deleting the event removes them.
	if _, err := s.SetEventException(ctx, training.ID, *date("2026-10-20"), true); err != nil {
		t.Fatal(err)
	}
	if records, _ := s.AttendanceRecords(ctx, from, to, AttendanceScope{All: true}, AttendanceFilter{}); len(records) != 3 {
		t.Errorf("records after excluding 20 Oct = %d, want 3", len(records))
	}
	if err := s.DeleteEvent(ctx, training.ID); err != nil {
		t.Fatal(err)
	}
	var live int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM event_attendance WHERE deleted_at IS NULL`).Scan(&live)
	if live != 0 {
		t.Errorf("attendance rows of the deleted event = %d", live)
	}
}
