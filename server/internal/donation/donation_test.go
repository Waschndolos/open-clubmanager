package donation

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/pdfdoc"
)

func TestWords(t *testing.T) {
	cases := map[int64]string{
		0:         "null",
		1:         "eins",
		7:         "sieben",
		16:        "sechzehn",
		17:        "siebzehn",
		21:        "einundzwanzig",
		30:        "dreißig",
		99:        "neunundneunzig",
		100:       "einhundert",
		101:       "einhunderteins",
		150:       "einhundertfünfzig",
		1000:      "eintausend",
		1001:      "eintausendeins",
		2345:      "zweitausenddreihundertfünfundvierzig",
		71000:     "einundsiebzigtausend",
		1_000_000: "eine Million",
		2_500_021: "zwei Millionen fünfhunderttausendeinundzwanzig",
	}
	for n, want := range cases {
		if got := Words(n); got != want {
			t.Errorf("Words(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestAmountWords(t *testing.T) {
	cases := map[int64]string{
		100:    "ein Euro",
		105:    "ein Euro und fünf Cent",
		101:    "ein Euro und ein Cent",
		15000:  "einhundertfünfzig Euro",
		25050:  "zweihundertfünfzig Euro und fünfzig Cent",
		5:      "null Euro und fünf Cent",
		120000: "eintausendzweihundert Euro",
	}
	for cents, want := range cases {
		if got := AmountWords(cents); got != want {
			t.Errorf("AmountWords(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestAmount(t *testing.T) {
	cases := map[int64]string{0: "0,00 €", 5: "0,05 €", 15000: "150,00 €", 123456789: "1.234.567,89 €", -250: "-2,50 €"}
	for cents, want := range cases {
		if got := Amount(cents); got != want {
			t.Errorf("Amount(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestExemptionExpired(t *testing.T) {
	notice := Issuer{ExemptionKind: ExemptionNotice, ExemptionDate: time.Date(2021, 3, 15, 0, 0, 0, 0, time.UTC)}
	if notice.ExemptionExpired(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)) {
		t.Error("a notice is valid for 5 years, including the last day")
	}
	if !notice.ExemptionExpired(time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)) {
		t.Error("a notice older than 5 years must be expired")
	}
	assessment := Issuer{ExemptionKind: ExemptionAssessment, ExemptionDate: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)}
	if !assessment.ExemptionExpired(time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)) {
		t.Error("an assessment under § 60a AO is valid for 3 years only")
	}
}

var shown = regexp.MustCompile(`\(((?:[^()\\]|\\.)*)\) Tj`)

// text joins the strings shown in the PDF with spaces, so that phrases
// wrapped across lines can be found.
func text(pdf []byte) []byte {
	var parts [][]byte
	for _, m := range shown.FindAllSubmatch(pdf, -1) {
		parts = append(parts, m[1])
	}
	return bytes.Join(parts, []byte(" "))
}

func contains(t *testing.T, pdf []byte, texts ...string) {
	t.Helper()
	all := text(pdf)
	for _, s := range texts {
		if !bytes.Contains(all, []byte(pdfdoc.WinAnsi(s))) {
			t.Errorf("PDF lacks %q", s)
		}
	}
}

func TestPDF(t *testing.T) {
	issuer := Issuer{Name: "Kiteclub Nordwind e.V.", Address: "Deichweg 1\n25980 Sylt", TaxOffice: "Nordfriesland",
		TaxNumber: "15/290/12345", ExemptionKind: ExemptionNotice, ExemptionDate: time.Date(2024, 5, 2, 0, 0, 0, 0, time.UTC),
		AssessmentPeriod: "2023", Purposes: "des Sports", Signatory: "Kai Kasse", Place: "Sylt"}
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	single := Receipt{Number: "2026-0001", Kind: KindSingle, RecipientName: "Merle Hansen", RecipientAddress: "Strandweg 2\n25980 Sylt",
		PeriodStart: day(3, 1), PeriodEnd: day(3, 1), IssueDate: day(12, 30), Issuer: issuer,
		Items: []Item{{Date: day(3, 1), Kind: ItemDonation, AmountCents: 15000}}}

	pdf := PDF([]Receipt{single})
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("not a PDF")
	}
	contains(t, pdf, titleSingle, "Lfd. Nr. 2026-0001", "Merle Hansen", "Strandweg 2", "150,00 €",
		"- einhundertfünfzig Euro -", "Tag der Zuwendung:", "01.03.2026", "Ja [  ]", "Nein [X]",
		"Finanzamtes Nordfriesland, StNr. 15/290/12345, vom 02.05.2024", "Veranlagungszeitraum 2023",
		"Sylt, 30.12.2026", "Kai Kasse", "Nur für steuerbegünstigte Einrichtungen")
	if bytes.Contains(pdf, []byte("DUPLIKAT")) {
		t.Error("an original must not be marked as duplicate")
	}

	collective := single
	collective.Kind, collective.Duplicate, collective.Number = KindCollective, true, "2026-0002"
	collective.PeriodStart, collective.PeriodEnd = day(1, 1), day(11, 15)
	collective.Issuer.ExemptionKind, collective.Issuer.FeesDeductible = ExemptionAssessment, true
	collective.Items = []Item{
		{Date: day(3, 1), Kind: ItemDonation, AmountCents: 15000},
		{Date: day(4, 1), Kind: ItemMembershipFee, AmountCents: 9050},
		{Date: day(11, 15), Kind: ItemDonation, AmountCents: 4000, ExpenseWaiver: true},
	}
	pdf = PDF([]Receipt{collective})
	contains(t, pdf, "DUPLIKAT", titleCollective, "Gesamtbetrag der Zuwendung", "280,50 €",
		"01.01.2026 - 15.11.2026", "Anlage zur Sammelbestätigung", "Mitgliedsbeitrag", "Geldzuwendung", "ja", "40,00 €",
		"nach § 60a AO gesondert festgestellt", "keine weiteren Bestätigungen")
	if bytes.Contains(pdf, []byte(pdfdoc.WinAnsi("Nur für steuerbegünstigte Einrichtungen"))) {
		t.Error("the membership fee note is only for clubs whose fees are not deductible")
	}
}
