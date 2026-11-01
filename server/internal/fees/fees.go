// Package fees computes membership fees from fee models (Beitragsmodelle):
// the billing periods of an interval, the model that applies to a member by
// age, and the pro-rata amount when a member joins or leaves during a period.
// All dates are calendar dates in UTC.
package fees

import (
	"fmt"
	"time"
)

// Interval is how often a fee model is charged. Periods are aligned to the
// calendar year (January, July, quarters, months).
type Interval string

const (
	Yearly     Interval = "yearly"
	HalfYearly Interval = "half_yearly"
	Quarterly  Interval = "quarterly"
	Monthly    Interval = "monthly"
)

// Months returns the length of a period in months, or 0 for an unknown interval.
func (i Interval) Months() int {
	switch i {
	case Yearly:
		return 12
	case HalfYearly:
		return 6
	case Quarterly:
		return 3
	case Monthly:
		return 1
	}
	return 0
}

// Valid reports whether i is a known interval.
func (i Interval) Valid() bool { return i.Months() > 0 }

// Model is a fee class with its amount per period. MinAge and MaxAge are
// inclusive; a model with either limit is picked automatically by the age on
// 1 January of the fee year (YearStart).
type Model struct {
	ID          string
	Name        string
	AmountCents int64
	Interval    Interval
	MinAge      *int
	MaxAge      *int
	// ProRata charges only the months of a period in which the member
	// belongs to the club.
	ProRata bool
}

// HasAgeLimit reports whether the model takes part in the selection by age.
func (m Model) HasAgeLimit() bool { return m.MinAge != nil || m.MaxAge != nil }

// FitsAge reports whether age lies within the model's age limits.
func (m Model) FitsAge(age int) bool {
	return (m.MinAge == nil || age >= *m.MinAge) && (m.MaxAge == nil || age <= *m.MaxAge)
}

// span is the width of the age range, used to prefer the most specific model.
func (m Model) span() int {
	lo, hi := 0, 200
	if m.MinAge != nil {
		lo = *m.MinAge
	}
	if m.MaxAge != nil {
		hi = *m.MaxAge
	}
	return hi - lo
}

// SelectByAge returns the model with an age limit that fits age. When
// several fit, the one with the narrowest age range wins, then the first by
// name, so the choice does not depend on the order of models.
func SelectByAge(models []Model, age int) (Model, bool) {
	var best Model
	found := false
	for _, m := range models {
		if !m.HasAgeLimit() || !m.FitsAge(age) {
			continue
		}
		if !found || m.span() < best.span() || (m.span() == best.span() && m.Name < best.Name) {
			best, found = m, true
		}
	}
	return best, found
}

// Age returns the age in whole years on the given day.
func Age(birthday, on time.Time) int {
	age := on.Year() - birthday.Year()
	if on.Month() < birthday.Month() || (on.Month() == birthday.Month() && on.Day() < birthday.Day()) {
		age--
	}
	return age
}

// Period is a billing period from Start to End (both inclusive, first and
// last day of a month).
type Period struct {
	Start time.Time
	End   time.Time
}

// Overlaps reports whether two periods share a day.
func (p Period) Overlaps(o Period) bool { return !p.Start.After(o.End) && !o.Start.After(p.End) }

// Label names the period language-neutrally: "2027", "2027-H1", "2027-Q3"
// or "2027-05".
func (p Period) Label(i Interval) string {
	y, m := p.Start.Year(), int(p.Start.Month())
	switch i {
	case Yearly:
		return fmt.Sprintf("%d", y)
	case HalfYearly:
		return fmt.Sprintf("%d-H%d", y, (m-1)/6+1)
	case Quarterly:
		return fmt.Sprintf("%d-Q%d", y, (m-1)/3+1)
	}
	return fmt.Sprintf("%d-%02d", y, m)
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// YearStart returns 1 January of t's year, the day whose age decides the
// fee model for the whole year.
func YearStart(t time.Time) time.Time { return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC) }

// MonthEnd returns the last day of t's month.
func MonthEnd(t time.Time) time.Time { return monthStart(t).AddDate(0, 1, -1) }

// Periods returns the periods of interval i that start between from and to
// (inclusive; only year and month count), in order. A period may end after to.
func Periods(i Interval, from, to time.Time) []Period {
	n := i.Months()
	if n == 0 {
		return nil
	}
	from, to = monthStart(from), monthStart(to)
	var out []Period
	for start := time.Date(from.Year(), 1, 1, 0, 0, 0, 0, time.UTC); !start.After(to); start = start.AddDate(0, n, 0) {
		if start.Before(from) {
			continue
		}
		out = append(out, Period{Start: start, End: start.AddDate(0, n, -1)})
	}
	return out
}

// Membership is a member's entry and exit date. Without an entry date the
// member belongs to the club from the beginning; on the exit date the member
// no longer belongs to it.
type Membership struct {
	Entry *time.Time
	Exit  *time.Time
}

// between reports whether the member belongs to the club during some part
// of [start, end].
func (ms Membership) between(start, end time.Time) bool {
	return (ms.Entry == nil || !ms.Entry.After(end)) && (ms.Exit == nil || ms.Exit.After(start))
}

// ActiveIn reports whether the member belongs to the club on any day of p.
func (ms Membership) ActiveIn(p Period) bool { return ms.between(p.Start, p.End) }

// ActiveMonths counts the months of p in which the member belongs to the
// club on at least one day.
func (ms Membership) ActiveMonths(p Period) int {
	n := 0
	for m := p.Start; !m.After(p.End); m = m.AddDate(0, 1, 0) {
		if ms.between(m, MonthEnd(m)) {
			n++
		}
	}
	return n
}

// Amount returns what a member owes for period p of model m: the full amount,
// or with ProRata the share of the months the member belongs to the club,
// rounded half up to the cent.
func Amount(m Model, p Period, ms Membership) int64 {
	if !m.ProRata {
		return m.AmountCents
	}
	total := int64(m.Interval.Months())
	if total == 0 {
		return 0
	}
	active := int64(ms.ActiveMonths(p))
	if active >= total {
		return m.AmountCents
	}
	return (m.AmountCents*active + total/2) / total
}
