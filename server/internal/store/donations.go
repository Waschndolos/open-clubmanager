package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/donation"
)

// donationSettingsID is the primary key of the single donation_settings row.
const donationSettingsID = "club"

var (
	// ErrDonationsNotConfigured is returned when a receipt is created before
	// the club's exemption data has been set up.
	ErrDonationsNotConfigured = errors.New("donation receipts not configured")
	// ErrDonationCertified is returned when a donation that is certified by a
	// receipt is changed or unassigned.
	ErrDonationCertified = errors.New("donation is certified by a receipt")
	// ErrNotIncome is returned when an expense is marked as a donation.
	ErrNotIncome = errors.New("transaction is not an income")
	// ErrDonorInUse is returned when a donor with donations is deleted.
	ErrDonorInUse = errors.New("donor has donations")
	// ErrReceiptVoided is returned when a voided receipt is voided again or
	// repeated as a duplicate.
	ErrReceiptVoided = errors.New("receipt is voided")
	// ErrMissingAddress is returned when a receipt is created for a donor
	// without a postal address.
	ErrMissingAddress = errors.New("donor has no address")
)

// ReceiptItemsError lists the items of a requested receipt that cannot be
// certified: not a donation or paid fee of the donor, not in one calendar
// year, or (Certified) already on another receipt that has not been voided.
type ReceiptItemsError struct {
	Invalid   []string
	Certified []string
}

func (e *ReceiptItemsError) Error() string {
	return fmt.Sprintf("%d items invalid, %d already certified", len(e.Invalid), len(e.Certified))
}

// Donor is a donor who is not a member (scope finance).
type Donor struct {
	ID         string
	Name       string
	Street     *string
	PostalCode *string
	City       *string
	Notes      *string
}

// DonorInput holds the writable fields of a donor.
type DonorInput struct {
	Name       string
	Street     *string
	PostalCode *string
	City       *string
	Notes      *string
}

// DonationEntry is an income in the cash book that is, or looks like, a
// donation: assigned to a member or donor, or with a donation category.
type DonationEntry struct {
	TransactionID string
	Date          time.Time
	Description   string
	AmountCents   int64
	Category      *string
	Member        *MemberRef
	Donor         *Donor
	ExpenseWaiver bool
	// ReceiptNumber is the number of the receipt certifying it, if any.
	ReceiptNumber *string
}

// Assigned reports whether the entry has a member or donor.
func (e DonationEntry) Assigned() bool { return e.Member != nil || e.Donor != nil }

// DonationCandidate is a member or donor with the donations and paid fees of
// a year that can appear on a receipt.
type DonationCandidate struct {
	Member  *MemberRef
	Donor   *Donor
	Name    string
	Address string
	Items   []CandidateItem
}

// Key identifies the candidate: "member:<id>" or "donor:<id>".
func (c DonationCandidate) Key() string {
	if c.Member != nil {
		return "member:" + c.Member.ID
	}
	return "donor:" + c.Donor.ID
}

// Open lists the items not certified yet.
func (c DonationCandidate) Open() []CandidateItem {
	var out []CandidateItem
	for _, it := range c.Items {
		if it.ReceiptID == nil {
			out = append(out, it)
		}
	}
	return out
}

// CandidateItem is a donation (TransactionID) or paid fee (FeeID).
type CandidateItem struct {
	Kind          string
	TransactionID *string
	FeeID         *string
	Date          time.Time
	AmountCents   int64
	ExpenseWaiver bool
	Description   string
	ReceiptID     *string
	ReceiptNumber *string
}

// Ref is the item's id: the transaction or fee id.
func (it CandidateItem) Ref() string {
	if it.TransactionID != nil {
		return *it.TransactionID
	}
	return *it.FeeID
}

// ReceiptRequest asks for a receipt for a member or donor over the given
// items (transaction ids of donations, fee ids of paid fees).
type ReceiptRequest struct {
	MemberID *string
	DonorID  *string
	ItemRefs []string
}

// DonationReceipt is an issued receipt.
type DonationReceipt struct {
	ID               string
	Number           string
	Year             int
	Kind             string
	MemberID         *string
	DonorID          *string
	RecipientName    string
	RecipientAddress string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	TotalCents       int64
	IssueDate        time.Time
	Issuer           donation.Issuer
	DuplicateOf      *string
	CreatedBy        *string
	VoidedAt         *time.Time
	VoidReason       *string
	CreatedAt        time.Time
	Items            []DonationReceiptItem
}

