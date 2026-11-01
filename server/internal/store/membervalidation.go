package store

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Waschndolos/open-clubmanager/server/internal/sepa"
)

// Codes of a FieldError. The frontend translates them, so they are part of
// the API (ErrorResponse.fields in openapi.yaml).
const (
	CodeRequired      = "required"     // empty or whitespace only
	CodeTooLong       = "too_long"     // longer than FieldError.Limit characters
	CodeNotPositive   = "not_positive" // member number below 1
	CodeInvalid       = "invalid"      // malformed (e-mail, IBAN country/length/characters, BIC)
	CodeChecksum      = "checksum"     // IBAN check digits are wrong
	CodeBeforeEntry   = "before_entry" // exit date before entry date
	CodeInFuture      = "in_future"    // birthday after today
	maxNameLength     = 100
	maxEmailLength    = 254
	maxShortLength    = 50  // phone numbers, postal code
	maxTextLength     = 200 // street, city, state, account holder, bank name
	maxCommentLength  = 2000
	maxMemberNumber   = 999_999_999
	calendarDayLayout = "2006-01-02"
)

// FieldError is one problem with one field of a member. Field uses the JSON
// names of the API (firstName, iban, …).
type FieldError struct {
	Field string
	Code  string
	// Limit is the maximum length for CodeTooLong, otherwise 0.
	Limit int
}

// ValidationError lists every problem found in a member input.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Code
	}
	return "invalid member data (" + strings.Join(parts, ", ") + ")"
}

// NormalizeMemberInput trims all text fields, turns empty optional fields
// into nil and writes IBAN and BIC without spaces in upper case.
func NormalizeMemberInput(in *MemberInput) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	trim := func(p **string, normalize func(string) string) {
		if *p == nil {
			return
		}
		v := normalize(**p)
		if v == "" {
			*p = nil
			return
		}
		*p = &v
	}
	for _, p := range []**string{&in.Comment, &in.Contact.Email, &in.Contact.Phone, &in.Contact.PhoneMobile,
		&in.Contact.Street, &in.Contact.PostalCode, &in.Contact.City, &in.Contact.State,
		&in.Bank.AccountHolder, &in.Bank.BankName} {
		trim(p, strings.TrimSpace)
	}
	trim(&in.Bank.IBAN, sepa.NormalizeIBAN)
	trim(&in.Bank.BIC, sepa.NormalizeBIC)
}

// ValidateMemberInput checks a normalized member input (NormalizeMemberInput)
// and returns a *ValidationError listing every problem, or nil. today is the
// local date of the person entering the data; a birthday may not lie after it.
//
// Every local write of a member goes through this check (store.CreateMember,
// store.UpdateMember, used by the REST API and so also by the member import
// of the frontend). Changes pulled from other devices of a collaboration club
// are applied without it (package replica): they are per-field writes that
// were validated on the device that made them, and dropping some of them
// would let the devices' data diverge for good.
func ValidateMemberInput(in MemberInput, today time.Time) error {
	var errs []FieldError
	add := func(field, code string, limit int) {
		errs = append(errs, FieldError{Field: field, Code: code, Limit: limit})
	}
	text := func(field string, v *string, limit int) {
		if v != nil && utf8.RuneCountInString(*v) > limit {
			add(field, CodeTooLong, limit)
		}
	}

	if in.Number < 1 {
		add("number", CodeNotPositive, 0)
	} else if in.Number > maxMemberNumber {
		add("number", CodeTooLong, len("999999999"))
	}
	for _, name := range []struct {
		field, value string
	}{{"firstName", in.FirstName}, {"lastName", in.LastName}} {
		if name.value == "" {
			add(name.field, CodeRequired, 0)
		} else {
			text(name.field, &name.value, maxNameLength)
		}
	}

	if in.Birthday != nil && calendarDay(*in.Birthday) > latestToday(today) {
		add("birthday", CodeInFuture, 0)
	}
	if in.EntryDate != nil && in.ExitDate != nil && calendarDay(*in.ExitDate) < calendarDay(*in.EntryDate) {
		add("exitDate", CodeBeforeEntry, 0)
	}
	text("comment", in.Comment, maxCommentLength)

	c := in.Contact
	if c.Email != nil {
		if utf8.RuneCountInString(*c.Email) > maxEmailLength {
			add("email", CodeTooLong, maxEmailLength)
		} else if !validEmail(*c.Email) {
			add("email", CodeInvalid, 0)
		}
	}
	text("phone", c.Phone, maxShortLength)
	text("phoneMobile", c.PhoneMobile, maxShortLength)
	text("street", c.Street, maxTextLength)
	text("postalCode", c.PostalCode, maxShortLength)
	text("city", c.City, maxTextLength)
	text("state", c.State, maxTextLength)

	b := in.Bank
	text("accountHolder", b.AccountHolder, maxTextLength)
	if b.IBAN != nil {
		switch {
		case !sepa.WellFormedIBAN(*b.IBAN):
			add("iban", CodeInvalid, 0)
		case !sepa.ValidIBAN(*b.IBAN):
			add("iban", CodeChecksum, 0)
		}
	}
	if b.BIC != nil && !sepa.ValidBIC(*b.BIC) {
		add("bic", CodeInvalid, 0)
	}
	text("bankName", b.BankName, maxTextLength)

	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

// validEmail accepts a plain address (no display name) with a local part and
// a domain.
func validEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || a.Name != "" {
		return false
	}
	at := strings.LastIndexByte(s, '@')
	return at > 0 && at < len(s)-1
}

// calendarDay is the day a date stands for. The API sends calendar days as
// UTC midnight, so the UTC date is the picked day.
func calendarDay(t time.Time) string { return t.UTC().Format(calendarDayLayout) }

// latestToday is the later of today's local and UTC date, so a birthday
// entered today is accepted whatever time zone the client is in.
func latestToday(now time.Time) string {
	local, utc := now.Format(calendarDayLayout), now.UTC().Format(calendarDayLayout)
	if utc > local {
		return utc
	}
	return local
}

// prepareMemberInput normalizes and validates in for writing.
func prepareMemberInput(in *MemberInput) error {
	NormalizeMemberInput(in)
	return ValidateMemberInput(*in, time.Now())
}
