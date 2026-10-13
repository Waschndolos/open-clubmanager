package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMaintenance(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestStore(t)
	member, err := s.CreateMember(ctx, MemberInput{Number: 1, FirstName: "Ole", LastName: "Kite"})
	if err != nil {
		t.Fatal(err)
	}
	item := func(name string) Item {
		it, err := s.CreateItem(ctx, ItemInput{Name: name, Category: "Sicherheit", Quantity: 1, Location: "Slip"})
		if err != nil {
			t.Fatal(err)
		}
		return it
	}
	vest, boat, kite := item("Rettungsweste"), item("Rettungsboot"), item("Kite")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Without a status row an item is ready and can be lent.
	loan, err := s.CreateLoan(ctx, LoanInput{ItemID: kite.ID, MemberID: member.ID, LoanedAt: now})
	if err != nil {
		t.Fatalf("lend ready item: %v", err)
	}

	if err := s.SetItemStatus(ctx, vest.ID, ItemStatus{Condition: ConditionReady, IntervalMonths: ptr(12), NextDue: date("2026-05-15")}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetItemStatus(ctx, boat.ID, ItemStatus{Condition: ConditionDamaged, IntervalMonths: ptr(12), NextDue: date("2026-06-20")}); err != nil {
		t.Fatal(err)
	}

	// A damaged item cannot be lent, nor can an open loan be moved to it.
	if _, err := s.CreateLoan(ctx, LoanInput{ItemID: boat.ID, MemberID: member.ID, LoanedAt: now}); !errors.Is(err, ErrItemNotReady) {
		t.Errorf("lend damaged item: %v, want ErrItemNotReady", err)
	}
	if _, err := s.UpdateLoan(ctx, loan.ID, LoanInput{ItemID: boat.ID, MemberID: member.ID, LoanedAt: now}); !errors.Is(err, ErrItemNotReady) {
		t.Errorf("move loan to damaged item: %v, want ErrItemNotReady", err)
	}
	// Returning a loan of an item that got damaged meanwhile still works.
	if err := s.SetItemStatus(ctx, kite.ID, ItemStatus{Condition: ConditionInRepair}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateLoan(ctx, loan.ID, LoanInput{ItemID: kite.ID, MemberID: member.ID, LoanedAt: now, ReturnedAt: &now}); err != nil {
		t.Errorf("return loan of damaged item: %v", err)
	}

	states, err := s.MaintenanceStates(ctx, now, 30)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ id, due string }{{vest.ID, DueOverdue}, {boat.ID, DueSoon}, {kite.ID, DueNone}}
	for i, w := range want {
		if states[i].ItemID != w.id || states[i].Due != w.due {
			t.Errorf("state %d = %s/%s, want %s/%s", i, states[i].Name, states[i].Due, w.id, w.due)
		}
	}

	// Recording a repair makes the boat ready, moves the due date by the
	// interval and books the cost.
	m, err := s.RecordMaintenance(ctx, boat.ID, MaintenanceInput{Date: *date("2026-06-03"), PerformedBy: ptr("Werft Kiel"),
		Note: ptr("Schlauch geflickt"), Cost: ptr(249.9), Condition: ptr(ConditionReady),
		Expense: &ExpenseInput{Description: "Wartung Rettungsboot", Category: ptr("Material")}})
	if err != nil {
		t.Fatalf("RecordMaintenance: %v", err)
	}
	if m.TransactionID == nil {
		t.Fatal("no expense booked")
	}
	tx, err := s.GetTransaction(ctx, *m.TransactionID)
	if err != nil || tx.Type != "expense" || tx.Amount != 249.9 || tx.Description != "Wartung Rettungsboot" {
		t.Errorf("booked transaction = %+v, %v", tx, err)
	}
	if _, err := s.CreateLoan(ctx, LoanInput{ItemID: boat.ID, MemberID: member.ID, LoanedAt: now}); err != nil {
		t.Errorf("lend repaired boat: %v", err)
	}
	states, _ = s.MaintenanceStates(ctx, now, 30)
	for _, st := range states {
		if st.ItemID == boat.ID {
			if st.NextDue == nil || !st.NextDue.Equal(*date("2027-06-03")) || st.Due != DueOK || st.Condition != ConditionReady ||
				st.LastMaintenance == nil || !st.LastMaintenance.Equal(*date("2026-06-03")) {
				t.Errorf("boat after maintenance = %+v", st)
			}
		}
	}

	// A maintenance of an item without status keeps it ready and books nothing.
	plain := item("Helm")
	m, err = s.RecordMaintenance(ctx, plain.ID, MaintenanceInput{Date: now, Cost: ptr(10.0)})
	if err != nil || m.TransactionID != nil {
		t.Errorf("plain maintenance = %+v, %v", m, err)
	}
	if list, _ := s.ListMaintenance(ctx, boat.ID); len(list) != 1 || *list[0].PerformedBy != "Werft Kiel" {
		t.Errorf("history = %+v", list)
	}

	if err := s.DeleteItem(ctx, boat.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListMaintenance(ctx, boat.ID); len(list) != 0 {
		t.Errorf("history of deleted item kept: %+v", list)
	}
}

func TestDueState(t *testing.T) {
	now := time.Date(2026, 6, 1, 23, 0, 0, 0, time.UTC)
	cases := []struct {
		st   ItemStatus
		want string
	}{
		{ItemStatus{Condition: ConditionReady}, DueNone},
		{ItemStatus{Condition: ConditionReady, NextDue: date("2026-06-01")}, DueSoon},
		{ItemStatus{Condition: ConditionReady, NextDue: date("2026-05-31")}, DueOverdue},
		{ItemStatus{Condition: ConditionReady, NextDue: date("2026-07-01")}, DueSoon},
		{ItemStatus{Condition: ConditionReady, NextDue: date("2026-07-02")}, DueOK},
		{ItemStatus{Condition: ConditionRetired, NextDue: date("2026-01-01")}, DueNone},
	}
	for _, c := range cases {
		if got := dueState(c.st, now, 30); got != c.want {
			t.Errorf("dueState(%+v) = %s, want %s", c.st, got, c.want)
		}
	}
}
