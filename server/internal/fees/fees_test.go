package fees

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func dayPtr(s string) *time.Time {
	t := day(s)
	return &t
}

func intPtr(v int) *int { return &v }

func TestAge(t *testing.T) {
	for _, c := range []struct {
		birthday, on string
		want         int
	}{
		{"2009-01-01", "2027-01-01", 18},
		{"2009-01-02", "2027-01-01", 17},
		{"2008-12-31", "2027-01-01", 18},
		{"2008-02-29", "2027-02-28", 18},
		{"2008-02-29", "2026-02-28", 17},
		{"2008-02-29", "2026-03-01", 18},
	} {
		if got := Age(day(c.birthday), day(c.on)); got != c.want {
			t.Errorf("Age(%s, %s) = %d, want %d", c.birthday, c.on, got, c.want)
		}
	}
	// The fee year counts the age on 1 January: 18 in June means youth
	// until the end of the year.
	if got := Age(day("2009-06-15"), YearStart(day("2027-09-01"))); got != 17 {
		t.Errorf("age for the fee year 2027 = %d", got)
	}
}

func TestSelectByAge(t *testing.T) {
	youth := Model{ID: "youth", Name: "Jugend", MaxAge: intPtr(17)}
	adult := Model{ID: "adult", Name: "Erwachsene", MinAge: intPtr(18)}
	student := Model{ID: "student", Name: "Studierende", MinAge: intPtr(18), MaxAge: intPtr(27)}
	passive := Model{ID: "passive", Name: "Passiv"}
	models := []Model{passive, adult, student, youth}

	for _, c := range []struct {
		age  int
		want string
	}{
		{0, "youth"}, {17, "youth"}, {18, "student"}, {27, "student"}, {28, "adult"}, {90, "adult"},
	} {
		got, ok := SelectByAge(models, c.age)
		if !ok || got.ID != c.want {
			t.Errorf("age %d: got %q (%v), want %s", c.age, got.ID, ok, c.want)
		}
	}
	if _, ok := SelectByAge([]Model{passive, adult}, 10); ok {
		t.Error("a model without age limit was picked by age")
	}
	// Equal ranges: the name decides, not the order.
	a := Model{ID: "a", Name: "A", MinAge: intPtr(18)}
	b := Model{ID: "b", Name: "B", MinAge: intPtr(18)}
	for _, list := range [][]Model{{a, b}, {b, a}} {
		if got, _ := SelectByAge(list, 30); got.ID != "a" {
			t.Errorf("tie picked %s", got.ID)
		}
	}
}

func TestPeriods(t *testing.T) {
	for _, c := range []struct {
		interval Interval
		from, to string
		want     []string
	}{
		{Yearly, "2027-01-01", "2027-12-31", []string{"2027"}},
		{Yearly, "2027-02-01", "2027-12-31", nil},
		{Yearly, "2027-01-01", "2028-01-31", []string{"2027", "2028"}},
		{HalfYearly, "2027-01-01", "2027-12-31", []string{"2027-H1", "2027-H2"}},
		{HalfYearly, "2027-03-01", "2027-07-31", []string{"2027-H2"}},
		{Quarterly, "2027-04-15", "2027-10-02", []string{"2027-Q2", "2027-Q3", "2027-Q4"}},
		{Monthly, "2027-11-01", "2028-02-01", []string{"2027-11", "2027-12", "2028-01", "2028-02"}},
		{Interval("weekly"), "2027-01-01", "2027-12-31", nil},
	} {
		var got []string
		for _, p := range Periods(c.interval, day(c.from), day(c.to)) {
			got = append(got, p.Label(c.interval))
		}
		if len(got) != len(c.want) {
			t.Errorf("%s %s–%s: got %v, want %v", c.interval, c.from, c.to, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s %s–%s: got %v, want %v", c.interval, c.from, c.to, got, c.want)
				break
			}
		}
	}
	p := Periods(Quarterly, day("2027-01-01"), day("2027-01-01"))[0]
	if !p.Start.Equal(day("2027-01-01")) || !p.End.Equal(day("2027-03-31")) {
		t.Errorf("Q1 = %v – %v", p.Start, p.End)
	}
	if !p.Overlaps(Period{day("2027-03-31"), day("2027-12-31")}) || p.Overlaps(Period{day("2027-04-01"), day("2027-06-30")}) {
		t.Error("Overlaps is wrong at the boundaries")
	}
}

