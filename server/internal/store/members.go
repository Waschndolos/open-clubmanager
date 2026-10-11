package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Member is a club member assembled from members (scope members.core),
// member_contacts (members.contact) and member_bank_details (members.bank).
type Member struct {
	ID        string
	Number    int
	FirstName string
	LastName  string
	Birthday  *time.Time
	EntryDate *time.Time
	ExitDate  *time.Time
	Comment   *string
	Contact   MemberContact
	Bank      MemberBank
	Roles     []NamedEntity
	Groups    []NamedEntity
	Sections  []NamedEntity
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MemberContact is the members.contact part of a member.
type MemberContact struct {
	Email       *string
	Phone       *string
	PhoneMobile *string
	Street      *string
	PostalCode  *string
	City        *string
	State       *string
}

func (c MemberContact) empty() bool {
	for _, v := range []*string{c.Email, c.Phone, c.PhoneMobile, c.Street, c.PostalCode, c.City, c.State} {
		if v != nil && *v != "" {
			return false
		}
	}
	return true
}

// MemberBank is the members.bank part of a member.
type MemberBank struct {
	AccountHolder   *string
	IBAN            *string
	BIC             *string
	BankName        *string
	SEPAMandateDate *time.Time
}

func (b MemberBank) empty() bool {
	for _, v := range []*string{b.AccountHolder, b.IBAN, b.BIC, b.BankName} {
		if v != nil && *v != "" {
			return false
		}
	}
	return b.SEPAMandateDate == nil
}

// MemberInput holds the writable fields of a member. A nil id slice leaves the
// corresponding links unchanged on update.
type MemberInput struct {
	Number     int
	FirstName  string
	LastName   string
	Birthday   *time.Time
	EntryDate  *time.Time
	ExitDate   *time.Time
	Comment    *string
	Contact    MemberContact
	Bank       MemberBank
	RoleIDs    *[]string
	GroupIDs   *[]string
	SectionIDs *[]string
}

// MemberSummary is the short form of a member used in lists and references.
type MemberSummary struct {
	ID        string
	Number    int
	FirstName string
	LastName  string
}

// MemberFilter selects a page of members.
type MemberFilter struct {
	// Search matches first name, last name, e-mail and member number.
	Search string
	Limit  int
	Offset int
}

const memberSelect = `
SELECT m.id, m.number, m.first_name, m.last_name, m.birthday, m.entry_date, m.exit_date, m.comment,
       m.created_at, m.updated_at,
       c.email, c.phone, c.phone_mobile, c.street, c.postal_code, c.city, c.state,
       b.account_holder, b.iban, b.bic, b.bank_name, b.sepa_mandate_date
FROM members m
LEFT JOIN member_contacts c ON c.member_id = m.id AND c.deleted_at IS NULL
LEFT JOIN member_bank_details b ON b.member_id = m.id AND b.deleted_at IS NULL
WHERE m.deleted_at IS NULL`

type scanner interface {
	Scan(dest ...any) error
}

func scanMember(row scanner) (Member, error) {
	var m Member
	var birthday, entryDate, exitDate, comment sql.NullString
	var createdAt, updatedAt string
	var email, phone, phoneMobile, street, postalCode, city, state sql.NullString
	var holder, iban, bic, bankName, mandate sql.NullString
	err := row.Scan(&m.ID, &m.Number, &m.FirstName, &m.LastName, &birthday, &entryDate, &exitDate, &comment,
		&createdAt, &updatedAt,
		&email, &phone, &phoneMobile, &street, &postalCode, &city, &state,
		&holder, &iban, &bic, &bankName, &mandate)
	if err != nil {
		return m, mapErr(err)
	}
	m.Birthday, m.EntryDate, m.ExitDate = nullTime(birthday), nullTime(entryDate), nullTime(exitDate)
	m.Comment = nullStr(comment)
	m.CreatedAt, m.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
	m.Contact = MemberContact{
		Email: nullStr(email), Phone: nullStr(phone), PhoneMobile: nullStr(phoneMobile),
		Street: nullStr(street), PostalCode: nullStr(postalCode), City: nullStr(city), State: nullStr(state),
	}
	m.Bank = MemberBank{
		AccountHolder: nullStr(holder), IBAN: nullStr(iban), BIC: nullStr(bic), BankName: nullStr(bankName),
		SEPAMandateDate: nullTime(mandate),
	}
	m.Roles, m.Groups, m.Sections = []NamedEntity{}, []NamedEntity{}, []NamedEntity{}
	return m, nil
}

// ListMembers returns a page of live members ordered by name, and the total
// number of matches.
func (s *Store) ListMembers(ctx context.Context, f MemberFilter) ([]Member, int, error) {
	where := ""
	var args []any
	if f.Search != "" {
		like := "%" + f.Search + "%"
		where = ` AND (m.first_name LIKE ? OR m.last_name LIKE ? OR c.email LIKE ? OR CAST(m.number AS TEXT) LIKE ?)`
		args = []any{like, like, like, like}
	}

	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (`+memberSelect+where+`)`, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := memberSelect + where + ` ORDER BY m.last_name, m.first_name, m.number`
	if f.Limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, f.Limit, f.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	members := []Member{}
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		members = append(members, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := s.loadMemberLinks(ctx, members); err != nil {
		return nil, 0, err
	}
	return members, total, nil
}

// GetMember returns a live member with its links.
func (s *Store) GetMember(ctx context.Context, id string) (Member, error) {
	m, err := scanMember(s.db.QueryRowContext(ctx, memberSelect+` AND m.id = ?`, id))
	if err != nil {
		return m, err
	}
	list := []Member{m}
	if err := s.loadMemberLinks(ctx, list); err != nil {
		return m, err
	}
	return list[0], nil
}

// loadMemberLinks fills Roles, Groups and Sections of the given members.
func (s *Store) loadMemberLinks(ctx context.Context, members []Member) error {
	if len(members) == 0 {
		return nil
	}
	index := make(map[string]int, len(members))
	ids := make([]string, len(members))
	for i, m := range members {
		index[m.ID] = i
		ids[i] = m.ID
	}
	for _, k := range []NamedKind{Roles, Groups, Sections} {
		rows, err := s.db.QueryContext(ctx,
			`SELECT l.member_id, e.id, e.name FROM `+k.linkTable+` l
			 JOIN `+k.table+` e ON e.id = l.`+k.linkCol+` AND e.deleted_at IS NULL
			 WHERE l.deleted_at IS NULL AND l.member_id IN (`+placeholders(len(ids))+`)
			 ORDER BY e.name`, stringArgs(ids)...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var memberID string
			var e NamedEntity
			if err := rows.Scan(&memberID, &e.ID, &e.Name); err != nil {
				rows.Close()
				return err
			}
			m := &members[index[memberID]]
			switch k {
			case Roles:
				m.Roles = append(m.Roles, e)
			case Groups:
				m.Groups = append(m.Groups, e)
			case Sections:
				m.Sections = append(m.Sections, e)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

// CreateMember inserts a member with its contact data, bank details and links.
func (s *Store) CreateMember(ctx context.Context, in MemberInput) (Member, error) {
	id := db.NewID()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		_, err := tx.ExecContext(ctx,
			`INSERT INTO members (id, number, first_name, last_name, birthday, entry_date, exit_date, comment, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, in.Number, in.FirstName, in.LastName, timeArg(in.Birthday), timeArg(in.EntryDate), timeArg(in.ExitDate),
			strArg(in.Comment), now, now)
		if err != nil {
			return mapErr(err)
		}
		return writeMemberParts(ctx, tx, id, in, now)
	})
	if err != nil {
		return Member{}, err
	}
	return s.GetMember(ctx, id)
}