// DonationReceiptItem is a certified donation or fee.
type DonationReceiptItem struct {
	ID            string
	Date          time.Time
	Kind          string
	AmountCents   int64
	ExpenseWaiver bool
	TransactionID *string
	FeeID         *string
}

// Document returns what is printed on the receipt.
func (r DonationReceipt) Document() donation.Receipt {
	doc := donation.Receipt{Number: r.Number, Kind: r.Kind, Duplicate: r.DuplicateOf != nil, Voided: r.VoidedAt != nil,
		RecipientName: r.RecipientName, RecipientAddress: r.RecipientAddress, PeriodStart: r.PeriodStart,
		PeriodEnd: r.PeriodEnd, IssueDate: r.IssueDate, Issuer: r.Issuer}
	for _, it := range r.Items {
		doc.Items = append(doc.Items, donation.Item{Date: it.Date, Kind: it.Kind, AmountCents: it.AmountCents,
			ExpenseWaiver: it.ExpenseWaiver})
	}
	return doc
}

// ── Settings ──────────────────────────────────────────────────────────────────

// GetDonationSettings returns the club's issuer data, or nil before it is set up.
func (s *Store) GetDonationSettings(ctx context.Context) (*donation.Issuer, error) {
	return getDonationSettings(ctx, s.db)
}

func getDonationSettings(ctx context.Context, q querier) (*donation.Issuer, error) {
	var i donation.Issuer
	var date string
	var period, signatory, place sql.NullString
	var fees int
	err := q.QueryRowContext(ctx, `SELECT issuer_name, issuer_address, tax_office, tax_number, exemption_kind,
		exemption_date, assessment_period, purposes, fees_deductible, signatory, place
		FROM donation_settings WHERE id = ? AND deleted_at IS NULL`, donationSettingsID).Scan(&i.Name, &i.Address,
		&i.TaxOffice, &i.TaxNumber, &i.ExemptionKind, &date, &period, &i.Purposes, &fees, &signatory, &place)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	i.ExemptionDate, i.FeesDeductible = parseTime(date), fees != 0
	i.AssessmentPeriod, i.Signatory, i.Place = period.String, signatory.String, place.String
	return &i, nil
}

// SetDonationSettings stores the club's issuer data.
func (s *Store) SetDonationSettings(ctx context.Context, i donation.Issuer) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return w.upsert("donation_settings", row{"id": donationSettingsID}, row{
			"issuer_name": i.Name, "issuer_address": i.Address, "tax_office": i.TaxOffice, "tax_number": i.TaxNumber,
			"exemption_kind": i.ExemptionKind, "exemption_date": db.FormatTime(i.ExemptionDate),
			"assessment_period": emptyNil(i.AssessmentPeriod), "purposes": i.Purposes, "fees_deductible": boolInt(i.FeesDeductible),
			"signatory": emptyNil(i.Signatory), "place": emptyNil(i.Place), "deleted_at": nil,
		})
	})
}

func emptyNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ── Donors ────────────────────────────────────────────────────────────────────

const donorSelect = `SELECT id, name, street, postal_code, city, notes FROM donors WHERE deleted_at IS NULL`

func scanDonor(r scanner) (Donor, error) {
	var d Donor
	var street, postal, city, notes sql.NullString
	if err := r.Scan(&d.ID, &d.Name, &street, &postal, &city, &notes); err != nil {
		return d, mapErr(err)
	}
	d.Street, d.PostalCode, d.City, d.Notes = nullStr(street), nullStr(postal), nullStr(city), nullStr(notes)
	return d, nil
}

