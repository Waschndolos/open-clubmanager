package store

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// ErrMemberInOtherHousehold is returned when a member that already belongs to
// another household is added to a household.
var ErrMemberInOtherHousehold = errors.New("member belongs to another household")

// Household is a family whose members share a main payer and, optionally,
// the address and bank details.
type Household struct {
	ID           string
	Name         string
	Payer        *MemberRef
	ShareAddress bool
	ShareBank    bool
	Members      []HouseholdMember
}

// HouseholdMember is a member of a household.
type HouseholdMember struct {
	MemberRef
	Birthday *time.Time
	ExitDate *time.Time
}

// HouseholdInput is the writable part of a household.
type HouseholdInput struct {
	Name         string
	PayerID      *string
	ShareAddress bool
	ShareBank    bool
	MemberIDs    []string
}

// Active reports whether the member has not left the club as of now.
func (m HouseholdMember) Active(now time.Time) bool {
	return m.ExitDate == nil || dateOnly(*m.ExitDate).After(dateOnly(now))
}

// PayerMissing reports whether active members remain without an active main
// payer, e.g. after the payer left the club.
func (h Household) PayerMissing(now time.Time) bool {
	payerActive, others := false, 0
	for _, m := range h.Members {
		if !m.Active(now) {
			continue
		}
		if h.Payer != nil && m.ID == h.Payer.ID {
			payerActive = true
		} else {
			others++
		}
	}
	return !payerActive && others > 0
}

// ListHouseholds returns all live households ordered by name.
func (s *Store) ListHouseholds(ctx context.Context) ([]Household, error) {
	return s.queryHouseholds(ctx, "")
}

