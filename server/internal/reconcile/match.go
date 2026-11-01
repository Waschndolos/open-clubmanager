// Package reconcile suggests which member fees a bank statement entry pays
// (or, for a returned direct debit, reverses). It only proposes; the
// treasurer confirms every assignment.
//
// An entry is matched by the member number in the remittance text, the
// payer's IBAN and the payer's name, together with the amount:
//
//   - certain: member number or IBAN identify the member(s), and the amount
//     equals one open fee, all open fees of those members or of their
//     household; for returns also the end-to-end ID of an exported debit
//   - probable: member number or IBAN with a different amount (partial
//     payment, overpayment), or the name together with an equal amount
//   - unclear: only the name, without an equal amount, or names of members
//     of different households
package reconcile

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Confidence of a suggestion.
type Confidence string

const (
	Certain  Confidence = "certain"
	Probable Confidence = "probable"
	Unclear  Confidence = "unclear"
)

// Reasons why a suggestion was made.
const (
	ReasonEndToEnd     = "END_TO_END_ID"
	ReasonMemberNumber = "MEMBER_NUMBER"
	ReasonIBAN         = "IBAN"
	ReasonName         = "NAME"
	ReasonAmount       = "AMOUNT"
	ReasonHousehold    = "HOUSEHOLD"
	ReasonPartial      = "PARTIAL"
	ReasonOverpaid     = "OVERPAID"
)

// Entry is a bank statement entry to reconcile.
type Entry struct {
	// AmountCents is positive, also for a return.
	AmountCents int64
	// Text is the remittance information.
	Text       string
	Name       string
	IBAN       string
	EndToEndID string
	// Return marks a returned direct debit: it is matched against paid fees.
	Return bool
}

// Member is a member that may pay fees.
type Member struct {
	ID            string
	Number        int
	FirstName     string
	LastName      string
	AccountHolder string
	IBAN          string
	// HouseholdID is empty for members without a household.
	HouseholdID string
}

// Fee is a fee that may be assigned: for payments an open fee with the
// amount still due, for returns a paid fee with the amount that was paid.
type Fee struct {
	ID          string
	MemberID    string
	AmountCents int64
	DueDate     time.Time
}

// Allocation assigns part of an entry to a fee.
type Allocation struct {
	FeeID       string
	AmountCents int64
}

// Suggestion is a proposed assignment of an entry.
type Suggestion struct {
	Confidence  Confidence
	Reasons     []string
	Allocations []Allocation
	// UnallocatedCents is the part of the entry not assigned to a fee.
	UnallocatedCents int64
}

// Match suggests fees for an entry, or returns nil when nothing fits.
func Match(e Entry, members []Member, fees []Fee) *Suggestion {
	if e.AmountCents <= 0 {
		return nil
	}
	feesOf := map[string][]Fee{}
	for _, f := range fees {
		if f.AmountCents > 0 {
			feesOf[f.MemberID] = append(feesOf[f.MemberID], f)
		}
	}
	// Payments go to the oldest fees first, returns reverse the newest.
	for id := range feesOf {
		list := feesOf[id]
		sort.SliceStable(list, func(i, j int) bool {
			if e.Return {
				return list[i].DueDate.After(list[j].DueDate)
			}
			return list[i].DueDate.Before(list[j].DueDate)
		})
	}
	// Only members with fees to assign take part.
	candidates := make([]Member, 0, len(members))
	byID := map[string]Member{}
	for _, m := range members {
		if len(feesOf[m.ID]) > 0 {
			candidates = append(candidates, m)
			byID[m.ID] = m
		}
	}

	if e.Return && e.EndToEndID != "" {
		ref := compactID(e.EndToEndID)
		for _, f := range fees {
			if compactID(f.ID) == ref {
				return allocate(e, []Fee{f}, Certain, []string{ReasonEndToEnd})
			}
		}
	}

	text := normalize(e.Text)
	name := normalize(e.Name)
	iban := normalizeIBAN(e.IBAN)
	numbers := memberNumbers(e.Text)

	var strong, named, surnamed []Member
	strongReasons := map[string]bool{}
	for _, m := range candidates {
		hit := false
		if numbers[m.Number] {
			strongReasons[ReasonMemberNumber], hit = true, true
		}
		if iban != "" && normalizeIBAN(m.IBAN) == iban {
			strongReasons[ReasonIBAN], hit = true, true
		}
		if hit {
			strong = append(strong, m)
			continue
		}
		switch nameMatch(m, text+" "+name) {
		case fullName:
			named = append(named, m)
		case lastName:
			surnamed = append(surnamed, m)
		}
	}

	if len(strong) > 0 {
		reasons := sortedKeys(strongReasons)
		if s := exactAmount(e, strong, candidates, feesOf, Certain, reasons); s != nil {
			return s
		}
		return allocate(e, feesFor(strong, feesOf), Probable, reasons)
	}
	for _, group := range [][]Member{named, surnamed} {
		if len(group) == 0 {
			continue
		}
		if s := exactAmount(e, group, candidates, feesOf, Probable, []string{ReasonName}); s != nil && sameHousehold(group, byID) {
			return s
		}
	}
	if len(named) > 0 {
		return allocate(e, feesFor(named, feesOf), Unclear, []string{ReasonName})
	}
	return nil
}

