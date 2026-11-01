package store

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/fees"
)

// How a member's fee model is chosen.
const (
	FeeModeAuto   = "auto"   // by age, among the models with an age limit
	FeeModeModel  = "model"  // a fixed model
	FeeModeExempt = "exempt" // no fee
)

// Reasons why a member is left out of a billing run.
const (
	FeeSkipNoModel        = "NO_MODEL"
	FeeSkipExempt         = "EXEMPT"
	FeeSkipAlreadyCharged = "ALREADY_CHARGED"
)

var (
	// ErrFeeRunEmpty is returned when a billing run would create no fee, for
	// example because the period has already been charged.
	ErrFeeRunEmpty = errors.New("billing run creates no fees")
	// ErrFeeRunInCollection is returned when a run is deleted while some of
	// its fees are in a direct debit run that has not been confirmed.
	ErrFeeRunInCollection = errors.New("fees of the run are being collected")
)

// FeeAssignment is how a member's fee model is chosen.
type FeeAssignment struct {
	Mode    string
	ModelID *string
}

// MemberFeeModel is the fee model that applies to a member on a day.
type MemberFeeModel struct {
	Member MemberRef
	Mode   string
	// Model is nil when the member is exempt or no model fits.
	Model *fees.Model
	// Age is the age on 1 January of the day's year, which decides the model.
	Age *int
}

// FeeRunInput selects the periods of a billing run: those starting in the
// months From to To. Fees are due on DueDate, or at the start of a later
// period.
type FeeRunInput struct {
	From    time.Time
	To      time.Time
	DueDate time.Time
}

// FeeRunItem is a fee a billing run creates.
type FeeRunItem struct {
	Member      MemberRef
	Model       fees.Model
	Period      fees.Period
	AmountCents int64
	DueDate     time.Time
}

// ProRata reports whether the amount is less than a full period.
func (it FeeRunItem) ProRata() bool { return it.AmountCents < it.Model.AmountCents }

// Description is the text of the created fee, e.g. "Erwachsene 2027-H1".
func (it FeeRunItem) Description() string {
	return it.Model.Name + " " + it.Period.Label(it.Model.Interval)
}

// FeeRunSkip is a member a billing run leaves out (for ALREADY_CHARGED: one
// period of it).
type FeeRunSkip struct {
	Member MemberRef
	Reason string
	Model  *fees.Model
	Period *fees.Period
}

// FeeRunPlan is the preview of a billing run.
type FeeRunPlan struct {
	Items   []FeeRunItem
	Skipped []FeeRunSkip
}

// TotalCents sums the amounts of the items.
func (p FeeRunPlan) TotalCents() int64 {
	var sum int64
	for _, it := range p.Items {
		sum += it.AmountCents
	}
	return sum
}

// FeeRun is a saved billing run with its live fees.
type FeeRun struct {
	ID         string
	From       time.Time
	To         time.Time
	DueDate    time.Time
	CreatedAt  time.Time
	Count      int
	TotalCents int64
	PaidCount  int
}

const feeModelSelect = `SELECT id, name, amount_cents, interval, min_age, max_age, pro_rata FROM fee_models WHERE deleted_at IS NULL`

func scanFeeModel(r scanner) (fees.Model, error) {
	var m fees.Model
	var interval string
	var minAge, maxAge sql.NullInt64
	var proRata int
	if err := r.Scan(&m.ID, &m.Name, &m.AmountCents, &interval, &minAge, &maxAge, &proRata); err != nil {
		return m, mapErr(err)
	}
	m.Interval, m.ProRata = fees.Interval(interval), proRata != 0
	if minAge.Valid {
		v := int(minAge.Int64)
		m.MinAge = &v
	}
	if maxAge.Valid {
		v := int(maxAge.Int64)
		m.MaxAge = &v
	}
	return m, nil
}

// ListFeeModels returns all live fee models ordered by name.
func (s *Store) ListFeeModels(ctx context.Context) ([]fees.Model, error) {
	return listFeeModels(ctx, s.db)
}

