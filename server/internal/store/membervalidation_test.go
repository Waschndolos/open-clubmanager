package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateMemberInput(t *testing.T) {
	today := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	day := func(s string) *time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}
	valid := func() MemberInput {
		return MemberInput{Number: 1, FirstName: "Ada", LastName: "Lovelace"}
	}
	long := func(n int) *string { s := strings.Repeat("x", n); return &s }

	tests := []struct {
		name   string
		modify func(*MemberInput)
		want   []FieldError
	}{
		{"minimal member", func(*MemberInput) {}, nil},
		{"complete member", func(in *MemberInput) {
			in.Birthday, in.EntryDate, in.ExitDate = day("1990-05-17"), day("2020-01-01"), day("2020-01-01")
			in.Comment = ptr("Gründungsmitglied")
			in.Contact = MemberContact{Email: ptr("ada@example.org"), Phone: ptr("+49 30 123456"), City: ptr("Kiel")}
			in.Bank = MemberBank{AccountHolder: ptr("Ada Lovelace"), IBAN: ptr("de89 3704 0044 0532 0130 00"), BIC: ptr("cobadeff xxx")}
		}, nil},
		{"birthday today", func(in *MemberInput) { in.Birthday = day("2026-11-01") }, nil},
		{"name of 100 characters", func(in *MemberInput) { in.FirstName = *long(100) }, nil},
		{"umlauts count as one character", func(in *MemberInput) { in.LastName = strings.Repeat("ä", 100) }, nil},
		{"empty body", func(in *MemberInput) { *in = MemberInput{} }, []FieldError{
			{"number", CodeNotPositive, 0}, {"firstName", CodeRequired, 0}, {"lastName", CodeRequired, 0}}},
		{"whitespace names", func(in *MemberInput) { in.FirstName, in.LastName = "  ", "\t" }, []FieldError{
			{"firstName", CodeRequired, 0}, {"lastName", CodeRequired, 0}}},
		{"negative number", func(in *MemberInput) { in.Number = -5 }, []FieldError{{"number", CodeNotPositive, 0}}},
		{"huge number", func(in *MemberInput) { in.Number = 1_000_000_000 }, []FieldError{{"number", CodeTooLong, 9}}},
		{"first name of 5000 characters", func(in *MemberInput) { in.FirstName = *long(5000) }, []FieldError{{"firstName", CodeTooLong, 100}}},
		{"invalid IBAN", func(in *MemberInput) { in.Bank.IBAN = ptr("XX00 not an iban") }, []FieldError{{"iban", CodeInvalid, 0}}},
		{"IBAN too short for its country", func(in *MemberInput) { in.Bank.IBAN = ptr("DE8937040044053201300") }, []FieldError{{"iban", CodeInvalid, 0}}},
		{"IBAN with wrong check digits", func(in *MemberInput) { in.Bank.IBAN = ptr("DE89370400440532013001") }, []FieldError{{"iban", CodeChecksum, 0}}},
		{"malformed BIC", func(in *MemberInput) { in.Bank.BIC = ptr("COBA") }, []FieldError{{"bic", CodeInvalid, 0}}},
		{"exit before entry", func(in *MemberInput) { in.EntryDate, in.ExitDate = day("2020-01-02"), day("2020-01-01") }, []FieldError{{"exitDate", CodeBeforeEntry, 0}}},
		{"exit without entry", func(in *MemberInput) { in.ExitDate = day("2020-01-01") }, nil},
		{"birthday in the future", func(in *MemberInput) { in.Birthday = day("2099-01-01") }, []FieldError{{"birthday", CodeInFuture, 0}}},
		{"birthday tomorrow", func(in *MemberInput) { in.Birthday = day("2026-11-02") }, []FieldError{{"birthday", CodeInFuture, 0}}},
		{"malformed e-mail", func(in *MemberInput) { in.Contact.Email = ptr("no-at-sign") }, []FieldError{{"email", CodeInvalid, 0}}},
		{"e-mail with display name", func(in *MemberInput) { in.Contact.Email = ptr("Ada <ada@example.org>") }, []FieldError{{"email", CodeInvalid, 0}}},
		{"long texts", func(in *MemberInput) {
			in.Comment, in.Contact.Phone, in.Contact.Street, in.Bank.BankName = long(2001), long(51), long(201), long(201)
		}, []FieldError{{"comment", CodeTooLong, 2000}, {"phone", CodeTooLong, 50}, {"street", CodeTooLong, 200}, {"bankName", CodeTooLong, 200}}},
		{"empty optional fields are dropped", func(in *MemberInput) {
			in.Contact.Email, in.Bank.IBAN, in.Bank.BIC = ptr(" "), ptr(""), ptr("  ")
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.modify(&in)
			NormalizeMemberInput(&in)
			err := ValidateMemberInput(in, today)
			var got []FieldError
			var ve *ValidationError
			if errors.As(err, &ve) {
				got = ve.Fields
			} else if err != nil {
				t.Fatalf("unexpected error type %T: %v", err, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNormalizeMemberInput(t *testing.T) {
	in := MemberInput{
		FirstName: "  Ada ", LastName: " Lovelace",
		Comment: ptr("   "),
		Contact: MemberContact{Email: ptr(" ada@example.org "), City: ptr("")},
		Bank:    MemberBank{IBAN: ptr(" de89 3704 0044 0532 0130 00 "), BIC: ptr("coba de ff xxx")},
	}
	NormalizeMemberInput(&in)
	if in.FirstName != "Ada" || in.LastName != "Lovelace" || in.Comment != nil || in.Contact.City != nil ||
		*in.Contact.Email != "ada@example.org" || *in.Bank.IBAN != "DE89370400440532013000" || *in.Bank.BIC != "COBADEFFXXX" {
		t.Errorf("normalized: %+v %+v %+v", in, in.Contact, in.Bank)
	}
}

// The store itself refuses invalid members, so no caller can skip the check.
func TestStoreRejectsInvalidMembers(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	var ve *ValidationError
	if _, err := s.CreateMember(ctx, MemberInput{}); !errors.As(err, &ve) {
		t.Fatalf("create empty member: %v", err)
	}
	m, err := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: " Ada ", LastName: "Lovelace",
		Bank: MemberBank{IBAN: ptr("de89 3704 0044 0532 0130 00")}})
	if err != nil {
		t.Fatal(err)
	}
	if m.FirstName != "Ada" || m.Bank.IBAN == nil || *m.Bank.IBAN != "DE89370400440532013000" {
		t.Errorf("stored %q / %v, want trimmed name and normalized IBAN", m.FirstName, m.Bank.IBAN)
	}
	if _, err := s.UpdateMember(ctx, m.ID, MemberInput{Number: 1, FirstName: "Ada", LastName: "Lovelace",
		Bank: MemberBank{IBAN: ptr("DE89370400440532013001")}}); !errors.As(err, &ve) || ve.Fields[0].Code != CodeChecksum {
		t.Fatalf("update with bad IBAN: %v", err)
	}
	if got, _ := s.GetMember(ctx, m.ID); *got.Bank.IBAN != "DE89370400440532013000" {
		t.Errorf("rejected update changed the IBAN to %s", *got.Bank.IBAN)
	}
}
