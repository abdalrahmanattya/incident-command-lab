package server

import (
	"github.com/abdalrahmanattya/incident-command-lab/internal/app"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidReservationAndFaultRoutes(t *testing.T) {
	h := New(app.New()).Handler()
	r := httptest.NewRequest(http.MethodPost, "/v1/reservations", strings.NewReader(`{"customer_id":"x","product_id":"concert","quantity":1}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatalf("missing idempotency code=%d", w.Code)
	}
	r = httptest.NewRequest(http.MethodPost, "/ops/faults", strings.NewReader(`{"fault":"unknown","enabled":true}`))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("unknown fault code=%d", w.Code)
	}
}

func TestIncidentListRoute(t *testing.T) {
	a := app.New()
	h := New(a).Handler()
	create := httptest.NewRequest(http.MethodPost, "/ops/incidents", strings.NewReader(`{"title":"Queue pressure","severity":"SEV2"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, create)
	if w.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", w.Code, w.Body.String())
	}
	list := httptest.NewRequest(http.MethodGet, "/ops/incidents", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, list)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Queue pressure") {
		t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
	}
}