// ListDonors returns the live donors by name.
func (s *Store) ListDonors(ctx context.Context) ([]Donor, error) {
	rows, err := s.db.QueryContext(ctx, donorSelect+` ORDER BY name COLLATE NOCASE, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Donor{}
	for rows.Next() {
		d, err := scanDonor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDonor returns a live donor.
func (s *Store) GetDonor(ctx context.Context, id string) (Donor, error) {
	return scanDonor(s.db.QueryRowContext(ctx, donorSelect+` AND id = ?`, id))
}

func donorColumns(in DonorInput) row {
	return row{"name": in.Name, "street": strArg(in.Street), "postal_code": strArg(in.PostalCode),
		"city": strArg(in.City), "notes": strArg(in.Notes)}
}

// CreateDonor inserts a donor.
func (s *Store) CreateDonor(ctx context.Context, in DonorInput) (Donor, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		values := donorColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("donors", values)
	})
	if err != nil {
		return Donor{}, err
	}
	return s.GetDonor(ctx, id)
}

// UpdateDonor replaces the writable fields of a live donor. Issued receipts
// keep the address they were printed with.
func (s *Store) UpdateDonor(ctx context.Context, id string, in DonorInput) (Donor, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("donors", `id = ? AND deleted_at IS NULL`, []any{id}, donorColumns(in)))
	})
	if err != nil {
		return Donor{}, err
	}
	return s.GetDonor(ctx, id)
}

// DeleteDonor tombstones a donor without donations.
func (s *Store) DeleteDonor(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		var n int
		if err := w.QueryRowContext(ctx, `SELECT COUNT(*) FROM finance_transaction_donations d
			JOIN finance_transactions t ON t.id = d.transaction_id AND t.deleted_at IS NULL
			WHERE d.donor_id = ? AND d.deleted_at IS NULL`, id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrDonorInUse
		}
		return affected(w.update("donors", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

// ── Donations ─────────────────────────────────────────────────────────────────

// donationCategory reports whether a cash book category names donations
// (Spende, Spenden, Geldspende, Donation, Don, ...).
func donationCategory(c *string) bool {
	if c == nil {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(*c))
	return strings.Contains(v, "spende") || strings.Contains(v, "donation") || v == "don" || v == "dons"
}

// liveReceiptItems is the subquery of items on receipts that have not been
// voided; duplicates repeat their original and are left out.
const liveReceiptItems = `SELECT i.transaction_id, i.fee_id, r.id AS receipt_id, r.number FROM donation_receipt_items i
	JOIN donation_receipts r ON r.id = i.receipt_id AND r.deleted_at IS NULL AND r.voided_at IS NULL AND r.duplicate_of IS NULL
	WHERE i.deleted_at IS NULL`

// ListDonations returns the incomes that are assigned to a donor or have a
// donation category, newest first.
func (s *Store) ListDonations(ctx context.Context) ([]DonationEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.date, t.description, t.amount_cents, t.category,
		d.expense_waiver, m.id, m.number, m.first_name, m.last_name, o.id, o.name, c.number
		FROM finance_transactions t
		LEFT JOIN finance_transaction_donations d ON d.transaction_id = t.id AND d.deleted_at IS NULL
		LEFT JOIN members m ON m.id = d.member_id
		LEFT JOIN donors o ON o.id = d.donor_id
		LEFT JOIN (`+liveReceiptItems+`) c ON c.transaction_id = t.id
		WHERE t.deleted_at IS NULL AND t.type = 'income'
		ORDER BY t.date DESC, t.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DonationEntry{}
	for rows.Next() {
		var e DonationEntry
		var date string
		var category, memberID, first, last, donorID, donorName, receipt sql.NullString
		var number sql.NullInt64
		var waiver sql.NullInt64
		if err := rows.Scan(&e.TransactionID, &date, &e.Description, &e.AmountCents, &category, &waiver,
			&memberID, &number, &first, &last, &donorID, &donorName, &receipt); err != nil {
			return nil, err
		}
		e.Date, e.Category, e.ExpenseWaiver, e.ReceiptNumber = parseTime(date), nullStr(category), waiver.Int64 != 0, nullStr(receipt)
		if memberID.Valid {
			e.Member = &MemberRef{ID: memberID.String, Number: int(number.Int64), FirstName: first.String, LastName: last.String}
		}
		if donorID.Valid {
			e.Donor = &Donor{ID: donorID.String, Name: donorName.String}
		}
		if e.Assigned() || donationCategory(e.Category) {
			out = append(out, e)
		}
	}
	return out, rows.Err()
}

// certifiedReceipt returns the number of the live receipt certifying a
// transaction or fee, or "" when there is none.
func certifiedReceipt(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, col, id string) (string, error) {
	var number string
	err := q.QueryRowContext(ctx, `SELECT number FROM (`+liveReceiptItems+`) WHERE `+col+` = ? LIMIT 1`, id).Scan(&number)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return number, err
}

// SetDonation marks an income as a donation by a member (memberID) or an
// external donor (donorID). Without a category the transaction gets the
// category "Spende".
func (s *Store) SetDonation(ctx context.Context, transactionID string, memberID, donorID *string, waiver bool) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		var typ string
		var category sql.NullString
		err := w.QueryRowContext(ctx, `SELECT type, category FROM finance_transactions WHERE id = ? AND deleted_at IS NULL`,
			transactionID).Scan(&typ, &category)
		if err != nil {
			return mapErr(err)
		}
		if typ != "income" {
			return ErrNotIncome
		}
		if n, err := certifiedReceipt(ctx, w, "transaction_id", transactionID); err != nil {
			return err
		} else if n != "" {
			return ErrDonationCertified
		}
		if memberID != nil {
			if err := requireLiveMember(ctx, w.Tx, *memberID); err != nil {
				return err
			}
		} else if _, err := s.donorIn(ctx, w, *donorID); err != nil {
			return err
		}
		if err := w.upsert("finance_transaction_donations", row{"transaction_id": transactionID}, row{
			"member_id": strArg(memberID), "donor_id": strArg(donorID), "expense_waiver": boolInt(waiver), "deleted_at": nil,
		}); err != nil {
			return err
		}
		if !category.Valid || strings.TrimSpace(category.String) == "" {
			_, err = w.update("finance_transactions", `id = ?`, []any{transactionID}, row{"category": "Spende"})
		}
		return err
	})
}

func (s *Store) donorIn(ctx context.Context, w *writer, id string) (Donor, error) {
	d, err := scanDonor(w.QueryRowContext(ctx, donorSelect+` AND id = ?`, id))
	if errors.Is(err, ErrNotFound) {
		return d, ErrInvalidReference
	}
	return d, err
}

// RemoveDonation removes the donor of a donation that is not certified.
func (s *Store) RemoveDonation(ctx context.Context, transactionID string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		if n, err := certifiedReceipt(ctx, w, "transaction_id", transactionID); err != nil {
			return err
		} else if n != "" {
			return ErrDonationCertified
		}
		return affected(w.update("finance_transaction_donations", `transaction_id = ? AND deleted_at IS NULL`,
			[]any{transactionID}, row{"deleted_at": db.Now()}))
	})
}

// ── Candidates ────────────────────────────────────────────────────────────────

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// postalAddress joins street and "postal code city" into lines.
func postalAddress(street, postal, city sql.NullString) string {
	var lines []string
	if v := strings.TrimSpace(street.String); v != "" {
		lines = append(lines, v)
	}
	if v := strings.TrimSpace(strings.TrimSpace(postal.String) + " " + strings.TrimSpace(city.String)); v != "" {
		lines = append(lines, v)
	}
	if len(lines) < 2 {
		return "" // a street and a town are both required
	}
	return strings.Join(lines, "\n")
}

// DonationCandidates returns every member and donor with donations in the
// year and, if fees are included, membership fees paid in the year; items
// already certified carry their receipt.
func (s *Store) DonationCandidates(ctx context.Context, year int, includeFees bool) ([]DonationCandidate, error) {
	return donationCandidates(ctx, s.db, year, includeFees)
}

func donationCandidates(ctx context.Context, q querier, year int, includeFees bool) ([]DonationCandidate, error) {
	byKey := map[string]*DonationCandidate{}
	var order []string
	add := func(c DonationCandidate, it CandidateItem) {
		k := c.Key()
		if _, ok := byKey[k]; !ok {
			byKey[k] = &c
			order = append(order, k)
		}
		byKey[k].Items = append(byKey[k].Items, it)
	}
	from, to := fmt.Sprintf("%04d", year), fmt.Sprintf("%04d", year+1)

	rows, err := q.QueryContext(ctx, `SELECT t.id, t.date, t.description, t.amount_cents, d.expense_waiver,
		m.id, m.number, m.first_name, m.last_name, mc.street, mc.postal_code, mc.city,
		o.id, o.name, o.street, o.postal_code, o.city, c.receipt_id, c.number
		FROM finance_transaction_donations d
		JOIN finance_transactions t ON t.id = d.transaction_id AND t.deleted_at IS NULL AND t.type = 'income'
		LEFT JOIN members m ON m.id = d.member_id
		LEFT JOIN member_contacts mc ON mc.member_id = m.id AND mc.deleted_at IS NULL
		LEFT JOIN donors o ON o.id = d.donor_id
		LEFT JOIN (`+liveReceiptItems+`) c ON c.transaction_id = t.id
		WHERE d.deleted_at IS NULL AND t.date >= ? AND t.date < ?
		ORDER BY t.date, t.id`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it CandidateItem
		var c DonationCandidate
		var txID, date string
		var waiver int
		var memberID, first, last, mStreet, mPostal, mCity, donorID, donorName, dStreet, dPostal, dCity, receiptID, receiptNo sql.NullString
		var number sql.NullInt64
		if err := rows.Scan(&txID, &date, &it.Description, &it.AmountCents, &waiver, &memberID, &number, &first, &last,
			&mStreet, &mPostal, &mCity, &donorID, &donorName, &dStreet, &dPostal, &dCity, &receiptID, &receiptNo); err != nil {
			rows.Close()
			return nil, err
		}
		it.Kind, it.TransactionID, it.Date, it.ExpenseWaiver = donation.ItemDonation, &txID, parseTime(date), waiver != 0
		it.ReceiptID, it.ReceiptNumber = nullStr(receiptID), nullStr(receiptNo)
		switch {
		case memberID.Valid:
			c.Member = &MemberRef{ID: memberID.String, Number: int(number.Int64), FirstName: first.String, LastName: last.String}
			c.Name, c.Address = first.String+" "+last.String, postalAddress(mStreet, mPostal, mCity)
		case donorID.Valid:
			c.Donor = &Donor{ID: donorID.String, Name: donorName.String}
			c.Name, c.Address = donorName.String, postalAddress(dStreet, dPostal, dCity)
		default:
			continue
		}
		add(c, it)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if includeFees {
		rows, err := q.QueryContext(ctx, `SELECT f.id, f.paid_date, f.amount_cents, f.description, f.year,
			m.id, m.number, m.first_name, m.last_name, mc.street, mc.postal_code, mc.city, c.receipt_id, c.number
			FROM member_fees f
			JOIN members m ON m.id = f.member_id
			LEFT JOIN member_contacts mc ON mc.member_id = m.id AND mc.deleted_at IS NULL
			LEFT JOIN (`+liveReceiptItems+`) c ON c.fee_id = f.id
			WHERE f.deleted_at IS NULL AND f.paid_date >= ? AND f.paid_date < ? AND f.amount_cents > 0
			ORDER BY f.paid_date, f.id`, from, to)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var it CandidateItem
			var c DonationCandidate
			var feeID, paid string
			var feeYear int
			var desc, mStreet, mPostal, mCity, receiptID, receiptNo sql.NullString
			var m MemberRef
			if err := rows.Scan(&feeID, &paid, &it.AmountCents, &desc, &feeYear, &m.ID, &m.Number, &m.FirstName, &m.LastName,
				&mStreet, &mPostal, &mCity, &receiptID, &receiptNo); err != nil {
				rows.Close()
				return nil, err
			}
			it.Kind, it.FeeID, it.Date = donation.ItemMembershipFee, &feeID, parseTime(paid)
			it.Description = fmt.Sprintf("Mitgliedsbeitrag %d", feeYear)
			if desc.Valid && strings.TrimSpace(desc.String) != "" {
				it.Description = strings.TrimSpace(desc.String)
			}
			it.ReceiptID, it.ReceiptNumber = nullStr(receiptID), nullStr(receiptNo)
			c.Member = &m
			c.Name, c.Address = m.FirstName+" "+m.LastName, postalAddress(mStreet, mPostal, mCity)
			add(c, it)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	out := make([]DonationCandidate, 0, len(order))
	for _, k := range order {
		c := *byKey[k]
		sort.SliceStable(c.Items, func(i, j int) bool { return c.Items[i].Date.Before(c.Items[j].Date) })
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// ── Receipts ──────────────────────────────────────────────────────────────────

// CreateReceipts issues one receipt per request in a single transaction:
// a single receipt for one item, a collective receipt for several. A request
// whose items are exactly those of an issued receipt yields a duplicate of
// it. Numbers run per year (2026-0001, ...).
func (s *Store) CreateReceipts(ctx context.Context, reqs []ReceiptRequest, issueDate time.Time, createdBy string) ([]DonationReceipt, error) {
	var ids []string
	err := s.inWriteTx(ctx, func(w *writer) error {
		issuer, err := getDonationSettings(ctx, w)
		if err != nil {
			return err
		}
		if issuer == nil {
			return ErrDonationsNotConfigured
		}
		for _, req := range reqs {
			id, err := s.createReceipt(ctx, w, *issuer, req, issueDate, createdBy)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]DonationReceipt, 0, len(ids))
	for _, id := range ids {
		r, err := s.GetReceipt(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) createReceipt(ctx context.Context, w *writer, issuer donation.Issuer, req ReceiptRequest,
	issueDate time.Time, createdBy string) (string, error) {
	if len(req.ItemRefs) == 0 || (req.MemberID == nil) == (req.DonorID == nil) {
		return "", &ReceiptItemsError{Invalid: req.ItemRefs}
	}
	// All items must be of one calendar year; the first item that resolves
	// fixes it.
	var cand *DonationCandidate
	if year, ok := itemYear(ctx, w, req.ItemRefs); ok {
		list, err := donationCandidates(ctx, w, year, issuer.FeesDeductible)
		if err != nil {
			return "", err
		}
		for i, c := range list {
			if (req.MemberID != nil && c.Member != nil && c.Member.ID == *req.MemberID) ||
				(req.DonorID != nil && c.Donor != nil && c.Donor.ID == *req.DonorID) {
				cand = &list[i]
			}
		}
	}
	byRef := map[string]CandidateItem{}
	if cand != nil {
		for _, it := range cand.Items {
			byRef[it.Ref()] = it
		}
	}
	var items []CandidateItem
	var bad ReceiptItemsError
	seen := map[string]bool{}
	for _, ref := range req.ItemRefs {
		it, ok := byRef[ref]
		switch {
		case !ok || seen[ref]:
			bad.Invalid = append(bad.Invalid, ref)
		case it.ReceiptID != nil:
			bad.Certified = append(bad.Certified, ref)
		}
		seen[ref] = true
		items = append(items, it)
	}
	if len(bad.Invalid) > 0 {
		return "", &bad
	}
	if len(bad.Certified) > 0 {
		// Exactly the items of one issued receipt: repeat it as a duplicate.
		if sameReceipt(items) {
			first := items[0].ReceiptID
			orig, err := getReceipt(ctx, w, *first)
			if err != nil {
				return "", err
			}
			if len(orig.Items) == len(items) {
				return s.duplicate(ctx, w, orig, createdBy)
			}
		}
		return "", &bad
	}
	if cand.Address == "" {
		return "", ErrMissingAddress
	}

	year := items[0].Date.Year()
	var seq int
	if err := w.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM donation_receipts WHERE year = ?`, year).Scan(&seq); err != nil {
		return "", err
	}
	kind := donation.KindSingle
	if len(items) > 1 {
		kind = donation.KindCollective
	}
	var total int64
	start, end := items[0].Date, items[0].Date
	for _, it := range items {
		total += it.AmountCents
		if it.Date.Before(start) {
			start = it.Date
		}
		if it.Date.After(end) {
			end = it.Date
		}
	}
	if kind == donation.KindCollective {
		// A collective receipt covers the calendar year up to its last item.
		start = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	issuerJSON, err := json.Marshal(issuer)
	if err != nil {
		return "", err
	}
	id, now := db.NewID(), db.Now()
	if err := w.insert("donation_receipts", row{
		"id": id, "number": fmt.Sprintf("%d-%04d", year, seq), "year": year, "sequence": seq, "kind": kind,
		"member_id": strArg(req.MemberID), "donor_id": strArg(req.DonorID), "recipient_name": cand.Name,
		"recipient_address": cand.Address, "period_start": db.FormatTime(start), "period_end": db.FormatTime(end),
		"total_cents": total, "issue_date": db.FormatTime(issueDate), "issuer": string(issuerJSON),
		"created_by": emptyNil(createdBy), "created_at": now,
	}); err != nil {
		return "", err
	}
	for _, it := range items {
		if err := w.insert("donation_receipt_items", row{
			"id": db.NewID(), "receipt_id": id, "item_date": db.FormatTime(it.Date), "kind": it.Kind,
			"amount_cents": it.AmountCents, "expense_waiver": boolInt(it.ExpenseWaiver),
			"transaction_id": strArg(it.TransactionID), "fee_id": strArg(it.FeeID), "created_at": now,
		}); err != nil {
			return "", err
		}
	}
	return id, nil
}

// sameReceipt reports whether all items are certified by the same receipt.
func sameReceipt(items []CandidateItem) bool {
	for _, it := range items {
		if it.ReceiptID == nil || *it.ReceiptID != *items[0].ReceiptID {
			return false
		}
	}
	return true
}

// itemYear returns the year of the first item ref that names a transaction
// or a paid fee.
func itemYear(ctx context.Context, q querier, refs []string) (int, bool) {
	for _, ref := range refs {
		var date string
		err := q.QueryRowContext(ctx, `SELECT date FROM finance_transactions WHERE id = ? AND deleted_at IS NULL
			UNION ALL SELECT paid_date FROM member_fees WHERE id = ? AND deleted_at IS NULL AND paid_date IS NOT NULL
			LIMIT 1`, ref, ref).Scan(&date)
		if err == nil {
			return parseTime(date).Year(), true
		}
	}
	return 0, false
}

// DuplicateReceipt issues a duplicate of a receipt: same number, data and
// issue date, marked as duplicate.
func (s *Store) DuplicateReceipt(ctx context.Context, id, createdBy string) (DonationReceipt, error) {
	var newID string
	err := s.inWriteTx(ctx, func(w *writer) error {
		orig, err := getReceipt(ctx, w, id)
		if err != nil {
			return err
		}
		newID, err = s.duplicate(ctx, w, orig, createdBy)
		return err
	})
	if err != nil {
		return DonationReceipt{}, err
	}
	return s.GetReceipt(ctx, newID)
}

func (s *Store) duplicate(ctx context.Context, w *writer, orig DonationReceipt, createdBy string) (string, error) {
	if orig.DuplicateOf != nil {
		var err error
		if orig, err = getReceipt(ctx, w, *orig.DuplicateOf); err != nil {
			return "", err
		}
	}
	if orig.VoidedAt != nil {
		return "", ErrReceiptVoided
	}
	issuerJSON, err := json.Marshal(orig.Issuer)
	if err != nil {
		return "", err
	}
	id, now := db.NewID(), db.Now()
	if err := w.insert("donation_receipts", row{
		"id": id, "number": orig.Number, "year": orig.Year, "sequence": 0, "kind": orig.Kind,
		"member_id": strArg(orig.MemberID), "donor_id": strArg(orig.DonorID), "recipient_name": orig.RecipientName,
		"recipient_address": orig.RecipientAddress, "period_start": db.FormatTime(orig.PeriodStart),
		"period_end": db.FormatTime(orig.PeriodEnd), "total_cents": orig.TotalCents, "issue_date": db.FormatTime(orig.IssueDate),
		"issuer": string(issuerJSON), "duplicate_of": orig.ID, "created_by": emptyNil(createdBy), "created_at": now,
	}); err != nil {
		return "", err
	}
	for _, it := range orig.Items {
		if err := w.insert("donation_receipt_items", row{
			"id": db.NewID(), "receipt_id": id, "item_date": db.FormatTime(it.Date), "kind": it.Kind,
			"amount_cents": it.AmountCents, "expense_waiver": boolInt(it.ExpenseWaiver),
			"transaction_id": strArg(it.TransactionID), "fee_id": strArg(it.FeeID), "created_at": now,
		}); err != nil {
			return "", err
		}
	}
	return id, nil
}

// VoidReceipt marks a receipt as voided (e.g. after a wrong amount); its
// items can be certified again. Duplicates of it are voided with it.
func (s *Store) VoidReceipt(ctx context.Context, id, reason string) (DonationReceipt, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		r, err := getReceipt(ctx, w, id)
		if err != nil {
			return err
		}
		if r.VoidedAt != nil {
			return ErrReceiptVoided
		}
		set := row{"voided_at": db.Now(), "void_reason": emptyNil(reason)}
		_, err = w.update("donation_receipts", `(id = ? OR duplicate_of = ?) AND deleted_at IS NULL AND voided_at IS NULL`,
			[]any{id, id}, set)
		return err
	})
	if err != nil {
		return DonationReceipt{}, err
	}
	return s.GetReceipt(ctx, id)
}

