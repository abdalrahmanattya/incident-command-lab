package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/abdalrahmanattya/incident-command-lab/internal/analysis"
	"github.com/abdalrahmanattya/incident-command-lab/internal/store"
	"sort"
	"strings"
	"sync"
	"time"
)

type Fault string

const (
	FaultLatency    Fault = "latency"
	FaultDependency Fault = "dependency"
	FaultBacklog    Fault = "backlog"
	FaultDuplicate  Fault = "duplicate"
	FaultDB         Fault = "database"
	FaultBadRelease Fault = "bad-release"
)

type Product struct {
	ID, Name   string
	PriceCents int `json:"price_cents"`
	Stock      int `json:"stock"`
}
type Reservation struct {
	ID, IdempotencyKey, ProductID, CustomerID, Status, Release string
	Quantity                                                   int
	TotalCents                                                 int `json:"total_cents"`
	CreatedAt                                                  time.Time
	UpdatedAt                                                  time.Time
	Failure                                                    string `json:"failure,omitempty"`
}
type Event struct {
	ID, Type, AggregateID, Status string
	Attempts                      int
	CreatedAt                     time.Time
	LastError                     string `json:"last_error,omitempty"`
}
type Incident struct {
	ID, Title, Status, StartedAt, EndedAt string
	Severity                              string
	Signals                               []string
	Timeline                              []string
	Runbooks                              []string
}
type State struct {
	Products     []Product     `json:"products"`
	Reservations []Reservation `json:"reservations"`
	Outbox       []Event       `json:"outbox"`
	Incidents    []Incident    `json:"incidents"`
	Faults       []Fault       `json:"faults"`
}

type App struct {
	mu           sync.Mutex
	products     map[string]Product
	reservations map[string]Reservation
	byKey        map[string]string
	events       map[string]Event
	incidents    map[string]Incident
	faults       map[Fault]bool
	release      string
	repo         *store.Postgres
}

