package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

// Task statuses.
const (
	TaskOpen       = "open"
	TaskInProgress = "in_progress"
	TaskDone       = "done"
)

// Task recurrences.
const (
	RecurNone    = "none"
	RecurWeekly  = "weekly"
	RecurMonthly = "monthly"
	RecurYearly  = "yearly"
)

// Task is a to-do of the board.
type Task struct {
	ID             string
	Title          string
	Description    *string
	DueDate        *time.Time
	Status         string
	Recurrence     string
	AssigneeMember *MemberRef
	AssigneeRole   *NamedEntity
	LinkedMember   *MemberRef
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TaskInput is the writable part of a task.
type TaskInput struct {
	Title            string
	Description      *string
	DueDate          *time.Time
	Status           string
	Recurrence       string
	AssigneeMemberID *string
	AssigneeRoleID   *string
	LinkedMemberID   *string
}

// Task list filters.
const (
	TaskFilterAll     = "all"
	TaskFilterMine    = "mine"
	TaskFilterOverdue = "overdue"
	TaskFilterDone    = "done"
)

const taskSelect = `SELECT t.id, t.title, t.description, t.due_date, t.status, t.recurrence, t.created_at, t.updated_at,
       am.id, am.number, am.first_name, am.last_name, r.id, r.name, lm.id, lm.number, lm.first_name, lm.last_name
FROM tasks t
LEFT JOIN members am ON am.id = t.assignee_member_id AND am.deleted_at IS NULL
LEFT JOIN roles r ON r.id = t.assignee_role_id AND r.deleted_at IS NULL
LEFT JOIN members lm ON lm.id = t.linked_member_id AND lm.deleted_at IS NULL
WHERE t.deleted_at IS NULL`

func scanTask(r scanner) (Task, error) {
	var t Task
	var desc, due sql.NullString
	var created, updated string
	var amID, lmID, roleID, roleName sql.NullString
	var amNum, lmNum sql.NullInt64
	var amFirst, amLast, lmFirst, lmLast sql.NullString
	err := r.Scan(&t.ID, &t.Title, &desc, &due, &t.Status, &t.Recurrence, &created, &updated,
		&amID, &amNum, &amFirst, &amLast, &roleID, &roleName, &lmID, &lmNum, &lmFirst, &lmLast)
	if err != nil {
		return t, mapErr(err)
	}
	t.Description, t.DueDate = nullStr(desc), nullTime(due)
	t.CreatedAt, t.UpdatedAt = parseTime(created), parseTime(updated)
	if amID.Valid {
		t.AssigneeMember = &MemberRef{ID: amID.String, Number: int(amNum.Int64), FirstName: amFirst.String, LastName: amLast.String}
	}
	if roleID.Valid {
		t.AssigneeRole = &NamedEntity{ID: roleID.String, Name: roleName.String}
	}
	if lmID.Valid {
		t.LinkedMember = &MemberRef{ID: lmID.String, Number: int(lmNum.Int64), FirstName: lmFirst.String, LastName: lmLast.String}
	}
	return t, nil
}

// GetTask returns a live task.
func (s *Store) GetTask(ctx context.Context, id string) (Task, error) {
	return scanTask(s.db.QueryRowContext(ctx, taskSelect+` AND t.id = ?`, id))
}

// ListTasks returns tasks matching the filter: "mine" are open tasks assigned
// to the member whose e-mail is email or to one of that member's roles;
// "overdue" are open tasks due before today; "done" are finished ones.
// Open tasks come first, ordered by due date (undated last).
func (s *Store) ListTasks(ctx context.Context, filter, email string, now time.Time) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, taskSelect+` ORDER BY (t.status = 'done'), t.due_date IS NULL, t.due_date, t.title`)
	if err != nil {
		return nil, err
	}
	var all []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var myMember string
	myRoles := map[string]bool{}
	if filter == TaskFilterMine && email != "" {
		err := s.db.QueryRowContext(ctx, `SELECT member_id FROM member_contacts c JOIN members m ON m.id = c.member_id
			WHERE c.deleted_at IS NULL AND m.deleted_at IS NULL AND lower(c.email) = lower(?)`, email).Scan(&myMember)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if myMember != "" {
			rrows, err := s.db.QueryContext(ctx, `SELECT role_id FROM member_roles WHERE member_id = ? AND deleted_at IS NULL`, myMember)
			if err != nil {
				return nil, err
			}
			defer rrows.Close()
			for rrows.Next() {
				var id string
				if err := rrows.Scan(&id); err != nil {
					return nil, err
				}
				myRoles[id] = true
			}
		}
	}
	return filterTasks(all, filter, myMember, myRoles, now), nil
}

