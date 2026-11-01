package reconcile

import (
	"slices"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	due := func(month int) time.Time { return time.Date(2026, time.Month(month), 1, 0, 0, 0, 0, time.UTC) }
	members := []Member{
		{ID: "petersen", Number: 1001, FirstName: "Jannik", LastName: "Petersen", IBAN: "DE89370400440532013000"},
		{ID: "andresen", Number: 1007, FirstName: "Mats", LastName: "Andresen", IBAN: "DE75 5121 0800 1245 1261 99"},
		{ID: "nele", Number: 1010, FirstName: "Nele", LastName: "Kruse", HouseholdID: "kruse"},
		{ID: "paul", Number: 1011, FirstName: "Paul", LastName: "Kruse", HouseholdID: "kruse"},
		{ID: "albers", Number: 1012, FirstName: "Tjark", LastName: "Albers", AccountHolder: "Familie Albers"},
		{ID: "wendt", Number: 1013, FirstName: "Ida", LastName: "Wendt"},
		{ID: "moeller", Number: 1015, FirstName: "Greta", LastName: "Möller"},
		{ID: "lorenzen", Number: 1006, FirstName: "Sophie", LastName: "Lorenzen", IBAN: "DE02100500000054540402"},
		{ID: "small", Number: 12, FirstName: "Kai", LastName: "Lund"},
	}
	open := []Fee{
		{ID: "f-andresen-2025", MemberID: "andresen", AmountCents: 4000, DueDate: due(1).AddDate(-1, 0, 0)},
		{ID: "f-andresen", MemberID: "andresen", AmountCents: 18000, DueDate: due(3)},
		{ID: "f-nele", MemberID: "nele", AmountCents: 9000, DueDate: due(3)},
		{ID: "f-paul", MemberID: "paul", AmountCents: 12000, DueDate: due(3)},
		{ID: "f-albers", MemberID: "albers", AmountCents: 9000, DueDate: due(3)},
		{ID: "f-wendt", MemberID: "wendt", AmountCents: 18000, DueDate: due(3)},
		{ID: "f-moeller", MemberID: "moeller", AmountCents: 18000, DueDate: due(3)},
		{ID: "f-small", MemberID: "small", AmountCents: 5000, DueDate: due(3)},
	}

	tests := []struct {
		name        string
		entry       Entry
		fees        []Fee
		confidence  Confidence // "" = no suggestion
		allocations []Allocation
		reasons     []string
	}{
		{
			name:        "member number and amount",
			entry:       Entry{AmountCents: 18000, Text: "Mitgliedsbeitrag 2026 Mitglied 1013", Name: "Ida Wendt"},
			confidence:  Certain,
			allocations: []Allocation{{"f-wendt", 18000}},
			reasons:     []string{ReasonMemberNumber, ReasonAmount},
		},
		{
			name:        "keyword with leading zeros",
			entry:       Entry{AmountCents: 5000, Text: "Beitrag Mitgl.-Nr. 0012"},
			confidence:  Certain,
			allocations: []Allocation{{"f-small", 5000}},
		},
		{
			name:       "short bare number is not a member number",
			entry:      Entry{AmountCents: 5000, Text: "Rechnung 12 vom 3.4."},
			confidence: "",
		},
		{
			name:        "IBAN and the sum of all open fees",
			entry:       Entry{AmountCents: 22000, Text: "Beitraege", IBAN: "DE75512108001245126199"},
			confidence:  Certain,
			allocations: []Allocation{{"f-andresen-2025", 4000}, {"f-andresen", 18000}},
			reasons:     []string{ReasonIBAN, ReasonAmount},
		},
		{
			name:        "IBAN with a partial payment goes to the oldest fee first",
			entry:       Entry{AmountCents: 9000, Text: "Beitrag erste Rate", IBAN: "DE75512108001245126199"},
			confidence:  Probable,
			allocations: []Allocation{{"f-andresen-2025", 4000}, {"f-andresen", 5000}},
			reasons:     []string{ReasonIBAN, ReasonPartial},
		},
		{
			name:        "overpayment leaves a rest",
			entry:       Entry{AmountCents: 30000, Text: "Nr. 1013"},
			confidence:  Probable,
			allocations: []Allocation{{"f-wendt", 18000}},
			reasons:     []string{ReasonMemberNumber, ReasonOverpaid},
		},
		{
			name:        "family transfer by names and amount",
			entry:       Entry{AmountCents: 21000, Text: "Jahresbeitrag Nele und Paul Kruse", Name: "Birte Kruse"},
			confidence:  Probable,
			allocations: []Allocation{{"f-nele", 9000}, {"f-paul", 12000}},
			reasons:     []string{ReasonName, ReasonAmount},
		},
		{
			name:        "household found by one member number",
			entry:       Entry{AmountCents: 21000, Text: "Familie Kruse Mitglied 1010"},
			confidence:  Certain,
			allocations: []Allocation{{"f-nele", 9000}, {"f-paul", 12000}},
			reasons:     []string{ReasonMemberNumber, ReasonAmount, ReasonHousehold},
		},
		{
			name:        "account holder and amount",
			entry:       Entry{AmountCents: 9000, Text: "Beitrag Tjark", Name: "FAMILIE ALBERS"},
			confidence:  Probable,
			allocations: []Allocation{{"f-albers", 9000}},
		},
		{
			name:        "transliterated name without the amount is unclear",
			entry:       Entry{AmountCents: 10000, Text: "Beitrag Greta Moeller", Name: "G. Moeller"},
			confidence:  Unclear,
			allocations: []Allocation{{"f-moeller", 10000}},
			reasons:     []string{ReasonName, ReasonPartial},
		},
		{
			name:        "typo in the name",
			entry:       Entry{AmountCents: 18000, Text: "Mats Andressen"},
			confidence:  Probable,
			allocations: []Allocation{{"f-andresen", 18000}},
		},
		{
			name:       "no member at all",
			entry:      Entry{AmountCents: 25000, Text: "Sponsoring Kitesurf-Festival 2026", Name: "Surfshop Fehmarn GmbH"},
			confidence: "",
		},
		{
			name:       "member without open fees is not suggested",
			entry:      Entry{AmountCents: 18000, Text: "Beitrag", IBAN: "DE89370400440532013000"},
			confidence: "",
		},
		{
			name:        "return found by end-to-end id",
			entry:       Entry{AmountCents: 18300, Return: true, EndToEndID: "0190abcd1234"},
			fees:        []Fee{{ID: "0190abcd-1234", MemberID: "lorenzen", AmountCents: 18000, DueDate: due(3)}},
			confidence:  Certain,
			allocations: []Allocation{{"0190abcd-1234", 18000}},
			reasons:     []string{ReasonEndToEnd, ReasonOverpaid},
		},
		{
			name:  "return found by IBAN reverses the newest paid fee",
			entry: Entry{AmountCents: 18000, Return: true, IBAN: "DE02100500000054540402", Text: "RUECKLASTSCHRIFT"},
			fees: []Fee{
				{ID: "old", MemberID: "lorenzen", AmountCents: 18000, DueDate: due(3).AddDate(-1, 0, 0)},
				{ID: "new", MemberID: "lorenzen", AmountCents: 18000, DueDate: due(3)},
			},
			confidence:  Certain,
			allocations: []Allocation{{"new", 18000}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fees := open
			if tc.fees != nil {
				fees = tc.fees
			}
			s := Match(tc.entry, members, fees)
			if tc.confidence == "" {
				if s != nil {
					t.Fatalf("unexpected suggestion %+v", s)
				}
				return
			}
			if s == nil {
				t.Fatal("no suggestion")
			}
			if s.Confidence != tc.confidence {
				t.Errorf("confidence %s, want %s (%v)", s.Confidence, tc.confidence, s.Reasons)
			}
			if !slices.Equal(s.Allocations, tc.allocations) {
				t.Errorf("allocations %v, want %v", s.Allocations, tc.allocations)
			}
			if tc.reasons != nil && !slices.Equal(s.Reasons, tc.reasons) {
				t.Errorf("reasons %v, want %v", s.Reasons, tc.reasons)
			}
		})
	}
}