func id() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func New() *App {
	return &App{products: map[string]Product{"concert": {ID: "concert", Name: "Concert ticket", PriceCents: 2500, Stock: 100}, "workshop": {ID: "workshop", Name: "Reliability workshop", PriceCents: 7500, Stock: 20}}, reservations: map[string]Reservation{}, byKey: map[string]string{}, events: map[string]Event{}, incidents: map[string]Incident{}, faults: map[Fault]bool{}, release: "v1.0.0"}
}
func NewWithStore(repo *store.Postgres) *App { a := New(); a.repo = repo; return a }
func (a *App) SetFault(f Fault, on bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if on {
		a.faults[f] = true
	} else {
		delete(a.faults, f)
	}
	if a.repo != nil {
		_ = a.repo.SetFault(context.Background(), string(f), on)
	}
}
func (a *App) Faults() []Fault {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []Fault{}
	for f := range a.faults {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func (a *App) Has(f Fault) bool      { return a.faults[f] }
func (a *App) HasFault(f Fault) bool { a.mu.Lock(); defer a.mu.Unlock(); return a.faults[f] }
func (a *App) Reserve(ctx context.Context, key, customer, product string, qty int) (Reservation, error) {
	if a.repo != nil {
		if a.repo.FaultEnabled(ctx, string(FaultLatency)) {
			select {
			case <-time.After(150 * time.Millisecond):
			case <-ctx.Done():
				return Reservation{}, ctx.Err()
			}
		}
		if a.repo.FaultEnabled(ctx, string(FaultDB)) {
			return Reservation{}, fmt.Errorf("database unavailable")
		}
		if a.repo.FaultEnabled(ctx, string(FaultDependency)) {
			return Reservation{ID: id(), IdempotencyKey: key, ProductID: product, CustomerID: customer, Status: "COMPENSATED", Quantity: qty, Release: a.release, Failure: "payment dependency unavailable", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		}
		r, err := a.repo.Reserve(ctx, key, customer, product, qty)
		return Reservation{ID: r.ID, IdempotencyKey: r.Key, ProductID: r.ProductID, CustomerID: r.CustomerID, Status: r.Status, Quantity: r.Quantity, TotalCents: r.TotalCents, Release: r.Release, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Failure: r.Failure}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if key == "" {
		return Reservation{}, fmt.Errorf("idempotency key is required")
	}
	if qty < 1 || qty > 10 {
		return Reservation{}, fmt.Errorf("quantity must be between 1 and 10")
	}
	if id, ok := a.byKey[key]; ok {
		return a.reservations[id], nil
	}
	if a.faults[FaultLatency] {
		select {
		case <-time.After(150 * time.Millisecond):
		case <-ctx.Done():
			return Reservation{}, ctx.Err()
		}
	}
	p, ok := a.products[product]
	if !ok {
		return Reservation{}, fmt.Errorf("product not found")
	}
	if a.faults[FaultDB] {
		return Reservation{}, fmt.Errorf("database unavailable")
	}
	if p.Stock < qty {
		return Reservation{}, fmt.Errorf("insufficient stock")
	}
	now := time.Now().UTC()
	r := Reservation{ID: id(), IdempotencyKey: key, ProductID: product, CustomerID: customer, Status: "CONFIRMED", Quantity: qty, TotalCents: p.PriceCents * qty, Release: a.release, CreatedAt: now, UpdatedAt: now}
	if a.faults[FaultDependency] {
		r.Status = "COMPENSATED"
		r.Failure = "payment dependency unavailable"
	}
	p.Stock -= qty
	a.products[product] = p
	a.reservations[r.ID] = r
	a.byKey[key] = r.ID
	a.addEvent("ReservationCreated", r.ID)
	if r.Status == "CONFIRMED" {
		a.addEvent("PaymentCaptured", r.ID)
		a.addEvent("NotificationRequested", r.ID)
	} else {
		a.addEvent("ReservationCompensated", r.ID)
		p.Stock += qty
		a.products[product] = p
	}
	return r, nil
}
func (a *App) addEvent(t, aggregate string) {
	e := Event{ID: id(), Type: t, AggregateID: aggregate, Status: "PENDING", CreatedAt: time.Now().UTC()}
	if a.faults[FaultDuplicate] {
		e.Attempts = 1
	}
	a.events[e.ID] = e
}
func (a *App) ListProducts() []Product {
	if a.repo != nil {
		xs, err := a.repo.Products(context.Background())
		if err == nil {
			out := make([]Product, len(xs))
			for i, x := range xs {
				out[i] = Product{ID: x.ID, Name: x.Name, PriceCents: x.PriceCents, Stock: x.Stock}
			}
			return out
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []Product{}
	for _, p := range a.products {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (a *App) GetReservation(id string) (Reservation, bool) {
	if a.repo != nil {
		r, ok := a.repo.GetReservation(context.Background(), id)
		return Reservation{ID: r.ID, IdempotencyKey: r.Key, ProductID: r.ProductID, CustomerID: r.CustomerID, Status: r.Status, Quantity: r.Quantity, TotalCents: r.TotalCents, Release: r.Release, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Failure: r.Failure}, ok
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	r, ok := a.reservations[id]
	return r, ok
}
func (a *App) Cancel(id string) (Reservation, error) {
	if a.repo != nil {
		r, err := a.repo.Cancel(context.Background(), id)
		return Reservation{ID: r.ID, IdempotencyKey: r.Key, ProductID: r.ProductID, CustomerID: r.CustomerID, Status: r.Status, Quantity: r.Quantity, TotalCents: r.TotalCents, Release: r.Release, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Failure: r.Failure}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	r, ok := a.reservations[id]
	if !ok {
		return Reservation{}, fmt.Errorf("reservation not found")
	}
	if r.Status == "CANCELLED" || r.Status == "COMPENSATED" {
		return r, nil
	}
	p := a.products[r.ProductID]
	p.Stock += r.Quantity
	a.products[p.ID] = p
	r.Status = "CANCELLED"
	r.UpdatedAt = time.Now().UTC()
	a.reservations[id] = r
	a.addEvent("ReservationCancelled", id)
	return r, nil
}
func (a *App) ProcessOutbox(limit int) []Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []Event{}
	for id, e := range a.events {
		if len(out) >= limit {
			break
		}
		if e.Status == "PENDING" || e.Status == "RETRY" {
			e.Attempts++
			if a.faults[FaultBacklog] && e.Attempts < 3 {
				e.Status = "RETRY"
				e.LastError = "simulated consumer backlog"
			} else {
				e.Status = "DELIVERED"
			}
			a.events[id] = e
			out = append(out, e)
		}
	}
	return out
}
func (a *App) CreateIncident(title, severity string) Incident {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := Incident{ID: id(), Title: title, Severity: severity, Status: "OPEN", StartedAt: time.Now().UTC().Format(time.RFC3339), Signals: []string{}, Timeline: []string{"incident opened"}, Runbooks: []string{"runbooks/reservation-dependency.md"}}
	for f := range a.faults {
		i.Signals = append(i.Signals, string(f)+" fault enabled")
	}
	sort.Strings(i.Signals)
	a.incidents[i.ID] = i
	return i
}
func (a *App) Incident(id string) (Incident, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	i, ok := a.incidents[id]
	return i, ok
}
func (a *App) ListIncidents() []Incident {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Incident, 0, len(a.incidents))
	for _, i := range a.incidents {
		out = append(out, i)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt == out[j].StartedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].StartedAt > out[j].StartedAt
	})
	return out
}
func (a *App) Evidence(id string) (analysis.Evidence, error) {
	i, ok := a.Incident(id)
	if !ok {
		return analysis.Evidence{}, fmt.Errorf("incident not found")
	}
	return analysis.Evidence{IncidentID: i.ID, Timeline: i.Timeline, Signals: i.Signals, Runbooks: i.Runbooks}, nil
}
func (a *App) Snapshot() State {
	if a.repo != nil {
		ps, rs, es, err := a.repo.Snapshot(context.Background())
		if err == nil {
			s := State{Faults: a.Faults()}
			for _, p := range ps {
				s.Products = append(s.Products, Product{ID: p.ID, Name: p.Name, PriceCents: p.PriceCents, Stock: p.Stock})
			}
			for _, r := range rs {
				s.Reservations = append(s.Reservations, Reservation{ID: r.ID, IdempotencyKey: r.Key, ProductID: r.ProductID, CustomerID: r.CustomerID, Status: r.Status, Quantity: r.Quantity, TotalCents: r.TotalCents, Release: r.Release, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Failure: r.Failure})
			}
			for _, e := range es {
				s.Outbox = append(s.Outbox, Event{ID: e.ID, Type: e.Type, AggregateID: e.AggregateID, Status: e.Status, Attempts: e.Attempts, CreatedAt: e.CreatedAt, LastError: e.LastError})
			}
			return s
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s := State{Faults: []Fault{}}
	for _, p := range a.products {
		s.Products = append(s.Products, p)
	}
	for _, r := range a.reservations {
		s.Reservations = append(s.Reservations, r)
	}
	for _, e := range a.events {
		s.Outbox = append(s.Outbox, e)
	}
	for _, i := range a.incidents {
		s.Incidents = append(s.Incidents, i)
	}
	for f := range a.faults {
		s.Faults = append(s.Faults, f)
	}
	return s
}
func ValidateTitle(s string) bool { return len(strings.TrimSpace(s)) >= 3 && len(s) <= 120 }
