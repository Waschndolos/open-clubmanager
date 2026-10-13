// Package ical writes iCalendar files (RFC 5545) that Google Calendar,
// Outlook and Apple Calendar can import.
package ical

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Event is one VEVENT. Timed events are written in UTC; all-day events as
// dates, with End exclusive (the day after the last day).
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Categories  []string
	Start, End  time.Time
	AllDay      bool
}

// Calendar renders a VCALENDAR with the given name and events. stamp is the
// DTSTAMP of every event.
func Calendar(name string, events []Event, stamp time.Time) string {
	var b strings.Builder
	line := func(s string) {
		b.WriteString(fold(s))
		b.WriteString("\r\n")
	}
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Open ClubManager//Events//EN")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	if name != "" {
		line("X-WR-CALNAME:" + Escape(name))
	}
	for _, e := range events {
		line("BEGIN:VEVENT")
		line("UID:" + Escape(e.UID))
		line("DTSTAMP:" + utc(stamp))
		if e.AllDay {
			line("DTSTART;VALUE=DATE:" + e.Start.Format("20060102"))
			line("DTEND;VALUE=DATE:" + e.End.Format("20060102"))
		} else {
			line("DTSTART:" + utc(e.Start))
			line("DTEND:" + utc(e.End))
		}
		line("SUMMARY:" + Escape(e.Summary))
		if e.Description != "" {
			line("DESCRIPTION:" + Escape(e.Description))
		}
		if e.Location != "" {
			line("LOCATION:" + Escape(e.Location))
		}
		if len(e.Categories) > 0 {
			escaped := make([]string, len(e.Categories))
			for i, c := range e.Categories {
				escaped[i] = Escape(c)
			}
			line("CATEGORIES:" + strings.Join(escaped, ","))
		}
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return b.String()
}

func utc(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// Escape escapes a TEXT value (RFC 5545 §3.3.11).
func Escape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`)
	return r.Replace(s)
}

// fold splits a content line into lines of at most 75 octets, continuing with
// a leading space, without splitting UTF-8 characters (RFC 5545 §3.1).
func fold(s string) string {
	if len(s) <= 75 {
		return s
	}
	var b strings.Builder
	limit := 75
	for len(s) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		b.WriteString(s[:cut])
		b.WriteString("\r\n ")
		s = s[cut:]
		limit = 74 // the leading space counts
	}
	b.WriteString(s)
	return b.String()
}
