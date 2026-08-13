package analysis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaAdapterParsesStrictJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"response":"{\"summary\":\"ok\",\"hypotheses\":[\"h\"],\"checks\":[\"c\"]}"}`))
	}))
	defer srv.Close()
	t.Setenv("INCIDENT_ANALYST_ALLOW_REMOTE", "true")
	r, err := (Remote{Provider: "ollama", Endpoint: srv.URL, Model: "test"}).Analyze(context.Background(), Evidence{IncidentID: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Provider != "ollama" || !r.Advisory {
		t.Fatalf("unexpected report: %+v", r)
	}
}
