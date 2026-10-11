package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

func newTestStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	conn, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn), conn
}

func ptr[T any](v T) *T { return &v }

func date(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestMemberCRUD(t *testing.T) {
	ctx := context.Background()
	s, conn := newTestStore(t)

	role, err := s.CreateNamed(ctx, Roles, "Trainer")
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CreateNamed(ctx, Groups, "Jugend")
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateMember(ctx, MemberInput{
		Number: 1, FirstName: "Ada", LastName: "Lovelace", Birthday: date("1815-12-10"),
		Contact: MemberContact{Email: ptr("ada@example.org"), City: ptr("London")},
		Bank:    MemberBank{IBAN: ptr("DE02120300000000202051"), SEPAMandateDate: date("2024-01-15")},
		RoleIDs: &[]string{role.ID}, GroupIDs: &[]string{group.ID},
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	if len(created.ID) != 36 || created.ID[14] != '7' {
		t.Errorf("id %q is not a UUIDv7", created.ID)
	}

	got, err := s.GetMember(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.FirstName != "Ada" || *got.Contact.Email != "ada@example.org" || *got.Contact.City != "London" {
		t.Errorf("core/contact not round-tripped: %+v", got)
	}
	if *got.Bank.IBAN != "DE02120300000000202051" || !got.Bank.SEPAMandateDate.Equal(*date("2024-01-15")) {
		t.Errorf("bank details not round-tripped: %+v", got.Bank)
	}
	if !got.Birthday.Equal(*date("1815-12-10")) {
		t.Errorf("birthday = %v", got.Birthday)
	}
	if len(got.Roles) != 1 || got.Roles[0].Name != "Trainer" || len(got.Groups) != 1 || len(got.Sections) != 0 {
		t.Errorf("links = roles %v groups %v sections %v", got.Roles, got.Groups, got.Sections)
	}

	// The IBAN lives only in member_bank_details.
	var iban string
	if err := conn.QueryRow(`SELECT iban FROM member_bank_details WHERE member_id = ?`, created.ID).Scan(&iban); err != nil {
		t.Errorf("bank row: %v", err)
	}

	// Update: drop bank details, change groups, leave roles untouched (nil).
	updated, err := s.UpdateMember(ctx, created.ID, MemberInput{
		Number: 1, FirstName: "Augusta Ada", LastName: "King",
		Contact:  MemberContact{Email: ptr("ada@example.org")},
		GroupIDs: &[]string{},
	})
	if err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if updated.FirstName != "Augusta Ada" || updated.Bank.IBAN != nil || updated.Contact.City != nil {
		t.Errorf("after update: %+v", updated)
	}
	if len(updated.Roles) != 1 || len(updated.Groups) != 0 {
		t.Errorf("after update links: roles %v groups %v", updated.Roles, updated.Groups)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("updated_at went backwards")
	}
	var bankDeleted sql.NullString
	conn.QueryRow(`SELECT deleted_at FROM member_bank_details WHERE member_id = ?`, created.ID).Scan(&bankDeleted)
	if !bankDeleted.Valid {
		t.Error("removing bank details should tombstone the member_bank_details row")
	}

	// Re-adding a group revives the tombstoned link instead of duplicating it.
	if _, err := s.UpdateMember(ctx, created.ID, MemberInput{
		Number: 1, FirstName: "Ada", LastName: "King", GroupIDs: &[]string{group.ID},
	}); err != nil {
		t.Fatal(err)
	}
	var linkRows int
	conn.QueryRow(`SELECT COUNT(*) FROM member_groups WHERE member_id = ? AND deleted_at IS NULL`, created.ID).Scan(&linkRows)
	if linkRows != 1 {
		t.Errorf("live member_groups rows = %d, want 1", linkRows)
	}

	list, total, err := s.ListMembers(ctx, MemberFilter{Limit: 10})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("ListMembers = %d/%d, %v", len(list), total, err)
	}

	if err := s.DeleteMember(ctx, created.ID); err != nil {
		t.Fatalf("DeleteMember: %v", err)
	}
	if _, err := s.GetMember(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetMember after delete: %v, want ErrNotFound", err)
	}
	if err := s.DeleteMember(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v, want ErrNotFound", err)
	}
	// Tombstoned, not removed, together with its parts.
	var deletedAt sql.NullString
	conn.QueryRow(`SELECT deleted_at FROM members WHERE id = ?`, created.ID).Scan(&deletedAt)
	if !deletedAt.Valid {
		t.Error("member row should be kept as tombstone")
	}
	var liveParts int
	conn.QueryRow(`SELECT (SELECT COUNT(*) FROM member_contacts WHERE member_id = ?1 AND deleted_at IS NULL)
		+ (SELECT COUNT(*) FROM member_roles WHERE member_id = ?1 AND deleted_at IS NULL)`, created.ID).Scan(&liveParts)
	if liveParts != 0 {
		t.Errorf("%d live contact/link rows remain after delete", liveParts)
	}
}

func TestMemberUniqueness(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)

	first, err := s.CreateMember(ctx, MemberInput{Number: 7, FirstName: "A", LastName: "A", Contact: MemberContact{Email: ptr("a@example.org")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateMember(ctx, MemberInput{Number: 7, FirstName: "B", LastName: "B"}); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate number: %v, want ErrConflict", err)
	}
	if _, err := s.CreateMember(ctx, MemberInput{Number: 8, FirstName: "B", LastName: "B", Contact: MemberContact{Email: ptr("a@example.org")}}); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate email: %v, want ErrConflict", err)
	}

	// A tombstone does not block re-use of its number or e-mail.
	if err := s.DeleteMember(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateMember(ctx, MemberInput{Number: 7, FirstName: "C", LastName: "C", Contact: MemberContact{Email: ptr("a@example.org")}}); err != nil {
		t.Errorf("re-use after delete: %v", err)
	}
}

func TestListMembersSearchAndPaging(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	for i, name := range []string{"Zuse", "Hopper", "Lovelace", "Turing"} {
		email := name + "@example.org"
		if _, err := s.CreateMember(ctx, MemberInput{Number: 10 + i, FirstName: "X", LastName: name, Contact: MemberContact{Email: &email}}); err != nil {
			t.Fatal(err)
		}
	}

	page, total, err := s.ListMembers(ctx, MemberFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(page) != 2 || page[0].LastName != "Turing" || page[1].LastName != "Zuse" {
		t.Errorf("page 2 = %d of %d: %v", len(page), total, page)
	}

	found, total, err := s.ListMembers(ctx, MemberFilter{Search: "hopper@"})
	if err != nil || total != 1 || len(found) != 1 || found[0].LastName != "Hopper" {
		t.Errorf("search by email = %v (%d, %v)", found, total, err)
	}
	found, _, _ = s.ListMembers(ctx, MemberFilter{Search: "12"})
	if len(found) != 1 || found[0].LastName != "Lovelace" {
		t.Errorf("search by number = %v", found)
	}
}

func TestNamedEntities(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)

	sec, err := s.CreateNamed(ctx, Sections, "Fußball")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateNamed(ctx, Sections, "Fußball"); !errors.Is(err, ErrConflict) {
		t.Errorf("duplicate name: %v, want ErrConflict", err)
	}
	m, _ := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "A", LastName: "A"})
	if err := s.SetNamedMembers(ctx, Sections, sec.ID, []string{m.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetNamedMembers(ctx, Sections, sec.ID, []string{"no-such-member"}); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("link to unknown member: %v, want ErrInvalidReference", err)
	}
	members, err := s.NamedMembers(ctx, Sections, sec.ID)
	if err != nil || len(members) != 1 || members[0].ID != m.ID {
		t.Errorf("NamedMembers = %v, %v", members, err)
	}

	if err := s.DeleteNamed(ctx, Sections, sec.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetMember(ctx, m.ID)
	if len(got.Sections) != 0 {
		t.Errorf("deleted section still linked: %v", got.Sections)
	}
	if _, err := s.CreateNamed(ctx, Sections, "Fußball"); err != nil {
		t.Errorf("re-create after delete: %v", err)
	}
	if _, err := s.RenameNamed(ctx, Sections, sec.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("rename tombstoned: %v, want ErrNotFound", err)
	}
}

func TestFinanceAmountsAndReferences(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)

	tx, err := s.CreateTransaction(ctx, TransactionInput{Date: *date("2025-03-01"), Description: "Spende", Amount: 19.99, Type: "income"})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Amount != 19.99 {
		t.Errorf("amount = %v, want 19.99", tx.Amount)
	}

	if _, err := s.CreateFee(ctx, FeeInput{MemberID: "missing", Amount: 1, DueDate: *date("2025-01-01"), Year: 2025}); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("fee for unknown member: %v, want ErrInvalidReference", err)
	}
	m, _ := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "A", LastName: "A"})
	fee, err := s.CreateFee(ctx, FeeInput{MemberID: m.ID, Amount: 60, DueDate: *date("2025-01-01"), Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if fee.Member.ID != m.ID || fee.Amount != 60 {
		t.Errorf("fee = %+v", fee)
	}

	// Deleting the member tombstones its fees.
	if err := s.DeleteMember(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if fees, _ := s.ListFees(ctx); len(fees) != 0 {
		t.Errorf("fees of deleted member still listed: %v", fees)
	}

	nFees, nTx, err := s.ResetFinance(ctx)
	if err != nil || nFees != 0 || nTx != 1 {
		t.Errorf("ResetFinance = %d, %d, %v", nFees, nTx, err)
	}
}
