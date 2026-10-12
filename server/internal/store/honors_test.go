package store

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func TestComputeJubilees(t *testing.T) {
	members := []jubileeMember{
		{ref: MemberRef{ID: "a", Number: 1}, entry: day(2000, 6, 15), birthday: day(1976, 7, 1)},
		{ref: MemberRef{ID: "b", Number: 2}, entry: day(2005, 6, 20)},
		{ref: MemberRef{ID: "c", Number: 3}, entry: day(2000, 2, 29)},
	}
	q := JubileeQuery{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC),
		MembershipYears: []int{25}, Ages: []int{50},
	}
	got := computeJubilees(members, map[honorKey]bool{}, q)
	// a: 25 years on 2025-06-15, 50th birthday 2026 (outside); b: 2030 (outside); c: 2025-03-01.
	if len(got) != 2 || got[0].Member.ID != "c" || !got[0].Date.Equal(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)) ||
		got[1].Member.ID != "a" || got[1].Milestone != 25 {
		t.Fatalf("unexpected jubilees: %+v", got)
	}

	// A negative correction (break in the membership) postpones the anniversary: 15 years -> entry + 20.
	members[1].adjust = -5
	q.MembershipYears = []int{15}
	if got = computeJubilees(members[1:2], nil, q); len(got) != 1 || got[0].Date.Year() != 2025 {
		t.Fatalf("adjusted jubilee: %+v", got)
	}

	// Honored stages are not suggested again.
	q.MembershipYears = []int{25}
	got = computeJubilees(members, map[honorKey]bool{{"a", HonorMembership, 25}: true}, q)
	if len(got) != 1 || got[0].Member.ID != "c" {
		t.Fatalf("honored stage suggested again: %+v", got)
	}
}