func TestProRata(t *testing.T) {
	year := Period{Start: day("2027-01-01"), End: day("2027-12-31")}
	model := Model{AmountCents: 12000, Interval: Yearly, ProRata: true}
	for _, c := range []struct {
		name         string
		ms           Membership
		months       int
		amountCents  int64
		activeInYear bool
	}{
		{"whole year", Membership{}, 12, 12000, true},
		{"joined before", Membership{Entry: dayPtr("2020-05-01")}, 12, 12000, true},
		{"joins on 1 April", Membership{Entry: dayPtr("2027-04-01")}, 9, 9000, true},
		{"joins on 30 April", Membership{Entry: dayPtr("2027-04-30")}, 9, 9000, true},
		{"joins on 31 December", Membership{Entry: dayPtr("2027-12-31")}, 1, 1000, true},
		{"leaves on 30 June", Membership{Exit: dayPtr("2027-06-30")}, 6, 6000, true},
		{"leaves on 1 July", Membership{Exit: dayPtr("2027-07-01")}, 6, 6000, true},
		{"leaves on 2 July", Membership{Exit: dayPtr("2027-07-02")}, 7, 7000, true},
		{"leaves on 1 January", Membership{Exit: dayPtr("2027-01-01")}, 0, 0, false},
		{"left before", Membership{Exit: dayPtr("2026-10-01")}, 0, 0, false},
		{"joins later", Membership{Entry: dayPtr("2028-01-01")}, 0, 0, false},
		{"March to August", Membership{Entry: dayPtr("2027-03-15"), Exit: dayPtr("2027-08-20")}, 6, 6000, true},
	} {
		if got := c.ms.ActiveMonths(year); got != c.months {
			t.Errorf("%s: %d active months, want %d", c.name, got, c.months)
		}
		if got := c.ms.ActiveIn(year); got != c.activeInYear {
			t.Errorf("%s: ActiveIn = %v", c.name, got)
		}
		if c.activeInYear {
			if got := Amount(model, year, c.ms); got != c.amountCents {
				t.Errorf("%s: amount %d, want %d", c.name, got, c.amountCents)
			}
		}
	}

	// Rounding half up to the cent: 100.00 for 7 of 12 months = 58.333…
	odd := Model{AmountCents: 10000, Interval: Yearly, ProRata: true}
	if got := Amount(odd, year, Membership{Entry: dayPtr("2027-06-01")}); got != 5833 {
		t.Errorf("7/12 of 100.00 = %d", got)
	}
	if got := Amount(Model{AmountCents: 1000, Interval: Yearly, ProRata: true}, year, Membership{Entry: dayPtr("2027-07-01")}); got != 500 {
		t.Errorf("6/12 of 10.00 = %d", got)
	}
	// Without pro rata the full amount is due as long as the member belongs
	// to the club in the period.
	full := Model{AmountCents: 12000, Interval: Yearly}
	if got := Amount(full, year, Membership{Entry: dayPtr("2027-11-01")}); got != 12000 {
		t.Errorf("without pro rata: %d", got)
	}
	// A quarter: joining in the second month leaves two of three months.
	q := Model{AmountCents: 3000, Interval: Quarterly, ProRata: true}
	if got := Amount(q, Period{day("2027-04-01"), day("2027-06-30")}, Membership{Entry: dayPtr("2027-05-10")}); got != 2000 {
		t.Errorf("quarter: %d", got)
	}
}