// UpdateMember replaces the writable fields of a live member.
func (s *Store) UpdateMember(ctx context.Context, id string, in MemberInput) (Member, error) {
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE members SET number = ?, first_name = ?, last_name = ?, birthday = ?, entry_date = ?, exit_date = ?,
			        comment = ?, updated_at = ?
			 WHERE id = ? AND deleted_at IS NULL`,
			in.Number, in.FirstName, in.LastName, timeArg(in.Birthday), timeArg(in.EntryDate), timeArg(in.ExitDate),
			strArg(in.Comment), now, id))
		if err != nil {
			return err
		}
		return writeMemberParts(ctx, tx, id, in, now)
	})
	if err != nil {
		return Member{}, err
	}
	return s.GetMember(ctx, id)
}

// writeMemberParts stores the contact/bank rows and links of a member. An
// empty contact or bank part tombstones an existing row instead of keeping a
// row full of NULLs, so scopes without data stay empty.
func writeMemberParts(ctx context.Context, tx *sql.Tx, id string, in MemberInput, now string) error {
	c := in.Contact
	if c.empty() {
		if _, err := tx.ExecContext(ctx, `UPDATE member_contacts SET deleted_at = ?, updated_at = ? WHERE member_id = ? AND deleted_at IS NULL`, now, now, id); err != nil {
			return err
		}
	} else {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO member_contacts (member_id, email, phone, phone_mobile, street, postal_code, city, state, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (member_id) DO UPDATE SET email = excluded.email, phone = excluded.phone,
			   phone_mobile = excluded.phone_mobile, street = excluded.street, postal_code = excluded.postal_code,
			   city = excluded.city, state = excluded.state, updated_at = excluded.updated_at, deleted_at = NULL`,
			id, strArg(c.Email), strArg(c.Phone), strArg(c.PhoneMobile), strArg(c.Street), strArg(c.PostalCode),
			strArg(c.City), strArg(c.State), now)
		if err != nil {
			return mapErr(err)
		}
	}

	b := in.Bank
	if b.empty() {
		if _, err := tx.ExecContext(ctx, `UPDATE member_bank_details SET deleted_at = ?, updated_at = ? WHERE member_id = ? AND deleted_at IS NULL`, now, now, id); err != nil {
			return err
		}
	} else {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO member_bank_details (member_id, account_holder, iban, bic, bank_name, sepa_mandate_date, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (member_id) DO UPDATE SET account_holder = excluded.account_holder, iban = excluded.iban,
			   bic = excluded.bic, bank_name = excluded.bank_name, sepa_mandate_date = excluded.sepa_mandate_date,
			   updated_at = excluded.updated_at, deleted_at = NULL`,
			id, strArg(b.AccountHolder), strArg(b.IBAN), strArg(b.BIC), strArg(b.BankName), timeArg(b.SEPAMandateDate), now)
		if err != nil {
			return mapErr(err)
		}
	}

	for _, l := range []struct {
		kind NamedKind
		ids  *[]string
	}{{Roles, in.RoleIDs}, {Groups, in.GroupIDs}, {Sections, in.SectionIDs}} {
		if l.ids == nil {
			continue
		}
		if err := setLinks(ctx, tx, l.kind.linkTable, "member_id", id, l.kind.linkCol, *l.ids, now); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMember tombstones a member together with everything that belongs to
// it: contact data, bank details, links, fees and inventory loans.
func (s *Store) DeleteMember(ctx context.Context, id string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		now := db.Now()
		err := requireAffected(tx.ExecContext(ctx,
			`UPDATE members SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now, now, id))
		if err != nil {
			return err
		}
		for _, table := range []string{"member_contacts", "member_bank_details", "member_roles", "member_groups",
			"member_sections", "member_fees", "inventory_loans"} {
			if _, err := tx.ExecContext(ctx,
				`UPDATE `+table+` SET deleted_at = ?, updated_at = ? WHERE member_id = ? AND deleted_at IS NULL`,
				now, now, id); err != nil {
				return err
			}
		}
		return nil
	})
}
