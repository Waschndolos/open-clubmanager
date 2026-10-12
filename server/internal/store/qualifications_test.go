package store

import (
	"testing"
	"time"
)

func TestCurrentQualifications(t *testing.T) {
	now := time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC)
	mk := func(id, member, typ string, exp *time.Time) Qualification {
		return Qualification{ID: id, Member: MemberRef{ID: member}, Type: QualificationType{ID: typ, Name: typ},
			IssuedDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), ExpiryDate: exp}
	}
	got := currentQualifications([]Qualification{
		mk("old", "a", "firstaid", day(2024, 1, 1)),  // superseded by the renewal below
		mk("new", "a", "firstaid", day(2026, 6, 20)), // within 30 days
		mk("exp", "b", "firstaid", day(2026, 5, 31)),
		mk("ok", "c", "licence", day(2030, 1, 1)),
		mk("forever", "d", "licence", nil),
	}, now, 30)
	want := []struct{ id, status string }{{"exp", QualExpired}, {"new", QualExpiring}, {"ok", QualValid}, {"forever", QualValid}}
	if len(got) != len(want) {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	for i, w := range want {
		if got[i].ID != w.id || got[i].Status != w.status {
			t.Errorf("entry %d = %s/%s, want %s/%s", i, got[i].ID, got[i].Status, w.id, w.status)
		}
	}
}
