// Package sepa validates SEPA identifiers and writes SEPA Core direct debit
// initiations (pain.008.001.08) for upload to the club's bank.
package sepa

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// NormalizeIBAN removes spaces and upper-cases an IBAN as typed by a person.
func NormalizeIBAN(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}

var ibanFormat = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$`)

// ValidIBAN reports whether iban (normalized) is well-formed and its check
// digits are correct (ISO 13616, mod 97).
func ValidIBAN(iban string) bool {
	if !ibanFormat.MatchString(iban) {
		return false
	}
	return mod97(iban[4:]+iban[:4]) == 1
}

var bicFormat = regexp.MustCompile(`^[A-Z]{6}[A-Z2-9][A-NP-Z0-9]([A-Z0-9]{3})?$`)

// ValidBIC reports whether bic (upper-case, no spaces) is a well-formed BIC.
func ValidBIC(bic string) bool { return bicFormat.MatchString(bic) }

var creditorIDFormat = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{3}[A-Z0-9]{1,28}$`)

// NormalizeCreditorID removes spaces and upper-cases a creditor identifier.
func NormalizeCreditorID(s string) string { return NormalizeIBAN(s) }

// ValidCreditorID reports whether id is a well-formed SEPA creditor
// identifier with correct check digits. Positions 5–7 (the creditor business
// code, "ZZZ" by default) are not part of the check (EPC262-08).
func ValidCreditorID(id string) bool {
	if !creditorIDFormat.MatchString(id) || len(id) > 35 {
		return false
	}
	return mod97(id[7:]+id[:4]) == 1
}

// mandateIDFormat is the character set the German banks accept for mandate
// references (DFÜ-Abkommen, Anlage 3): no spaces, at most 35 characters.
var mandateIDFormat = regexp.MustCompile(`^[A-Za-z0-9+?/\-:().,']{1,35}$`)

// ValidMandateReference reports whether ref may be used as a mandate
// reference (MndtId).
func ValidMandateReference(ref string) bool { return mandateIDFormat.MatchString(ref) }

// mod97 computes the ISO 7064 MOD 97-10 remainder of s, with letters
// replaced by two-digit numbers (A=10 … Z=35).
func mod97(s string) int {
	var digits strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			digits.WriteString(strconv.Itoa(int(r - 'A' + 10)))
		default:
			return -1
		}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	if !ok {
		return -1
	}
	return int(new(big.Int).Mod(n, big.NewInt(97)).Int64())
}

var transliterations = map[rune]string{
	'Ä': "Ae", 'Ö': "Oe", 'Ü': "Ue", 'ä': "ae", 'ö': "oe", 'ü': "ue", 'ß': "ss",
	'&': "+", '–': "-", '—': "-", '_': "-", '’': "'", '‘': "'",
	'é': "e", 'è': "e", 'ê': "e", 'á': "a", 'à': "a", 'â': "a", 'ç': "c",
	'É': "E", 'È': "E", 'À': "A", 'Ç': "C", 'ô': "o", 'î': "i", 'ï': "i", 'ë': "e", 'ñ': "n",
}

// Text converts s to the SEPA character set (EPC217-08: a–z A–Z 0–9 / - ? : ( ) . , ' + and
// space), transliterating umlauts, and cuts it to max characters.
func Text(s string, max int) string {
	var b strings.Builder
	for _, r := range s {
		if t, ok := transliterations[r]; ok {
			b.WriteString(t)
			continue
		}
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case strings.ContainsRune("/-?:().,'+", r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > max {
		out = strings.TrimSpace(out[:max])
	}
	return out
}