const receiptSelect = `SELECT id, number, year, kind, member_id, donor_id, recipient_name, recipient_address, period_start,
	period_end, total_cents, issue_date, issuer, duplicate_of, created_by, voided_at, void_reason, created_at
	FROM donation_receipts WHERE deleted_at IS NULL`

func scanReceipt(r scanner) (DonationReceipt, error) {
	var d DonationReceipt
	var start, end, issued, issuer, created string
	var memberID, donorID, dup, by, voided, reason sql.NullString
	if err := r.Scan(&d.ID, &d.Number, &d.Year, &d.Kind, &memberID, &donorID, &d.RecipientName, &d.RecipientAddress,
		&start, &end, &d.TotalCents, &issued, &issuer, &dup, &by, &voided, &reason, &created); err != nil {
		return d, mapErr(err)
	}
	d.PeriodStart, d.PeriodEnd, d.IssueDate, d.CreatedAt = parseTime(start), parseTime(end), parseTime(issued), parseTime(created)
	d.MemberID, d.DonorID, d.DuplicateOf, d.CreatedBy = nullStr(memberID), nullStr(donorID), nullStr(dup), nullStr(by)
	d.VoidedAt, d.VoidReason = nullTime(voided), nullStr(reason)
	if err := json.Unmarshal([]byte(issuer), &d.Issuer); err != nil {
		return d, fmt.Errorf("receipt %s: issuer: %w", d.ID, err)
	}
	return d, nil
}

