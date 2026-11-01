package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sepa"
)

// creditorID is the primary key of the single sepa_creditor row.
const creditorID = "club"

// DefaultRemittanceTemplate is used when neither the run nor the creditor
// settings define a remittance text.
const DefaultRemittanceTemplate = "{description} Mitglied {number}"

// Problems that keep a fee out of a direct debit run.
const (
	IssueMissingIBAN             = "MISSING_IBAN"
	IssueInvalidIBAN             = "INVALID_IBAN"
	IssueInvalidBIC              = "INVALID_BIC"
	IssueMissingMandate          = "MISSING_MANDATE"
	IssueMissingMandateReference = "MISSING_MANDATE_REFERENCE"
	IssueInvalidMandateReference = "INVALID_MANDATE_REFERENCE"
	IssueInvalidAmount           = "INVALID_AMOUNT"
)

var (
	// ErrSEPANotConfigured is returned when a run is created before the
	// creditor data has been set up.
	ErrSEPANotConfigured = errors.New("sepa creditor not configured")
	// ErrSEPARunPaid is returned when a confirmed run is cancelled.
	ErrSEPARunPaid = errors.New("sepa run already confirmed")
	// ErrSEPANoFees is returned when a run would contain no fee.
	ErrSEPANoFees = errors.New("no fees to collect")
)

// SEPAInvalidError lists the fees of a requested run that cannot be
// collected (not open any more, or with the issues of SEPACandidate).
type SEPAInvalidError struct {
	FeeIDs []string
}

func (e *SEPAInvalidError) Error() string {
	return fmt.Sprintf("%d fees cannot be collected", len(e.FeeIDs))
}

// SEPACreditor is the club as creditor of its direct debits.
type SEPACreditor struct {
	Name               string
	CreditorID         string
	IBAN               string
	BIC                *string
	RemittanceTemplate *string
}

// SEPACandidate is an open fee (unpaid and not in a run) with the bank and
// mandate data of its member.
type SEPACandidate struct {
	FeeID         string
	Member        MemberRef
	AmountCents   int64
	Year          int
	Description   *string
	DueDate       time.Time
	AccountHolder *string
	IBAN          *string
	BIC           *string
	MandateDate   *time.Time
	MandateRef    *string
	// SequenceType is FRST until the mandate has been used in a run.
	SequenceType string
	Issues       []string
}

// DebtorName is the account holder or, without one, the member's name.
func (c SEPACandidate) DebtorName() string {
	if c.AccountHolder != nil && strings.TrimSpace(*c.AccountHolder) != "" {
		return strings.TrimSpace(*c.AccountHolder)
	}
	return c.Member.FirstName + " " + c.Member.LastName
}

// Remittance renders a remittance template for the fee. Placeholders:
// {firstName}, {lastName}, {number}, {year}, {description} (the fee
// description, or "Beitrag <year>" without one) and {amount}.
func (c SEPACandidate) Remittance(template string) string {
	if strings.TrimSpace(template) == "" {
		template = DefaultRemittanceTemplate
	}
	desc := "Beitrag " + strconv.Itoa(c.Year)
	if c.Description != nil && strings.TrimSpace(*c.Description) != "" {
		desc = strings.TrimSpace(*c.Description)
	}
	r := strings.NewReplacer("{firstName}", c.Member.FirstName, "{lastName}", c.Member.LastName,
		"{number}", strconv.Itoa(c.Member.Number), "{year}", strconv.Itoa(c.Year), "{description}", desc,
		"{amount}", strings.Replace(sepa.Amount(c.AmountCents), ".", ",", 1)+" EUR")
	return sepa.Text(r.Replace(template), 140)
}

func (c *SEPACandidate) check() {
	switch {
	case c.IBAN == nil || *c.IBAN == "":
		c.Issues = append(c.Issues, IssueMissingIBAN)
	case !sepa.ValidIBAN(sepa.NormalizeIBAN(*c.IBAN)):
		c.Issues = append(c.Issues, IssueInvalidIBAN)
	}
	if c.BIC != nil && *c.BIC != "" && !sepa.ValidBIC(sepa.NormalizeIBAN(*c.BIC)) {
		c.Issues = append(c.Issues, IssueInvalidBIC)
	}
	if c.MandateDate == nil {
		c.Issues = append(c.Issues, IssueMissingMandate)
	}
	switch {
	case c.MandateRef == nil:
		c.Issues = append(c.Issues, IssueMissingMandateReference)
	case !sepa.ValidMandateReference(*c.MandateRef):
		c.Issues = append(c.Issues, IssueInvalidMandateReference)
	}
	if c.AmountCents <= 0 {
		c.Issues = append(c.Issues, IssueInvalidAmount)
	}
}