// GetHousehold returns a live household.
func (s *Store) GetHousehold(ctx context.Context, id string) (Household, error) {
	list, err := s.queryHouseholds(ctx, id)
	if err != nil {
		return Household{}, err
	}
	if len(list) == 0 {
		return Household{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) queryHouseholds(ctx context.Context, id string) ([]Household, error) {
	query := `SELECT h.id, h.name, h.payer_member_id, h.share_address, h.share_bank FROM households h WHERE h.deleted_at IS NULL`
	var args []any
	if id != "" {
		query += ` AND h.id = ?`
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY h.name, h.id`, args...)
	if err != nil {
		return nil, err
	}
	result := []Household{}
	payers := map[string]string{}
	index := map[string]int{}
	for rows.Next() {
		var h Household
		var payer sql.NullString
		var addr, bank int
		if err := rows.Scan(&h.ID, &h.Name, &payer, &addr, &bank); err != nil {
			rows.Close()
			return nil, err
		}
		h.ShareAddress, h.ShareBank, h.Members = addr != 0, bank != 0, []HouseholdMember{}
		if payer.Valid {
			payers[h.ID] = payer.String
		}
		index[h.ID] = len(result)
		result = append(result, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(result) == 0 {
		return result, err
	}

	query = `SELECT hm.household_id, m.id, m.number, m.first_name, m.last_name, m.birthday, m.exit_date
		FROM household_members hm
		JOIN members m ON m.id = hm.member_id AND m.deleted_at IS NULL
		JOIN households h ON h.id = hm.household_id AND h.deleted_at IS NULL
		WHERE hm.deleted_at IS NULL`
	if id != "" {
		query += ` AND hm.household_id = ?`
	}
	rows, err = s.db.QueryContext(ctx, query+` ORDER BY m.last_name, m.first_name, m.number`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var hid string
		var m HouseholdMember
		var birthday, exit sql.NullString
		if err := rows.Scan(&hid, &m.ID, &m.Number, &m.FirstName, &m.LastName, &birthday, &exit); err != nil {
			return nil, err
		}
		m.Birthday, m.ExitDate = nullTime(birthday), nullTime(exit)
		h := &result[index[hid]]
		h.Members = append(h.Members, m)
		if payers[hid] == m.ID {
			ref := m.MemberRef
			h.Payer = &ref
		}
	}
	return result, rows.Err()
}

// CreateHousehold inserts a household with its members.
func (s *Store) CreateHousehold(ctx context.Context, in HouseholdInput) (Household, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := w.insert("households", householdColumns(in, row{"id": id, "created_at": db.Now()})); err != nil {
			return err
		}
		return writeHouseholdMembers(w, id, in)
	})
	if err != nil {
		return Household{}, err
	}
	return s.GetHousehold(ctx, id)
}

// UpdateHousehold replaces the fields and members of a live household.
func (s *Store) UpdateHousehold(ctx context.Context, id string, in HouseholdInput) (Household, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update("households", `id = ? AND deleted_at IS NULL`, []any{id}, householdColumns(in, row{}))); err != nil {
			return err
		}
		return writeHouseholdMembers(w, id, in)
	})
	if err != nil {
		return Household{}, err
	}
	return s.GetHousehold(ctx, id)
}

func householdColumns(in HouseholdInput, values row) row {
	bool2int := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	values["name"], values["payer_member_id"] = in.Name, strArg(in.PayerID)
	values["share_address"], values["share_bank"] = bool2int(in.ShareAddress), bool2int(in.ShareBank)
	return values
}

// writeHouseholdMembers makes the members of the household exactly
// in.MemberIDs and aligns the shared address and bank details with those of
// the payer (or the first member when there is no payer).
func writeHouseholdMembers(w *writer, id string, in HouseholdInput) error {
	if in.PayerID != nil && !slices.Contains(in.MemberIDs, *in.PayerID) {
		return ErrInvalidReference
	}
	for _, m := range in.MemberIDs {
		if err := requireLiveMember(w.ctx, w.Tx, m); err != nil {
			return err
		}
		var other string
		err := w.QueryRowContext(w.ctx, `SELECT hm.household_id FROM household_members hm
			JOIN households h ON h.id = hm.household_id AND h.deleted_at IS NULL
			WHERE hm.member_id = ? AND hm.deleted_at IS NULL AND hm.household_id != ?`, m, id).Scan(&other)
		switch {
		case err == nil:
			return ErrMemberInOtherHousehold
		case err != sql.ErrNoRows:
			return err
		}
	}

	now := db.Now()
	where, args := `household_id = ? AND deleted_at IS NULL`, []any{id}
	if len(in.MemberIDs) > 0 {
		where += ` AND member_id NOT IN (` + placeholders(len(in.MemberIDs)) + `)`
		args = append(args, stringArgs(in.MemberIDs)...)
	}
	if _, err := w.update("household_members", where, args, row{"deleted_at": now}); err != nil {
		return err
	}
	for _, m := range in.MemberIDs {
		if err := w.upsert("household_members", row{"member_id": m}, row{"household_id": id, "deleted_at": nil}); err != nil {
			return err
		}
	}

	if len(in.MemberIDs) == 0 {
		return nil
	}
	source := in.MemberIDs[0]
	if in.PayerID != nil {
		source = *in.PayerID
	}
	return shareHouseholdData(w, source, now)
}

// DeleteHousehold tombstones a household and its memberships; the members
// themselves stay.
func (s *Store) DeleteHousehold(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update("households", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		_, err := w.update("household_members", `household_id = ? AND deleted_at IS NULL`, []any{id}, tombstone)
		return err
	})
}

// HouseholdPayer returns the main payer of the household of a member, or nil
// when the member belongs to no household or the household has no payer.
// Direct debits and payment matching charge the payer instead of the member.
func (s *Store) HouseholdPayer(ctx context.Context, memberID string) (*MemberRef, error) {
	var p MemberRef
	err := s.db.QueryRowContext(ctx, `SELECT p.id, p.number, p.first_name, p.last_name
		FROM household_members hm
		JOIN households h ON h.id = hm.household_id AND h.deleted_at IS NULL
		JOIN members p ON p.id = h.payer_member_id AND p.deleted_at IS NULL
		WHERE hm.member_id = ? AND hm.deleted_at IS NULL`, memberID).Scan(&p.ID, &p.Number, &p.FirstName, &p.LastName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// shareHouseholdData copies the address and/or bank details of source to the
// other members of its household, as far as the household shares them.
func shareHouseholdData(w *writer, source, now string) error {
	var hid string
	var addr, bank int
	err := w.QueryRowContext(w.ctx, `SELECT h.id, h.share_address, h.share_bank FROM household_members hm
		JOIN households h ON h.id = hm.household_id AND h.deleted_at IS NULL
		WHERE hm.member_id = ? AND hm.deleted_at IS NULL`, source).Scan(&hid, &addr, &bank)
	if err == sql.ErrNoRows || (err == nil && addr == 0 && bank == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	others, err := w.QueryContext(w.ctx, `SELECT hm.member_id FROM household_members hm
		JOIN members m ON m.id = hm.member_id AND m.deleted_at IS NULL
		WHERE hm.household_id = ? AND hm.deleted_at IS NULL AND hm.member_id != ?`, hid, source)
	if err != nil {
		return err
	}
	var targets []string
	for others.Next() {
		var id string
		if err := others.Scan(&id); err != nil {
			others.Close()
			return err
		}
		targets = append(targets, id)
	}
	others.Close()
	if err := others.Err(); err != nil {
		return err
	}
	sort.Strings(targets)

	if addr != 0 {
		address, err := liveColumns(w, "member_contacts", source, "street", "postal_code", "city", "state")
		if err != nil {
			return err
		}
		for _, t := range targets {
			if err := copyAddress(w, t, address); err != nil {
				return err
			}
		}
	}
	if bank != 0 {
		details, err := liveColumns(w, "member_bank_details", source,
			"account_holder", "iban", "bic", "bank_name", "sepa_mandate_date")
		if err != nil {
			return err
		}
		for _, t := range targets {
			if err := copyBank(w, t, details, now); err != nil {
				return err
			}
		}
	}
	return nil
}

// liveColumns reads the given columns of the live row of a member_id keyed
// table; all values are nil when there is no live row.
func liveColumns(w *writer, table, memberID string, cols ...string) (row, error) {
	dest := make([]sql.NullString, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	query := `SELECT `
	for i, c := range cols {
		if i > 0 {
			query += `, `
		}
		query += c
	}
	err := w.QueryRowContext(w.ctx, query+` FROM `+table+` WHERE member_id = ? AND deleted_at IS NULL`, memberID).Scan(ptrs...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	values := row{}
	for i, c := range cols {
		values[c] = nil
		if dest[i].Valid {
			values[c] = dest[i].String
		}
	}
	return values, nil
}

func allNil(values row) bool {
	for _, v := range values {
		if v != nil && v != "" {
			return false
		}
	}
	return true
}

// copyAddress sets the address of a member. A missing contact row is only
// created when there is an address to store, without reviving old values of a
// tombstoned row.
func copyAddress(w *writer, memberID string, address row) error {
	n, err := w.update("member_contacts", `member_id = ? AND deleted_at IS NULL`, []any{memberID}, address)
	if err != nil || n > 0 || allNil(address) {
		return err
	}
	set := row{"email": nil, "phone": nil, "phone_mobile": nil, "deleted_at": nil}
	for k, v := range address {
		set[k] = v
	}
	return w.upsert("member_contacts", row{"member_id": memberID}, set)
}

// copyBank replaces the bank details of a member; empty details remove them.
func copyBank(w *writer, memberID string, details row, now string) error {
	if allNil(details) {
		_, err := w.update("member_bank_details", `member_id = ? AND deleted_at IS NULL`, []any{memberID}, row{"deleted_at": now})
		return err
	}
	set := row{"deleted_at": nil}
	for k, v := range details {
		set[k] = v
	}
	return w.upsert("member_bank_details", row{"member_id": memberID}, set)
}
