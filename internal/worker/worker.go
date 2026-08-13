package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/abdalrahmanattya/incident-command-lab/internal/messaging"
	"github.com/abdalrahmanattya/incident-command-lab/internal/store"
	"github.com/nats-io/nats.go"
	"os"
)

func Run(role string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := os.Getenv("DATABASE_URL")
	if db == "" {
		return fmt.Errorf("DATABASE_URL required for %s worker", role)
	}
	s, err := store.Open(ctx, db)
	if err != nil {
		return err
	}
	defer s.Close()
	b, err := messaging.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		return err
	}
	defer b.Close()
	subject := "incident.events.PaymentCaptured"
	if role == "notification" {
		subject = "incident.events.NotificationRequested"
	}
	if err = b.StartRoleWorker(ctx, role, subject, func(c context.Context, m *nats.Msg) error {
		var e store.Event
		if err := json.Unmarshal(m.Data, &e); err != nil {
			return err
		}
		var x struct {
			ReservationID string `json:"reservation_id"`
		}
		if err = json.Unmarshal(m.Data, &x); err != nil {
			return err
		}
		if role == "payment" {
			if os.Getenv("PAYMENT_FAIL") == "true" || s.FaultEnabled(c, "dependency") {
				return s.Payment(c, e.ID, x.ReservationID, "FAILED", "payment simulator fault")
			}
			if err := s.Payment(c, e.ID, x.ReservationID, "CAPTURED", ""); err != nil {
				return err
			}
		} else {
			if err := s.Notification(c, e.ID, x.ReservationID, "SENT"); err != nil {
				return err
			}
		}
		_, err := s.MarkProcessed(c, e.ID, role)
		return err
	}); err != nil {
		return err
	}
	select {}
}
