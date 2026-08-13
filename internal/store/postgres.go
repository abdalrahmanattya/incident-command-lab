package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Product struct {
	ID, Name          string
	PriceCents, Stock int
}
type Reservation struct {
	ID, Key, ProductID, CustomerID, Status, Release, Failure string
	Quantity, TotalCents                                     int
	CreatedAt, UpdatedAt                                     time.Time
}
type Event struct {
	ID, Type, AggregateID, Status, LastError string
	Attempts                                 int
	CreatedAt                                time.Time
}

func (p *Postgres) Cancel(ctx context.Context, id string) (Reservation, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback(ctx)
	var r Reservation
	err = tx.QueryRow(ctx, `SELECT id,key,product_id,customer_id,status,quantity,total_cents,release,created_at,updated_at,COALESCE(failure,'') FROM reservations WHERE id=$1 FOR UPDATE`, id).Scan(&r.ID, &r.Key, &r.ProductID, &r.CustomerID, &r.Status, &r.Quantity, &r.TotalCents, &r.Release, &r.CreatedAt, &r.UpdatedAt, &r.Failure)
	if err != nil {
		return Reservation{}, fmt.Errorf("reservation not found")
	}
	if r.Status == "CANCELLED" || r.Status == "COMPENSATED" {
		return r, tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `UPDATE products SET stock=stock+$1 WHERE id=$2`, r.Quantity, r.ProductID); err != nil {
		return Reservation{}, err
	}
	r.Status = "CANCELLED"
	r.UpdatedAt = time.Now().UTC()
	if _, err = tx.Exec(ctx, `UPDATE reservations SET status='CANCELLED',updated_at=$2 WHERE id=$1`, id, r.UpdatedAt); err != nil {
		return Reservation{}, err
	}
	payload, _ := json.Marshal(map[string]any{"reservation_id": id, "type": "ReservationCancelled"})
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,aggregate_id,event_type,payload,status) VALUES($1,$2,'ReservationCancelled',$3,'PENDING')`, uuid.NewString(), id, payload); err != nil {
		return Reservation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	return r, nil
}
func (p *Postgres) Snapshot(ctx context.Context) ([]Product, []Reservation, []Event, error) {
	ps, err := p.Products(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	rows, err := p.Pool.Query(ctx, `SELECT id,key,product_id,customer_id,status,quantity,total_cents,release,created_at,updated_at,COALESCE(failure,'') FROM reservations ORDER BY created_at`)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	var rs []Reservation
	for rows.Next() {
		var r Reservation
		if err := rows.Scan(&r.ID, &r.Key, &r.ProductID, &r.CustomerID, &r.Status, &r.Quantity, &r.TotalCents, &r.Release, &r.CreatedAt, &r.UpdatedAt, &r.Failure); err != nil {
			return nil, nil, nil, err
		}
		rs = append(rs, r)
	}
	erows, err := p.Pool.Query(ctx, `SELECT id,aggregate_id,event_type,status,attempts,created_at,COALESCE(last_error,'') FROM outbox_events ORDER BY created_at`)
	if err != nil {
		return nil, nil, nil, err
	}
	defer erows.Close()
	var es []Event
	for erows.Next() {
		var e Event
		if err := erows.Scan(&e.ID, &e.AggregateID, &e.Type, &e.Status, &e.Attempts, &e.CreatedAt, &e.LastError); err != nil {
			return nil, nil, nil, err
		}
		es = append(es, e)
	}
	return ps, rs, es, erows.Err()
}
func (p *Postgres) MarkProcessed(ctx context.Context, eventID, consumer string) (bool, error) {
	r, err := p.Pool.Exec(ctx, `INSERT INTO processed_events(event_id,consumer) VALUES($1,$2) ON CONFLICT DO NOTHING`, eventID, consumer)
	return r.RowsAffected() == 1, err
}
func (p *Postgres) Payment(ctx context.Context, eventID, reservationID, status, reason string) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO payment_outcomes(event_id,reservation_id,status,reason) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, eventID, reservationID, status, reason); err != nil {
		return err
	}
	if status == "FAILED" {
		if _, err = tx.Exec(ctx, `UPDATE products p SET stock=p.stock+r.quantity FROM reservations r WHERE r.id=$1 AND r.status='CONFIRMED' AND p.id=r.product_id`, reservationID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE reservations SET status='COMPENSATED',failure=$2,updated_at=now() WHERE id=$1 AND status='CONFIRMED'`, reservationID, reason); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"reservation_id": reservationID, "type": "ReservationCompensated"})
		if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,aggregate_id,event_type,payload,status) VALUES($1,$2,'ReservationCompensated',$3,'PENDING')`, uuid.NewString(), reservationID, payload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (p *Postgres) Notification(ctx context.Context, eventID, reservationID, status string) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO notification_outcomes(event_id,reservation_id,status) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, eventID, reservationID, status)
	return err
}
func (p *Postgres) SetFault(ctx context.Context, name string, on bool) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO fault_flags(name,enabled) VALUES($1,$2) ON CONFLICT(name) DO UPDATE SET enabled=$2,updated_at=now()`, name, on)
	return err
}
func (p *Postgres) FaultEnabled(ctx context.Context, name string) bool {
	var on bool
	return p.Pool.QueryRow(ctx, `SELECT enabled FROM fault_flags WHERE name=$1`, name).Scan(&on) == nil && on
}

