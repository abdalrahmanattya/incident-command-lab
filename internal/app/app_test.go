package app

import (
	"context"
	"testing"
)

func TestReserveIsIdempotentAndEmitsOutbox(t *testing.T) {
	a := New()
	r1, err := a.Reserve(context.Background(), "request-1", "customer-1", "concert", 2)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := a.Reserve(context.Background(), "request-1", "customer-1", "concert", 2)
	if err != nil {
		t.Fatal(err)
	}
	if r1.ID != r2.ID {
		t.Fatalf("idempotency returned %s then %s", r1.ID, r2.ID)
	}
	if len(a.Snapshot().Outbox) != 3 {
		t.Fatalf("expected saga events")
	}
}
func TestDependencyCompensatesStock(t *testing.T) {
	a := New()
	a.SetFault(FaultDependency, true)
	r, err := a.Reserve(context.Background(), "request-2", "customer-1", "concert", 1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "COMPENSATED" {
		t.Fatalf("status=%s", r.Status)
	}
	if got := a.ListProducts()[0].Stock; a.ListProducts()[0].ID == "concert" && got != 100 {
		t.Fatalf("stock was not compensated: %d", got)
	}
}
func TestBacklogRetriesThenDelivers(t *testing.T) {
	a := New()
	a.SetFault(FaultBacklog, true)
	_, _ = a.Reserve(context.Background(), "request-3", "customer-1", "concert", 1)
	for i := 0; i < 3; i++ {
		a.ProcessOutbox(10)
	}
	for _, e := range a.Snapshot().Outbox {
		if e.Status != "DELIVERED" {
			t.Fatalf("event %s status=%s", e.Type, e.Status)
		}
	}
}