// exactAmount looks for fees of the members, or of their households, whose
// sum equals the entry: one fee, all fees of the members, or all fees of the
// households.
func exactAmount(e Entry, group, all []Member, feesOf map[string][]Fee, c Confidence, reasons []string) *Suggestion {
	withAmount := append(append([]string{}, reasons...), ReasonAmount)
	var single []Fee
	for _, m := range group {
		for _, f := range feesOf[m.ID] {
			if f.AmountCents == e.AmountCents {
				single = append(single, f)
			}
		}
	}
	if len(single) > 0 {
		// Several equal fees: the oldest (for returns the newest) one.
		sortFees(single, e.Return)
		return allocate(e, single[:1], c, withAmount)
	}
	own := feesFor(group, feesOf)
	if sum(own) == e.AmountCents {
		return allocate(e, own, c, withAmount)
	}
	households := map[string]bool{}
	for _, m := range group {
		if m.HouseholdID != "" {
			households[m.HouseholdID] = true
		}
	}
	if len(households) == 0 {
		return nil
	}
	var family []Member
	for _, m := range all {
		if households[m.HouseholdID] {
			family = append(family, m)
		}
	}
	if fam := feesFor(family, feesOf); len(family) > len(group) && sum(fam) == e.AmountCents {
		return allocate(e, fam, c, append(withAmount, ReasonHousehold))
	}
	return nil
}

// allocate assigns the entry to the fees in order until it is used up.
func allocate(e Entry, fees []Fee, c Confidence, reasons []string) *Suggestion {
	s := &Suggestion{Confidence: c, Reasons: reasons, Allocations: []Allocation{}}
	left := e.AmountCents
	for _, f := range fees {
		if left == 0 {
			break
		}
		amount := min(left, f.AmountCents)
		s.Allocations = append(s.Allocations, Allocation{FeeID: f.ID, AmountCents: amount})
		left -= amount
	}
	if len(s.Allocations) == 0 {
		return nil
	}
	s.UnallocatedCents = left
	switch {
	case left > 0:
		s.Reasons = append(s.Reasons, ReasonOverpaid)
	case !e.Return && sum(fees) > e.AmountCents:
		s.Reasons = append(s.Reasons, ReasonPartial)
	}
	return s
}

func feesFor(members []Member, feesOf map[string][]Fee) []Fee {
	var out []Fee
	for _, m := range members {
		out = append(out, feesOf[m.ID]...)
	}
	return out
}

func sortFees(fees []Fee, newestFirst bool) {
	sort.SliceStable(fees, func(i, j int) bool {
		if newestFirst {
			return fees[i].DueDate.After(fees[j].DueDate)
		}
		return fees[i].DueDate.Before(fees[j].DueDate)
	})
}