func receiptItems(ctx context.Context, q querier, ids []string) (map[string][]DonationReceiptItem, error) {
	out := map[string][]DonationReceiptItem{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx, `SELECT id, receipt_id, item_date, kind, amount_cents, expense_waiver, transaction_id, fee_id
		FROM donation_receipt_items WHERE deleted_at IS NULL AND receipt_id IN (`+placeholders(len(ids))+`)
		ORDER BY item_date, id`, stringArgs(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it DonationReceiptItem
		var receiptID, date string
		var waiver int
		var tx, fee sql.NullString
		if err := rows.Scan(&it.ID, &receiptID, &date, &it.Kind, &it.AmountCents, &waiver, &tx, &fee); err != nil {
			return nil, err
		}
		it.Date, it.ExpenseWaiver, it.TransactionID, it.FeeID = parseTime(date), waiver != 0, nullStr(tx), nullStr(fee)
		out[receiptID] = append(out[receiptID], it)
	}
	return out, rows.Err()
}

func getReceipt(ctx context.Context, q querier, id string) (DonationReceipt, error) {
	r, err := scanReceipt(q.QueryRowContext(ctx, receiptSelect+` AND id = ?`, id))
	if err != nil {
		return r, err
	}
	items, err := receiptItems(ctx, q, []string{id})
	r.Items = items[id]
	return r, err
}

// GetReceipt returns a receipt with its items.
func (s *Store) GetReceipt(ctx context.Context, id string) (DonationReceipt, error) {
	return getReceipt(ctx, s.db, id)
}

// ListReceipts returns the receipts of a year (all years for 0), the log of
// what was certified: newest number first, duplicates after their original.
func (s *Store) ListReceipts(ctx context.Context, year int) ([]DonationReceipt, error) {
	query, args := receiptSelect, []any{}
	if year != 0 {
		query += ` AND year = ?`
		args = append(args, year)
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY year DESC, number DESC, created_at`, args...)
	if err != nil {
		return nil, err
	}
	out := []DonationReceipt{}
	var ids []string
	for rows.Next() {
		r, err := scanReceipt(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
		ids = append(ids, r.ID)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	items, err := receiptItems(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Items = items[out[i].ID]
	}
	return out, nil
}
