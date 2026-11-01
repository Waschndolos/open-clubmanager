// Package donation writes donation receipts (Zuwendungsbestätigungen) after
// the official template of the German Federal Ministry of Finance for
// monetary donations and membership fees to a tax-privileged corporation
// (§ 5 Abs. 1 Nr. 9 KStG): the single receipt and the collective receipt
// (Sammelbestätigung) with its annex.
package donation

import (
	"strconv"
	"strings"
	"time"
)

// Kinds of exemption basis.
const (
	// ExemptionNotice is a Freistellungsbescheid or the annex to the
	// corporate tax assessment; receipts are recognised for 5 years.
	ExemptionNotice = "notice"
	// ExemptionAssessment is the separate assessment of the statutes under
	// § 60a AO; receipts are recognised for 3 years.
	ExemptionAssessment = "assessment"
)

// Kinds of a receipt and of its items.
const (
	KindSingle     = "single"
	KindCollective = "collective"

	ItemDonation      = "donation"
	ItemMembershipFee = "membership_fee"
)

// Issuer is the club's data printed on a receipt. It is copied into every
// receipt, so that a receipt can be printed again unchanged.
type Issuer struct {
	Name             string    `json:"name"`
	Address          string    `json:"address"`
	TaxOffice        string    `json:"taxOffice"`
	TaxNumber        string    `json:"taxNumber"`
	ExemptionKind    string    `json:"exemptionKind"`
	ExemptionDate    time.Time `json:"exemptionDate"`
	AssessmentPeriod string    `json:"assessmentPeriod,omitempty"`
	Purposes         string    `json:"purposes"`
	FeesDeductible   bool      `json:"feesDeductible"`
	Signatory        string    `json:"signatory,omitempty"`
	Place            string    `json:"place,omitempty"`
}

// ValidYears is how long receipts based on the exemption are recognised
// (§ 63 Abs. 5 AO).
func (i Issuer) ValidYears() int {
	if i.ExemptionKind == ExemptionAssessment {
		return 3
	}
	return 5
}

// ExemptionExpired reports whether receipts issued on the given day would no
// longer be recognised because the exemption is too old.
func (i Issuer) ExemptionExpired(on time.Time) bool {
	return on.After(i.ExemptionDate.AddDate(i.ValidYears(), 0, 0))
}

// Item is one certified donation or membership fee.
type Item struct {
	Date          time.Time
	Kind          string
	AmountCents   int64
	ExpenseWaiver bool
}

// Receipt holds what is printed on one receipt.
type Receipt struct {
	Number           string
	Kind             string
	Duplicate        bool
	Voided           bool
	RecipientName    string
	RecipientAddress string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	IssueDate        time.Time
	Issuer           Issuer
	Items            []Item
}

// TotalCents is the sum of the receipt's items.
func (r Receipt) TotalCents() int64 {
	var sum int64
	for _, it := range r.Items {
		sum += it.AmountCents
	}
	return sum
}

// Amount formats cents in German notation: 1.234,50 €.
func Amount(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	euros := strconv.FormatInt(cents/100, 10)
	var b strings.Builder
	for i, c := range euros {
		if i > 0 && (len(euros)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	s := b.String() + "," + pad2(cents%100) + " €"
	if neg {
		s = "-" + s
	}
	return s
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

var (
	ones = []string{"", "ein", "zwei", "drei", "vier", "fünf", "sechs", "sieben", "acht", "neun", "zehn",
		"elf", "zwölf", "dreizehn", "vierzehn", "fünfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
	tens = []string{"", "", "zwanzig", "dreißig", "vierzig", "fünfzig", "sechzig", "siebzig", "achtzig", "neunzig"}
)

// below1000 spells 1..999; a trailing one stays "ein" (as in "einhundert").
func below1000(n int64) string {
	var s string
	if n >= 100 {
		s = ones[n/100] + "hundert"
		n %= 100
	}
	switch {
	case n == 0:
	case n < 20:
		s += ones[n]
	case n%10 == 0:
		s += tens[n/10]
	default:
		s += ones[n%10] + "und" + tens[n/10]
	}
	return s
}

// Words spells a non-negative whole number in German, as used for amounts:
// 1 "eins", 101 "einhunderteins", 2.000.000 "zwei Millionen".
func Words(n int64) string {
	if n == 0 {
		return "null"
	}
	var parts []string
	if m := n / 1_000_000; m > 0 {
		if m == 1 {
			parts = append(parts, "eine Million")
		} else {
			parts = append(parts, Words(m)+" Millionen")
		}
		n %= 1_000_000
	}
	var s string
	if t := n / 1000; t > 0 {
		s = below1000(t) + "tausend"
		n %= 1000
	}
	if n > 0 {
		s += below1000(n)
		if n%100 == 1 {
			s += "s"
		}
	}
	if s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// AmountWords spells an amount in euros for the receipt: "einhundertfünfzig
// Euro", "ein Euro und fünf Cent".
func AmountWords(cents int64) string {
	euros, rest := cents/100, cents%100
	var s string
	if euros == 1 {
		s = "ein Euro"
	} else {
		s = Words(euros) + " Euro"
	}
	if rest > 0 {
		cent := Words(rest)
		if rest == 1 {
			cent = "ein"
		}
		s += " und " + cent + " Cent"
	}
	return s
}
