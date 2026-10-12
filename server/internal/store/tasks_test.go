package store

import (
	"testing"
	"time"
)

func TestNextOccurrence(t *testing.T) {
	in := TaskInput{Title: "Jahresmeldung", Status: TaskDone, Recurrence: RecurYearly, DueDate: day(2026, 1, 31)}
	next := nextOccurrence(in, TaskOpen)
	if next == nil || next.Status != TaskOpen || !next.DueDate.Equal(*day(2027, 1, 31)) {
		t.Fatalf("yearly: %+v", next)
	}
	in.Recurrence = RecurMonthly
	if next = nextOccurrence(in, TaskOpen); next == nil || !next.DueDate.Equal(*day(2026, 2, 28)) {
		t.Fatalf("monthly clamp: %+v", next)
	}
	in.Recurrence = RecurWeekly
	if next = nextOccurrence(in, TaskInProgress); next == nil || !next.DueDate.Equal(*day(2026, 2, 7)) {
		t.Fatalf("weekly: %+v", next)
	}
	if nextOccurrence(in, TaskDone) != nil {
		t.Error("already done tasks must not recur again")
	}
	in.Recurrence = RecurNone
	if nextOccurrence(in, TaskOpen) != nil {
		t.Error("non-recurring task recurred")
	}
}

func TestFilterTasks(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "mine", Status: TaskOpen, AssigneeMember: &MemberRef{ID: "me"}},
		{ID: "role", Status: TaskInProgress, AssigneeRole: &NamedEntity{ID: "r1"}},
		{ID: "other", Status: TaskOpen, AssigneeMember: &MemberRef{ID: "x"}, DueDate: day(2026, 5, 1)},
		{ID: "done", Status: TaskDone, AssigneeMember: &MemberRef{ID: "me"}, DueDate: day(2026, 5, 1)},
		{ID: "today", Status: TaskOpen, DueDate: day(2026, 6, 1)},
	}
	ids := func(l []Task) string {
		s := ""
		for _, t := range l {
			s += t.ID + " "
		}
		return s
	}
	if got := ids(filterTasks(tasks, TaskFilterMine, "me", map[string]bool{"r1": true}, now)); got != "mine role " {
		t.Errorf("mine = %q", got)
	}
	if got := ids(filterTasks(tasks, TaskFilterMine, "", nil, now)); got != "" {
		t.Errorf("mine without member = %q", got)
	}
	if got := ids(filterTasks(tasks, TaskFilterOverdue, "", nil, now)); got != "other " {
		t.Errorf("overdue = %q", got)
	}
	if got := ids(filterTasks(tasks, TaskFilterDone, "", nil, now)); got != "done " {
		t.Errorf("done = %q", got)
	}
}
