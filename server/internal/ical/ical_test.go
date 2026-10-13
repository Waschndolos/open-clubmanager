package ical

import (
	"strings"
	"testing"
	"time"
)

func TestCalendar(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	start := time.Date(2026, 10, 13, 18, 0, 0, 0, berlin)
	out := Calendar("Kiteclub", []Event{
		{UID: "a-20261013@x", Summary: "Training; Jugend, Gruppe A", Start: start, End: start.Add(90 * time.Minute),
			Description: "Zeile 1\nZeile 2", Location: "Strand", Categories: []string{"Training"}},
		{UID: "b-20261224@x", Summary: "Weihnachtsfeier", AllDay: true,
			Start: time.Date(2026, 12, 24, 0, 0, 0, 0, berlin), End: time.Date(2026, 12, 25, 0, 0, 0, 0, berlin)},
	}, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n", "VERSION:2.0\r\n", "X-WR-CALNAME:Kiteclub\r\n",
		"DTSTART:20261013T160000Z\r\n", "DTEND:20261013T173000Z\r\n", // CEST is UTC+2
		`SUMMARY:Training\; Jugend\, Gruppe A` + "\r\n", `DESCRIPTION:Zeile 1\nZeile 2` + "\r\n",
		"DTSTART;VALUE=DATE:20261224\r\n", "DTEND;VALUE=DATE:20261225\r\n", "DTSTAMP:20261001T000000Z\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Count(out, "BEGIN:VEVENT") != 2 {
		t.Errorf("want 2 events:\n%s", out)
	}
}

func TestFold(t *testing.T) {
	long := "DESCRIPTION:" + strings.Repeat("Übung ", 40)
	folded := fold(long)
	for i, l := range strings.Split(folded, "\r\n") {
		if len(l) > 75 {
			t.Errorf("line %d has %d octets", i, len(l))
		}
		if i > 0 && !strings.HasPrefix(l, " ") {
			t.Errorf("continuation line %d does not start with a space", i)
		}
	}
	if strings.ReplaceAll(folded, "\r\n ", "") != long {
		t.Error("unfolding does not restore the line")
	}
}
