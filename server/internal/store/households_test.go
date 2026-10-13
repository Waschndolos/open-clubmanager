package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHouseholds(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	mk := func(n int, first string, c MemberContact, b MemberBank) Member {
		m, err := s.CreateMember(ctx, MemberInput{Number: n, FirstName: first, LastName: "Kite", Contact: c, Bank: b})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	parent := mk(1, "Pia", MemberContact{Email: ptr("pia@example.org"), Street: ptr("Deich 1"), City: ptr("Kiel")},
		MemberBank{AccountHolder: ptr("Pia Kite"), IBAN: ptr("DE02120300000000202051")})
	child := mk(2, "Ole", MemberContact{Email: ptr("ole@example.org"), City: ptr("Hamburg")}, MemberBank{})
	baby := mk(3, "Ida", MemberContact{}, MemberBank{})
	other := mk(4, "Max", MemberContact{}, MemberBank{})

	h, err := s.CreateHousehold(ctx, HouseholdInput{Name: "Familie Kite", PayerID: &parent.ID, ShareAddress: true,
		ShareBank: true, MemberIDs: []string{parent.ID, child.ID, baby.ID}})
	if err != nil {
		t.Fatalf("CreateHousehold: %v", err)
	}
	if h.Payer == nil || h.Payer.ID != parent.ID || len(h.Members) != 3 {
		t.Fatalf("household = %+v", h)
	}

	// Creating the household takes over the payer's address and bank details,
	// without touching the child's own e-mail.
	got, _ := s.GetMember(ctx, child.ID)
	if got.Contact.City == nil || *got.Contact.City != "Kiel" || *got.Contact.Email != "ole@example.org" {
		t.Errorf("child contact = %+v", got.Contact)
	}
	if got.Bank.IBAN == nil || *got.Bank.IBAN != "DE02120300000000202051" {
		t.Errorf("child bank = %+v", got.Bank)
	}
	got, _ = s.GetMember(ctx, baby.ID)
	if got.Contact.Street == nil || *got.Contact.Street != "Deich 1" || got.Contact.Email != nil {
		t.Errorf("baby contact = %+v", got.Contact)
	}

	// A change of one member applies to all.
	in := MemberInput{Number: 2, FirstName: "Ole", LastName: "Kite",
		Contact: MemberContact{Email: ptr("ole@example.org"), Street: ptr("Strand 5"), City: ptr("Sylt")},
		Bank:    MemberBank{AccountHolder: ptr("Pia Kite"), IBAN: ptr("DE89370400440532013000")}}
	if _, err := s.UpdateMember(ctx, child.ID, in); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetMember(ctx, parent.ID)
	if *got.Contact.City != "Sylt" || *got.Contact.Email != "pia@example.org" || *got.Bank.IBAN != "DE89370400440532013000" {
		t.Errorf("parent after child update = %+v %+v", got.Contact, got.Bank)
	}
	if got, _ := s.GetMember(ctx, other.ID); got.Contact.City != nil {
		t.Errorf("member outside the household changed: %+v", got.Contact)
	}

	// A member belongs to at most one household.
	_, err = s.CreateHousehold(ctx, HouseholdInput{Name: "Andere", MemberIDs: []string{other.ID, child.ID}})
	if !errors.Is(err, ErrMemberInOtherHousehold) {
		t.Errorf("second household: %v, want ErrMemberInOtherHousehold", err)
	}

	if p, err := s.HouseholdPayer(ctx, baby.ID); err != nil || p == nil || p.ID != parent.ID {
		t.Errorf("HouseholdPayer = %+v, %v", p, err)
	}
	if p, err := s.HouseholdPayer(ctx, other.ID); err != nil || p != nil {
		t.Errorf("HouseholdPayer of other = %+v, %v", p, err)
	}

	now := time.Now()
	if h.PayerMissing(now) {
		t.Error("payer missing while the payer is active")
	}
	// When the payer leaves, the remaining members are flagged.
	if err := s.DeleteMember(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	h, _ = s.GetHousehold(ctx, h.ID)
	if h.Payer != nil || len(h.Members) != 2 || !h.PayerMissing(now) {
		t.Errorf("after payer deletion = %+v", h)
	}

	// Removing a member from the household frees it for another one.
	if _, err := s.UpdateHousehold(ctx, h.ID, HouseholdInput{Name: "Familie Kite", PayerID: &child.ID,
		MemberIDs: []string{child.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateHousehold(ctx, HouseholdInput{Name: "Andere", MemberIDs: []string{other.ID, baby.ID}}); err != nil {
		t.Errorf("baby could not join another household: %v", err)
	}
}

func TestHouseholdPayerExit(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	payer := HouseholdMember{MemberRef: MemberRef{ID: "p"}}
	kid := HouseholdMember{MemberRef: MemberRef{ID: "k"}}
	h := Household{Payer: &payer.MemberRef, Members: []HouseholdMember{payer, kid}}
	if h.PayerMissing(now) {
		t.Error("active payer flagged")
	}
	h.Members[0].ExitDate = date("2026-06-01")
	if !h.PayerMissing(now) {
		t.Error("payer who left today not flagged")
	}
	h.Members[1].ExitDate = date("2026-05-01")
	if h.PayerMissing(now) {
		t.Error("flagged although no active member remains")
	}
}

func TestGuardians(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	kid, _ := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "Ida", LastName: "Kite"})
	mum, _ := s.CreateMember(ctx, MemberInput{Number: 2, FirstName: "Pia", LastName: "Kite"})

	if _, err := s.CreateGuardian(ctx, kid.ID, GuardianInput{GuardianMemberID: &mum.ID, Relation: ptr("Mutter")}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateGuardian(ctx, kid.ID, GuardianInput{Name: ptr("Jan Kite"), Phone: ptr("0170 1234")}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateGuardian(ctx, kid.ID, GuardianInput{GuardianMemberID: &kid.ID}); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("self as guardian: %v", err)
	}
	list, err := s.ListGuardians(ctx, kid.ID)
	if err != nil || len(list) != 2 || list[0].Member == nil || list[0].Member.ID != mum.ID || *list[1].Name != "Jan Kite" {
		t.Fatalf("guardians = %+v, %v", list, err)
	}
	if err := s.DeleteMember(ctx, mum.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListGuardians(ctx, kid.ID); len(list) != 1 {
		t.Errorf("guardian entry of deleted member kept: %+v", list)
	}
}