func sum(fees []Fee) int64 {
	var total int64
	for _, f := range fees {
		total += f.AmountCents
	}
	return total
}

// sameHousehold reports whether the members are one person or all belong to
// the same household, so a name shared by strangers is not taken as certain.
func sameHousehold(group []Member, byID map[string]Member) bool {
	if len(group) == 1 {
		return true
	}
	h := byID[group[0].ID].HouseholdID
	for _, m := range group {
		if h == "" || m.HouseholdID != h {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ── text matching ────────────────────────────────────────────────────────────

type nameHit int

const (
	noName nameHit = iota
	lastName
	fullName
)

// nameMatch reports how well the member's name, or the account holder,
// appears in the normalized text. Words may differ by one letter (typos,
// transliterations the normalization misses).
func nameMatch(m Member, text string) nameHit {
	words := strings.Fields(text)
	if holder := normalize(m.AccountHolder); holder != "" && containsAll(words, strings.Fields(holder)) {
		return fullName
	}
	last := strings.Fields(normalize(m.LastName))
	if len(last) == 0 || !containsAll(words, last) {
		return noName
	}
	if first := strings.Fields(normalize(m.FirstName)); len(first) > 0 && containsAny(words, first[:1]) {
		return fullName
	}
	return lastName
}

func containsAll(words, want []string) bool {
	for _, w := range want {
		if !containsAny(words, []string{w}) {
			return false
		}
	}
	return true
}

func containsAny(words, want []string) bool {
	for _, w := range want {
		for _, word := range words {
			if similar(word, w) {
				return true
			}
		}
	}
	return false
}

// similar compares words exactly, or for words of five or more letters with
// at most one edit.
func similar(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) < 5 || len(b) < 5 {
		return false
	}
	return withinOneEdit(a, b)
}

func withinOneEdit(a, b string) bool {
	if len(a) < len(b) {
		a, b = b, a
	}
	if len(a)-len(b) > 1 {
		return false
	}
	i, j, edits := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			i++
			j++
			continue
		}
		edits++
		if edits > 1 {
			return false
		}
		if len(a) == len(b) {
			j++
		}
		i++
	}
	return edits+(len(a)-i) <= 1
}

var transliteration = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"á", "a", "à", "a", "â", "a", "å", "a", "æ", "ae", "ç", "c",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "í", "i", "ì", "i", "î", "i", "ï", "i",
	"ñ", "n", "ó", "o", "ò", "o", "ô", "o", "ø", "oe", "ú", "u", "ù", "u", "û", "u", "ý", "y",
)

// normalize lowercases, transliterates umlauts and accents and reduces the
// text to words of letters and digits.
func normalize(s string) string {
	s = transliteration.Replace(strings.ToLower(s))
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}), " ")
}

func normalizeIBAN(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}

func compactID(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", ""))
}

var (
	// A number after a keyword such as "Mitglied", "Mitgl.-Nr.", "Nr." or "#".
	numberAfterKeyword = regexp.MustCompile(`(?i)(?:\b(?:mitglieds?(?:nummer|nr)?|mitgl|member|mnr|nr|no)|#)\s*[.:\-]*\s*(?:nr\.?\s*)?0*(\d{1,9})\b`)
	anyNumber          = regexp.MustCompile(`\b0*(\d{1,9})\b`)
)

// memberNumbers returns the member numbers the text may refer to: numbers
// after a keyword, and other numbers of four or more digits that are not a
// plausible year. Shorter bare numbers are too often amounts or dates.
func memberNumbers(text string) map[int]bool {
	out := map[int]bool{}
	for _, m := range numberAfterKeyword.FindAllStringSubmatch(text, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out[n] = true
		}
	}
	for _, m := range anyNumber.FindAllStringSubmatch(text, -1) {
		if len(m[1]) < 4 {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && (n < 1900 || n > 2100) {
			out[n] = true
		}
	}
	return out
}