func TestNamesOfDifferentHouseholdsAreUnclear(t *testing.T) {
	members := []Member{
		{ID: "a", Number: 1, FirstName: "Anna", LastName: "Schulz"},
		{ID: "b", Number: 2, FirstName: "Anna", LastName: "Schulz"},
	}
	fees := []Fee{{ID: "fa", MemberID: "a", AmountCents: 5000}, {ID: "fb", MemberID: "b", AmountCents: 5000}}
	s := Match(Entry{AmountCents: 5000, Text: "Beitrag Anna Schulz"}, members, fees)
	if s == nil || s.Confidence != Unclear {
		t.Fatalf("namesakes: %+v", s)
	}
}

func TestMemberNumbers(t *testing.T) {
	for text, want := range map[string][]int{
		"Mitglied 12":                    {12},
		"Mitgliedsnr. 7 Beitrag 2026":    {7},
		"Member no. 42":                  {42},
		"#003 Beitrag":                   {3},
		"Beitrag 2026 fuer 1013":         {1013},
		"Kino 12 Uhr":                    {},
		"Mitgliedsnummer: 1013, Nr 1014": {1013, 1014},
	} {
		got := memberNumbers(text)
		if len(got) != len(want) {
			t.Errorf("%q: %v, want %v", text, got, want)
		}
		for _, n := range want {
			if !got[n] {
				t.Errorf("%q: %d missing in %v", text, n, got)
			}
		}
	}
}