func listFeeModels(ctx context.Context, q queryer) ([]fees.Model, error) {
	rows, err := q.QueryContext(ctx, feeModelSelect+` ORDER BY name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []fees.Model{}
	for rows.Next() {
		m, err := scanFeeModel(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetFeeModel returns a live fee model.
func (s *Store) GetFeeModel(ctx context.Context, id string) (fees.Model, error) {
	return scanFeeModel(s.db.QueryRowContext(ctx, feeModelSelect+` AND id = ?`, id))
}

func feeModelColumns(m fees.Model) row {
	proRata := 0
	if m.ProRata {
		proRata = 1
	}
	return row{"name": m.Name, "amount_cents": m.AmountCents, "interval": string(m.Interval),
		"min_age": intArg(m.MinAge), "max_age": intArg(m.MaxAge), "pro_rata": proRata}
}

// CreateFeeModel inserts a fee model; names are unique.
func (s *Store) CreateFeeModel(ctx context.Context, in fees.Model) (fees.Model, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := w.requireUnique("fee_models", "name", in.Name, "id", id); err != nil {
			return err
		}
		values := feeModelColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("fee_models", values)
	})
	if err != nil {
		return fees.Model{}, err
	}
	return s.GetFeeModel(ctx, id)
}

// UpdateFeeModel replaces the fields of a live fee model. Fees created
// before keep their amount.
func (s *Store) UpdateFeeModel(ctx context.Context, id string, in fees.Model) (fees.Model, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := affected(w.update("fee_models", `id = ? AND deleted_at IS NULL`, []any{id}, feeModelColumns(in))); err != nil {
			return err
		}
		return w.requireUnique("fee_models", "name", in.Name, "id", id)
	})
	if err != nil {
		return fees.Model{}, err
	}
	return s.GetFeeModel(ctx, id)
}

// DeleteFeeModel tombstones a fee model. Members assigned to it fall back to
// the selection by age; fees created from it stay.
func (s *Store) DeleteFeeModel(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		tombstone := row{"deleted_at": db.Now()}
		if err := affected(w.update("fee_models", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)); err != nil {
			return err
		}
		_, err := w.update("member_fee_models", `fee_model_id = ? AND deleted_at IS NULL`, []any{id}, tombstone)
		return err
	})
}

// GetFeeAssignment returns how a live member's fee model is chosen.
func (s *Store) GetFeeAssignment(ctx context.Context, memberID string) (FeeAssignment, error) {
	var one int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM members WHERE id = ? AND deleted_at IS NULL`, memberID).Scan(&one); err != nil {
		return FeeAssignment{}, mapErr(err)
	}
	var modelID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT a.fee_model_id FROM member_fee_models a
		LEFT JOIN fee_models m ON m.id = a.fee_model_id
		WHERE a.member_id = ? AND a.deleted_at IS NULL AND (a.fee_model_id IS NULL OR m.deleted_at IS NULL)`, memberID).Scan(&modelID)
	switch {
	case err == sql.ErrNoRows:
		return FeeAssignment{Mode: FeeModeAuto}, nil
	case err != nil:
		return FeeAssignment{}, err
	case !modelID.Valid:
		return FeeAssignment{Mode: FeeModeExempt}, nil
	}
	return FeeAssignment{Mode: FeeModeModel, ModelID: &modelID.String}, nil
}

// SetFeeAssignment sets how a member's fee model is chosen. FeeModeAuto
// removes the assignment.
func (s *Store) SetFeeAssignment(ctx context.Context, memberID string, a FeeAssignment) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		if err := requireLiveMember(ctx, w.Tx, memberID); err != nil {
			return err
		}
		pk := row{"member_id": memberID}
		switch a.Mode {
		case FeeModeAuto:
			_, err := w.update("member_fee_models", `member_id = ? AND deleted_at IS NULL`, []any{memberID}, row{"deleted_at": db.Now()})
			return err
		case FeeModeExempt:
			return w.upsert("member_fee_models", pk, row{"fee_model_id": nil, "deleted_at": nil, "created_at": db.Now()})
		}
		if a.ModelID == nil {
			return ErrInvalidReference
		}
		var one int
		err := w.QueryRowContext(ctx, `SELECT 1 FROM fee_models WHERE id = ? AND deleted_at IS NULL`, *a.ModelID).Scan(&one)
		if err == sql.ErrNoRows {
			return ErrInvalidReference
		} else if err != nil {
			return err
		}
		return w.upsert("member_fee_models", pk, row{"fee_model_id": *a.ModelID, "deleted_at": nil, "created_at": db.Now()})
	})
}

// feeMember is a live member with what billing needs to know.
type feeMember struct {
	ref        MemberRef
	birthday   *time.Time
	membership fees.Membership
	assignment FeeAssignment
}

// modelFor returns the model that applies to the member in the fee year of
// day on, or nil, with the member's age on 1 January of that year, which
// decides the model. All periods of a year therefore use the same model, so
// a member who comes of age changes model at the turn of the year, without a
// gap or an overlap.
func (m feeMember) modelFor(models []fees.Model, byID map[string]fees.Model, on time.Time) (*fees.Model, *int) {
	var age *int
	if m.birthday != nil {
		a := fees.Age(*m.birthday, fees.YearStart(on))
		age = &a
	}
	switch m.assignment.Mode {
	case FeeModeExempt:
		return nil, age
	case FeeModeModel:
		if model, ok := byID[*m.assignment.ModelID]; ok {
			return &model, age
		}
		return nil, age
	}
	if age == nil {
		return nil, nil
	}
	if model, ok := fees.SelectByAge(models, *age); ok {
		return &model, age
	}
	return nil, age
}

func feeMembers(ctx context.Context, q queryer) ([]feeMember, error) {
	rows, err := q.QueryContext(ctx, `SELECT m.id, m.number, m.first_name, m.last_name, m.birthday, m.entry_date, m.exit_date,
			a.member_id, a.fee_model_id
		FROM members m
		LEFT JOIN member_fee_models a ON a.member_id = m.id AND a.deleted_at IS NULL
			AND (a.fee_model_id IS NULL OR EXISTS (SELECT 1 FROM fee_models f WHERE f.id = a.fee_model_id AND f.deleted_at IS NULL))
		WHERE m.deleted_at IS NULL
		ORDER BY m.last_name, m.first_name, m.number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []feeMember
	for rows.Next() {
		var m feeMember
		var birthday, entry, exit, assigned, modelID sql.NullString
		if err := rows.Scan(&m.ref.ID, &m.ref.Number, &m.ref.FirstName, &m.ref.LastName, &birthday, &entry, &exit,
			&assigned, &modelID); err != nil {
			return nil, err
		}
		m.birthday = dayPtr(nullTime(birthday))
		m.membership = fees.Membership{Entry: dayPtr(nullTime(entry)), Exit: dayPtr(nullTime(exit))}
		switch {
		case !assigned.Valid:
			m.assignment = FeeAssignment{Mode: FeeModeAuto}
		case !modelID.Valid:
			m.assignment = FeeAssignment{Mode: FeeModeExempt}
		default:
			m.assignment = FeeAssignment{Mode: FeeModeModel, ModelID: &modelID.String}
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// dayPtr returns the calendar day of a stored member date. The member dialog
// stores dates as local midnight in UTC (e.g. 1 April in Germany as
// 2027-03-31T22:00Z), so the time is rounded to the nearest midnight, which
// gives the intended day for every time zone within ±12 hours.
func dayPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	d := t.UTC().Add(12 * time.Hour).Truncate(24 * time.Hour)
	return &d
}

// FeeModelsOn returns the fee model of every member who belongs to the club
// on the given day.
func (s *Store) FeeModelsOn(ctx context.Context, on time.Time) ([]MemberFeeModel, error) {
	on = dateOnly(on)
	models, err := listFeeModels(ctx, s.db)
	if err != nil {
		return nil, err
	}
	members, err := feeMembers(ctx, s.db)
	if err != nil {
		return nil, err
	}
	byID := modelsByID(models)
	result := []MemberFeeModel{}
	for _, m := range members {
		if !m.membership.ActiveIn(fees.Period{Start: on, End: on}) {
			continue
		}
		model, age := m.modelFor(models, byID, on)
		result = append(result, MemberFeeModel{Member: m.ref, Mode: m.assignment.Mode, Model: model, Age: age})
	}
	return result, nil
}

func modelsByID(models []fees.Model) map[string]fees.Model {
	byID := make(map[string]fees.Model, len(models))
	for _, m := range models {
		byID[m.ID] = m
	}
	return byID
}

// PlanFeeRun previews a billing run without saving it.
func (s *Store) PlanFeeRun(ctx context.Context, in FeeRunInput) (FeeRunPlan, error) {
	return planFeeRun(ctx, s.db, in)
}

// planFeeRun charges every member, for each period of the model that applies
// in the period's year (modelFor), unless the member does not belong to the club in
// the period or a live fee of a run already covers part of it.
func planFeeRun(ctx context.Context, q queryer, in FeeRunInput) (FeeRunPlan, error) {
	models, err := listFeeModels(ctx, q)
	if err != nil {
		return FeeRunPlan{}, err
	}
	members, err := feeMembers(ctx, q)
	if err != nil {
		return FeeRunPlan{}, err
	}
	charged, err := chargedPeriods(ctx, q)
	if err != nil {
		return FeeRunPlan{}, err
	}
	byID := modelsByID(models)
	due := dateOnly(in.DueDate)
	plan := FeeRunPlan{Items: []FeeRunItem{}, Skipped: []FeeRunSkip{}}

	for _, m := range members {
		// The candidate models: the fixed one, or every model with an age limit.
		var candidates []fees.Model
		switch m.assignment.Mode {
		case FeeModeModel:
			candidates = []fees.Model{byID[*m.assignment.ModelID]}
		case FeeModeAuto:
			for _, model := range models {
				if model.HasAgeLimit() {
					candidates = append(candidates, model)
				}
			}
		}
		type candidate struct {
			model  fees.Model
			period fees.Period
		}
		var periods []candidate
		for _, model := range candidates {
			for _, p := range fees.Periods(model.Interval, in.From, in.To) {
				if !m.membership.ActiveIn(p) {
					continue
				}
				if applies, _ := m.modelFor(models, byID, p.Start); applies != nil && applies.ID == model.ID {
					periods = append(periods, candidate{model, p})
				}
			}
		}
		if len(periods) == 0 {
			if !m.membership.ActiveIn(fees.Period{Start: in.From, End: fees.MonthEnd(in.To)}) {
				continue
			}
			reason := FeeSkipNoModel
			if m.assignment.Mode == FeeModeExempt {
				reason = FeeSkipExempt
			}
			plan.Skipped = append(plan.Skipped, FeeRunSkip{Member: m.ref, Reason: reason})
			continue
		}
		sort.SliceStable(periods, func(i, j int) bool { return periods[i].period.Start.Before(periods[j].period.Start) })
		taken := charged[m.ref.ID]
		for _, c := range periods {
			if overlapsAny(c.period, taken) {
				model, period := c.model, c.period
				plan.Skipped = append(plan.Skipped, FeeRunSkip{Member: m.ref, Reason: FeeSkipAlreadyCharged, Model: &model, Period: &period})
				continue
			}
			taken = append(taken, c.period)
			itemDue := due
			if c.period.Start.After(itemDue) {
				itemDue = c.period.Start
			}
			plan.Items = append(plan.Items, FeeRunItem{Member: m.ref, Model: c.model, Period: c.period,
				AmountCents: fees.Amount(c.model, c.period, m.membership), DueDate: itemDue})
		}
	}
	return plan, nil
}

func overlapsAny(p fees.Period, list []fees.Period) bool {
	for _, o := range list {
		if p.Overlaps(o) {
			return true
		}
	}
	return false
}

// chargedPeriods returns, per member, the periods charged by live fees of runs.
func chargedPeriods(ctx context.Context, q queryer) (map[string][]fees.Period, error) {
	rows, err := q.QueryContext(ctx, `SELECT c.member_id, c.period_start, c.period_end FROM fee_charges c
		JOIN member_fees f ON f.id = c.fee_id AND f.deleted_at IS NULL
		WHERE c.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string][]fees.Period{}
	for rows.Next() {
		var memberID, start, end string
		if err := rows.Scan(&memberID, &start, &end); err != nil {
			return nil, err
		}
		result[memberID] = append(result[memberID], fees.Period{Start: dateOnly(parseTime(start)), End: dateOnly(parseTime(end))})
	}
	return result, rows.Err()
}

// CreateFeeRun creates the fees of a billing run, planned again inside the
// transaction so that a concurrent run cannot charge a period twice.
func (s *Store) CreateFeeRun(ctx context.Context, in FeeRunInput) (FeeRun, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		plan, err := planFeeRun(ctx, w.Tx, in)
		if err != nil {
			return err
		}
		if len(plan.Items) == 0 {
			return ErrFeeRunEmpty
		}
		now := db.Now()
		if err := w.insert("fee_runs", row{"id": id, "period_from": db.FormatTime(dateOnly(in.From)),
			"period_to": db.FormatTime(fees.MonthEnd(in.To)), "due_date": db.FormatTime(dateOnly(in.DueDate)), "created_at": now}); err != nil {
			return err
		}
		for _, it := range plan.Items {
			feeID := db.NewID()
			if err := w.insert("member_fees", row{"id": feeID, "member_id": it.Member.ID, "amount_cents": it.AmountCents,
				"due_date": db.FormatTime(it.DueDate), "description": it.Description(), "year": it.Period.Start.Year(),
				"created_at": now}); err != nil {
				return err
			}
			if err := w.insert("fee_charges", row{"fee_id": feeID, "run_id": id, "member_id": it.Member.ID,
				"fee_model_id": it.Model.ID, "period_start": db.FormatTime(it.Period.Start),
				"period_end": db.FormatTime(it.Period.End), "created_at": now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return FeeRun{}, err
	}
	return s.GetFeeRun(ctx, id)
}

const feeRunSelect = `SELECT r.id, r.period_from, r.period_to, r.due_date, r.created_at,
       COUNT(f.id), COALESCE(SUM(f.amount_cents), 0), COUNT(f.paid_date)
FROM fee_runs r
LEFT JOIN fee_charges c ON c.run_id = r.id AND c.deleted_at IS NULL
LEFT JOIN member_fees f ON f.id = c.fee_id AND f.deleted_at IS NULL
WHERE r.deleted_at IS NULL`

func scanFeeRun(r scanner) (FeeRun, error) {
	var run FeeRun
	var from, to, due, created string
	if err := r.Scan(&run.ID, &from, &to, &due, &created, &run.Count, &run.TotalCents, &run.PaidCount); err != nil {
		return run, mapErr(err)
	}
	run.From, run.To, run.DueDate, run.CreatedAt = parseTime(from), parseTime(to), parseTime(due), parseTime(created)
	return run, nil
}

// GetFeeRun returns a live billing run.
func (s *Store) GetFeeRun(ctx context.Context, id string) (FeeRun, error) {
	return scanFeeRun(s.db.QueryRowContext(ctx, feeRunSelect+` AND r.id = ? GROUP BY r.id`, id))
}

// ListFeeRuns returns the live billing runs, newest first.
func (s *Store) ListFeeRuns(ctx context.Context) ([]FeeRun, error) {
	rows, err := s.db.QueryContext(ctx, feeRunSelect+` GROUP BY r.id ORDER BY r.period_from DESC, r.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []FeeRun{}
	for rows.Next() {
		run, err := scanFeeRun(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, run)
	}
	return result, rows.Err()
}

// DeleteFeeRun removes the open fees of a billing run, so that its periods
// can be charged again. Paid fees stay, and with them the run. It reports
// how many fees were removed.
func (s *Store) DeleteFeeRun(ctx context.Context, id string) (int, error) {
	removed := 0
	err := s.inWriteTx(ctx, func(w *writer) error {
		var one int
		if err := w.QueryRowContext(ctx, `SELECT 1 FROM fee_runs WHERE id = ? AND deleted_at IS NULL`, id).Scan(&one); err != nil {
			return mapErr(err)
		}
		var collecting int
		if err := w.QueryRowContext(ctx, `SELECT COUNT(*) FROM fee_charges c
			JOIN member_fees f ON f.id = c.fee_id AND f.deleted_at IS NULL AND f.paid_date IS NULL
			JOIN sepa_run_items i ON i.fee_id = f.id AND i.deleted_at IS NULL
			JOIN sepa_runs s ON s.id = i.run_id AND s.deleted_at IS NULL AND s.paid_date IS NULL
			WHERE c.run_id = ? AND c.deleted_at IS NULL`, id).Scan(&collecting); err != nil {
			return err
		}
		if collecting > 0 {
			return ErrFeeRunInCollection
		}
		rows, err := w.QueryContext(ctx, `SELECT c.fee_id FROM fee_charges c
			JOIN member_fees f ON f.id = c.fee_id AND f.deleted_at IS NULL AND f.paid_date IS NULL
			WHERE c.run_id = ? AND c.deleted_at IS NULL`, id)
		if err != nil {
			return err
		}
		var open []string
		for rows.Next() {
			var feeID string
			if err := rows.Scan(&feeID); err != nil {
				rows.Close()
				return err
			}
			open = append(open, feeID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		tombstone := row{"deleted_at": db.Now()}
		for _, feeID := range open {
			if _, err := w.update("member_fees", `id = ? AND deleted_at IS NULL`, []any{feeID}, tombstone); err != nil {
				return err
			}
			if _, err := w.update("fee_charges", `fee_id = ? AND deleted_at IS NULL`, []any{feeID}, tombstone); err != nil {
				return err
			}
		}
		removed = len(open)
		var left int
		if err := w.QueryRowContext(ctx, `SELECT COUNT(*) FROM fee_charges c
			JOIN member_fees f ON f.id = c.fee_id AND f.deleted_at IS NULL
			WHERE c.run_id = ? AND c.deleted_at IS NULL`, id).Scan(&left); err != nil {
			return err
		}
		if left == 0 {
			_, err = w.update("fee_runs", `id = ? AND deleted_at IS NULL`, []any{id}, tombstone)
		}
		return err
	})
	return removed, err
}