// SEPARun is an exported direct debit run.
type SEPARun struct {
	ID             string
	CollectionDate time.Time
	Creditor       SEPACreditor
	PaidDate       *time.Time
	CreatedAt      time.Time
	Items          []SEPARunItem
}

// TotalCents is the sum of all debits of the run.
func (r SEPARun) TotalCents() int64 {
	var sum int64
	for _, it := range r.Items {
		sum += it.AmountCents
	}
	return sum
}

// SEPARunItem is one debit of a run.
type SEPARunItem struct {
	ID           string
	FeeID        string
	Member       MemberRef
	AmountCents  int64
	SequenceType string
	MandateRef   string
	MandateDate  time.Time
	DebtorName   string
	IBAN         string
	BIC          *string
	Remittance   string
}

// SEPARunInput describes a run to create.
type SEPARunInput struct {
	CollectionDate time.Time
	// RemittanceTemplate overrides the template of the creditor settings.
	RemittanceTemplate *string
	FeeIDs             []string
}

// GetSEPACreditor returns the creditor settings, or nil before they are set up.
func (s *Store) GetSEPACreditor(ctx context.Context) (*SEPACreditor, error) {
	var c SEPACreditor
	var bic, tpl sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT name, creditor_id, iban, bic, remittance_template FROM sepa_creditor
		WHERE id = ? AND deleted_at IS NULL`, creditorID).Scan(&c.Name, &c.CreditorID, &c.IBAN, &bic, &tpl)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.BIC, c.RemittanceTemplate = nullStr(bic), nullStr(tpl)
	return &c, nil
}

// SetSEPACreditor stores the creditor settings.
func (s *Store) SetSEPACreditor(ctx context.Context, c SEPACreditor) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return w.upsert("sepa_creditor", row{"id": creditorID}, row{
			"name": c.Name, "creditor_id": c.CreditorID, "iban": c.IBAN, "bic": strArg(c.BIC),
			"remittance_template": strArg(c.RemittanceTemplate), "deleted_at": nil,
		})
	})
}

// GetMandateReference returns the mandate reference of a live member, or nil.
func (s *Store) GetMandateReference(ctx context.Context, memberID string) (*string, error) {
	var ref sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT s.mandate_reference FROM members m
		LEFT JOIN member_sepa_mandates s ON s.member_id = m.id AND s.deleted_at IS NULL
		WHERE m.id = ? AND m.deleted_at IS NULL`, memberID).Scan(&ref)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return nullStr(ref), nil
}

// SetMandateReference sets (or with nil removes) the mandate reference of a
// live member. References are unique among members.
func (s *Store) SetMandateReference(ctx context.Context, memberID string, ref *string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			if errors.Is(err, ErrInvalidReference) {
				return ErrNotFound
			}
			return err
		}
		if ref == nil {
			_, err := w.update("member_sepa_mandates", `member_id = ? AND deleted_at IS NULL`, []any{memberID},
				row{"deleted_at": db.Now()})
			return err
		}
		if err := w.requireUnique("member_sepa_mandates", "mandate_reference", *ref, "member_id", memberID); err != nil {
			return err
		}
		return w.upsert("member_sepa_mandates", row{"member_id": memberID},
			row{"mandate_reference": *ref, "deleted_at": nil})
	})
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SEPACandidates returns all open fees of live members with their bank and
// mandate data and the issues that keep them out of a run, ordered by member.
func (s *Store) SEPACandidates(ctx context.Context) ([]SEPACandidate, error) {
	return sepaCandidates(ctx, s.db, nil)
}