type Postgres struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Postgres, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Postgres{Pool: p}, nil
}
func (p *Postgres) Close() { p.Pool.Close() }
func (p *Postgres) Products(ctx context.Context) ([]Product, error) {
	rows, err := p.Pool.Query(ctx, `SELECT id,name,price_cents,stock FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var x Product
		if err := rows.Scan(&x.ID, &x.Name, &x.PriceCents, &x.Stock); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Reserve atomically claims stock, writes idempotency state and all saga events.
func (p *Postgres) Reserve(ctx context.Context, key, customer, product string, qty int) (Reservation, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback(ctx)
	var old Reservation
	err = tx.QueryRow(ctx, `SELECT id,key,product_id,customer_id,status,quantity,total_cents,release,created_at,updated_at,COALESCE(failure,'') FROM reservations WHERE key=$1`, key).Scan(&old.ID, &old.Key, &old.ProductID, &old.CustomerID, &old.Status, &old.Quantity, &old.TotalCents, &old.Release, &old.CreatedAt, &old.UpdatedAt, &old.Failure)
	if err == nil {
		return old, tx.Commit(ctx)
	}
	var price, stock int
	if err = tx.QueryRow(ctx, `SELECT price_cents,stock FROM products WHERE id=$1 FOR UPDATE`, product).Scan(&price, &stock); err != nil {
		return Reservation{}, fmt.Errorf("product unavailable: %w", err)
	}
	if stock < qty {
		return Reservation{}, fmt.Errorf("insufficient stock")
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	r := Reservation{ID: id, Key: key, ProductID: product, CustomerID: customer, Status: "CONFIRMED", Quantity: qty, TotalCents: price * qty, Release: "v1.0.0", CreatedAt: now, UpdatedAt: now}
	if _, err = tx.Exec(ctx, `UPDATE products SET stock=stock-$1 WHERE id=$2`, qty, product); err != nil {
		return Reservation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO reservations(id,key,product_id,customer_id,status,quantity,total_cents,release,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, r.ID, key, product, customer, r.Status, qty, r.TotalCents, r.Release, now); err != nil {
		return Reservation{}, err
	}
	for _, typ := range []string{"ReservationCreated", "PaymentCaptured", "NotificationRequested"} {
		payload, _ := json.Marshal(map[string]any{"reservation_id": r.ID, "type": typ})
		if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,aggregate_id,event_type,payload,status) VALUES($1,$2,$3,$4,'PENDING')`, uuid.NewString(), r.ID, typ, payload); err != nil {
			return Reservation{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Reservation{}, err
	}
	return r, nil
}
func (p *Postgres) GetReservation(ctx context.Context, id string) (Reservation, bool) {
	var r Reservation
	err := p.Pool.QueryRow(ctx, `SELECT id,key,product_id,customer_id,status,quantity,total_cents,release,created_at,updated_at,COALESCE(failure,'') FROM reservations WHERE id=$1`, id).Scan(&r.ID, &r.Key, &r.ProductID, &r.CustomerID, &r.Status, &r.Quantity, &r.TotalCents, &r.Release, &r.CreatedAt, &r.UpdatedAt, &r.Failure)
	return r, err == nil
}
func (p *Postgres) PublishPending(ctx context.Context, limit int, publish func(context.Context, Event) error) error {
	rows, err := p.Pool.Query(ctx, `SELECT id,aggregate_id,event_type,status,attempts,created_at,COALESCE(last_error,'') FROM outbox_events WHERE status IN ('PENDING','RETRY') ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.Type, &e.Status, &e.Attempts, &e.CreatedAt, &e.LastError); err != nil {
			return err
		}
		if err := publish(ctx, e); err != nil {
			_, _ = p.Pool.Exec(ctx, `UPDATE outbox_events SET status=CASE WHEN attempts+1>=5 THEN 'DLQ' ELSE 'RETRY' END,attempts=attempts+1,last_error=$2 WHERE id=$1`, e.ID, err.Error())
		} else {
			_, _ = p.Pool.Exec(ctx, `UPDATE outbox_events SET status='DELIVERED',attempts=attempts+1 WHERE id=$1`, e.ID)
		}
	}
	return rows.Err()
}