func filterTasks(all []Task, filter, myMember string, myRoles map[string]bool, now time.Time) []Task {
	today := dateOnly(now)
	result := []Task{}
	for _, t := range all {
		keep := true
		switch filter {
		case TaskFilterMine:
			keep = t.Status != TaskDone && myMember != "" &&
				((t.AssigneeMember != nil && t.AssigneeMember.ID == myMember) || (t.AssigneeRole != nil && myRoles[t.AssigneeRole.ID]))
		case TaskFilterOverdue:
			keep = t.Status != TaskDone && t.DueDate != nil && dateOnly(*t.DueDate).Before(today)
		case TaskFilterDone:
			keep = t.Status == TaskDone
		}
		if keep {
			result = append(result, t)
		}
	}
	return result
}

func taskColumns(in TaskInput) row {
	return row{
		"title": in.Title, "description": strArg(in.Description), "due_date": timeArg(in.DueDate),
		"status": in.Status, "recurrence": in.Recurrence,
		"assignee_member_id": strArg(in.AssigneeMemberID), "assignee_role_id": strArg(in.AssigneeRoleID),
		"linked_member_id": strArg(in.LinkedMemberID),
	}
}

func requireRefs(ctx context.Context, w *writer, in TaskInput) error {
	for _, id := range []*string{in.AssigneeMemberID, in.LinkedMemberID} {
		if id != nil {
			if err := requireLiveMember(ctx, w.Tx, *id); err != nil {
				return err
			}
		}
	}
	if in.AssigneeRoleID != nil {
		var one int
		err := w.QueryRowContext(ctx, `SELECT 1 FROM roles WHERE id = ? AND deleted_at IS NULL`, *in.AssigneeRoleID).Scan(&one)
		if err == sql.ErrNoRows {
			return ErrInvalidReference
		}
		return err
	}
	return nil
}

// CreateTask inserts a task.
func (s *Store) CreateTask(ctx context.Context, in TaskInput) (Task, error) {
	id := db.NewID()
	err := s.inWriteTx(ctx, func(w *writer) error {
		if err := requireRefs(ctx, w, in); err != nil {
			return err
		}
		values := taskColumns(in)
		values["id"], values["created_at"] = id, db.Now()
		return w.insert("tasks", values)
	})
	if err != nil {
		return Task{}, err
	}
	return s.GetTask(ctx, id)
}

// UpdateTask replaces the fields of a live task. Finishing a recurring task
// that has a due date creates the next occurrence. Only changed fields are
// written, so concurrent edits of different fields (or a status change next
// to an edit of the title) merge in sync.
func (s *Store) UpdateTask(ctx context.Context, id string, in TaskInput) (Task, error) {
	err := s.inWriteTx(ctx, func(w *writer) error {
		var prev string
		err := w.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ? AND deleted_at IS NULL`, id).Scan(&prev)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := requireRefs(ctx, w, in); err != nil {
			return err
		}
		if err := affected(w.update("tasks", `id = ? AND deleted_at IS NULL`, []any{id}, taskColumns(in))); err != nil {
			return err
		}
		if next := nextOccurrence(in, prev); next != nil {
			values := taskColumns(*next)
			values["id"], values["created_at"] = db.NewID(), db.Now()
			return w.insert("tasks", values)
		}
		return nil
	})
	if err != nil {
		return Task{}, err
	}
	return s.GetTask(ctx, id)
}

// nextOccurrence returns the follow-up of a recurring task that has just been
// finished, or nil.
func nextOccurrence(in TaskInput, prevStatus string) *TaskInput {
	if in.Status != TaskDone || prevStatus == TaskDone || in.DueDate == nil {
		return nil
	}
	var due time.Time
	switch in.Recurrence {
	case RecurWeekly:
		due = in.DueDate.AddDate(0, 0, 7)
	case RecurMonthly:
		due = addMonthClamped(*in.DueDate, 1)
	case RecurYearly:
		due = addMonthClamped(*in.DueDate, 12)
	default:
		return nil
	}
	next := in
	next.Status, next.DueDate = TaskOpen, &due
	return &next
}

// addMonthClamped adds months, keeping the day within the target month
// (31 January + 1 month = 28/29 February).
func addMonthClamped(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, t.Hour(), t.Minute(), t.Second(), 0, t.Location()).AddDate(0, months, 0)
	last := first.AddDate(0, 1, -1).Day()
	day := t.Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

// DeleteTask tombstones a task.
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	return s.inWriteTx(ctx, func(w *writer) error {
		return affected(w.update("tasks", `id = ? AND deleted_at IS NULL`, []any{id}, row{"deleted_at": db.Now()}))
	})
}

// ValidTaskStatus reports whether s is a task status.
func ValidTaskStatus(s string) bool {
	return s == TaskOpen || s == TaskInProgress || s == TaskDone
}

// ValidRecurrence reports whether s is a task recurrence.
func ValidRecurrence(s string) bool {
	return strings.Contains(" none weekly monthly yearly ", " "+s+" ") && s != ""
}