func sepaCandidates(ctx context.Context, q queryer, feeIDs []string) ([]SEPACandidate, error) {
	query := `SELECT f.id, f.amount_cents, f.year, f.description, f.due_date,
		m.id, m.number, m.first_name, m.last_name,
		b.account_holder, b.iban, b.bic, b.sepa_mandate_date, sm.mandate_reference,
		EXISTS (SELECT 1 FROM sepa_run_items i JOIN sepa_runs r ON r.id = i.run_id AND r.deleted_at IS NULL
			WHERE i.deleted_at IS NULL AND i.member_id = m.id AND i.mandate_reference = sm.mandate_reference)
	FROM member_fees f
	JOIN members m ON m.id = f.member_id AND m.deleted_at IS NULL
	LEFT JOIN member_bank_details b ON b.member_id = m.id AND b.deleted_at IS NULL
	LEFT JOIN member_sepa_mandates sm ON sm.member_id = m.id AND sm.deleted_at IS NULL
	WHERE f.deleted_at IS NULL AND f.paid_date IS NULL
	AND NOT EXISTS (SELECT 1 FROM sepa_run_items i JOIN sepa_runs r ON r.id = i.run_id AND r.deleted_at IS NULL
		WHERE i.deleted_at IS NULL AND i.fee_id = f.id)`
	var args []any
	if feeIDs != nil {
		query += ` AND f.id IN (` + placeholders(len(feeIDs)) + `)`
		args = stringArgs(feeIDs)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY m.last_name, m.first_name, m.number, f.due_date, f.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SEPACandidate{}
	for rows.Next() {
		var c SEPACandidate
		var desc, holder, iban, bic, mandate, ref sql.NullString
		var due string
		var used bool
		if err := rows.Scan(&c.FeeID, &c.AmountCents, &c.Year, &desc, &due,
			&c.Member.ID, &c.Member.Number, &c.Member.FirstName, &c.Member.LastName,
			&holder, &iban, &bic, &mandate, &ref, &used); err != nil {
			return nil, err
		}
		c.Description, c.DueDate = nullStr(desc), parseTime(due)
		c.AccountHolder, c.IBAN, c.BIC, c.MandateDate, c.MandateRef = nullStr(holder), nullStr(iban), nullStr(bic), nullTime(mandate), nullStr(ref)
		c.SequenceType = sepa.SeqFirst
		if used {
			c.SequenceType = sepa.SeqRecurring
		}
		c.Issues = []string{}
		c.check()
		result = append(result, c)
	}
	return result, rows.Err()
}

// CreateSEPARun creates a run for the given open fees. Every fee must be
// open and collectable, otherwise a *SEPAInvalidError names the others.
func (s *Store) CreateSEPARun(ctx context.Context, in SEPARunInput) (SEPARun, error) {
	ids := slices.Compact(slices.Sorted(slices.Values(in.FeeIDs)))
	if len(ids) == 0 {
		return SEPARun{}, ErrSEPANoFees
	}
	creditor, err := s.GetSEPACreditor(ctx)
	if err != nil {
		return SEPARun{}, err
	}
	if creditor == nil {
		return SEPARun{}, ErrSEPANotConfigured
	}
	template := ""
	if in.RemittanceTemplate != nil {
		template = *in.RemittanceTemplate
	} else if creditor.RemittanceTemplate != nil {
		template = *creditor.RemittanceTemplate
	}
	runID := db.NewID()
	err = s.inWriteTx(ctx, func(w *writer) error {
		candidates, err := sepaCandidates(ctx, w.Tx, ids)
		if err != nil {
			return err
		}
		found := map[string]bool{}
		invalid := []string{}
		for _, c := range candidates {
			found[c.FeeID] = true
			if len(c.Issues) > 0 {
				invalid = append(invalid, c.FeeID)
			}
		}
		for _, id := range ids {
			if !found[id] {
				invalid = append(invalid, id)
			}
		}
		if len(invalid) > 0 {
			sort.Strings(invalid)
			return &SEPAInvalidError{FeeIDs: invalid}
		}
		now := db.Now()
		if err := w.insert("sepa_runs", row{
			"id": runID, "collection_date": db.FormatTime(dateOnly(in.CollectionDate)), "creditor_name": creditor.Name,
			"creditor_id": creditor.CreditorID, "creditor_iban": creditor.IBAN, "creditor_bic": strArg(creditor.BIC),
			"created_at": now,
		}); err != nil {
			return err
		}
		for _, c := range candidates {
			var bic any
			if c.BIC != nil && *c.BIC != "" {
				bic = sepa.NormalizeIBAN(*c.BIC)
			}
			if err := w.insert("sepa_run_items", row{
				"id": db.NewID(), "run_id": runID, "fee_id": c.FeeID, "member_id": c.Member.ID,
				"amount_cents": c.AmountCents, "sequence_type": c.SequenceType, "mandate_reference": *c.MandateRef,
				"mandate_date": db.FormatTime(*c.MandateDate), "debtor_name": c.DebtorName(),
				"iban": sepa.NormalizeIBAN(*c.IBAN), "bic": bic, "remittance": c.Remittance(template), "created_at": now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return SEPARun{}, err
	}
	return s.GetSEPARun(ctx, runID)
}

// ListSEPARuns returns all live runs with their debits, newest first.
func (s *Store) ListSEPARuns(ctx context.Context) ([]SEPARun, error) {
	return s.querySEPARuns(ctx, "")
}

// GetSEPARun returns a live run.
func (s *Store) GetSEPARun(ctx context.Context, id string) (SEPARun, error) {
	runs, err := s.querySEPARuns(ctx, id)
	if err != nil {
		return SEPARun{}, err
	}
	if len(runs) == 0 {
		return SEPARun{}, ErrNotFound
	}
	return runs[0], nil
}

func (s *Store) querySEPARuns(ctx context.Context, id string) ([]SEPARun, error) {
	query := `SELECT id, collection_date, creditor_name, creditor_id, creditor_iban, creditor_bic, paid_date, created_at
		FROM sepa_runs WHERE deleted_at IS NULL`
	var args []any
	if id != "" {
		query += ` AND id = ?`
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY created_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	result := []SEPARun{}
	index := map[string]int{}
	for rows.Next() {
		var r SEPARun
		var collection, created string
		var bic, paid sql.NullString
		if err := rows.Scan(&r.ID, &collection, &r.Creditor.Name, &r.Creditor.CreditorID, &r.Creditor.IBAN, &bic,
			&paid, &created); err != nil {
			rows.Close()
			return nil, err
		}
		r.CollectionDate, r.CreatedAt, r.Creditor.BIC, r.PaidDate = parseTime(collection), parseTime(created), nullStr(bic), nullTime(paid)
		r.Items = []SEPARunItem{}
		index[r.ID] = len(result)
		result = append(result, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(result) == 0 {
		return result, err
	}

	query = `SELECT i.run_id, i.id, i.fee_id, m.id, m.number, m.first_name, m.last_name, i.amount_cents,
		i.sequence_type, i.mandate_reference, i.mandate_date, i.debtor_name, i.iban, i.bic, i.remittance
		FROM sepa_run_items i JOIN members m ON m.id = i.member_id
		JOIN sepa_runs r ON r.id = i.run_id AND r.deleted_at IS NULL
		WHERE i.deleted_at IS NULL`
	args = nil
	if id != "" {
		query += ` AND i.run_id = ?`
		args = append(args, id)
	}
	rows, err = s.db.QueryContext(ctx, query+` ORDER BY m.last_name, m.first_name, m.number, i.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var runID, mandate string
		var it SEPARunItem
		var bic sql.NullString
		if err := rows.Scan(&runID, &it.ID, &it.FeeID, &it.Member.ID, &it.Member.Number, &it.Member.FirstName,
			&it.Member.LastName, &it.AmountCents, &it.SequenceType, &it.MandateRef, &mandate, &it.DebtorName,
			&it.IBAN, &bic, &it.Remittance); err != nil {
			return nil, err
		}
		it.MandateDate, it.BIC = parseTime(mandate), nullStr(bic)
		if i, ok := index[runID]; ok {
			result[i].Items = append(result[i].Items, it)
		}
	}
	return result, rows.Err()
}

// CancelSEPARun tombstones a run that has not been confirmed; its fees are
// open again.
func (s *Store) CancelSEPARun(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		var paid sql.NullString
		err := w.QueryRowContext(ctx, `SELECT paid_date FROM sepa_runs WHERE id = ? AND deleted_at IS NULL`, id).Scan(&paid)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if paid.Valid {
			return ErrSEPARunPaid
		}
		tombstone := row{"deleted_at": db.Now()}
		if _, err := w.update("sepa_run_items", `run_id = ? AND deleted_at IS NULL`, []any{id}, tombstone); err != nil {
			return err
		}
		return affected(w.update("sepa_runs", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone))
	})
}

// ConfirmSEPARun records that the bank has collected the run: the run and
// its fees that are still unpaid get paid as paid date.
func (s *Store) ConfirmSEPARun(ctx context.Context, id string, paid time.Time) (SEPARun, error) {
	date := db.FormatTime(dateOnly(paid))
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update("sepa_runs", `id = ? AND deleted_at IS NULL`, []any{id}, row{"paid_date": date})); err != nil {
			return err
		}
		_, err := w.update("member_fees", `deleted_at IS NULL AND paid_date IS NULL AND id IN
			(SELECT fee_id FROM sepa_run_items WHERE run_id = ? AND deleted_at IS NULL)`, []any{id}, row{"paid_date": date})
		return err
	})
	if err != nil {
		return SEPARun{}, err
	}
	return s.GetSEPARun(ctx, id)
}

// Initiation converts a run into a pain.008 document.
func (r SEPARun) Initiation() sepa.Initiation {
	in := sepa.Initiation{
		MessageID: strings.ReplaceAll(r.ID, "-", ""), Created: r.CreatedAt, CollectionDate: r.CollectionDate,
		Creditor: sepa.Creditor{Name: r.Creditor.Name, CreditorID: r.Creditor.CreditorID, IBAN: r.Creditor.IBAN},
	}
	if r.Creditor.BIC != nil {
		in.Creditor.BIC = *r.Creditor.BIC
	}
	for _, it := range r.Items {
		d := sepa.Debit{
			EndToEndID: strings.ReplaceAll(it.FeeID, "-", ""), AmountCents: it.AmountCents, SequenceType: it.SequenceType,
			MandateID: it.MandateRef, MandateDate: it.MandateDate, DebtorName: it.DebtorName, IBAN: it.IBAN,
			Remittance: it.Remittance,
		}
		if it.BIC != nil {
			d.BIC = *it.BIC
		}
		in.Debits = append(in.Debits, d)
	}
	return in
}
