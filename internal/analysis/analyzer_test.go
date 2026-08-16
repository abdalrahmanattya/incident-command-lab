package analysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testEvidence() Evidence {
	return Evidence{IncidentID: "i", Timeline: []string{"incident opened"}, Signals: []string{"backlog fault enabled"}, Runbooks: []string{"runbooks/reservation-dependency.md"}}
}

func TestDeterministicUsesExactEvidenceCitations(t *testing.T) {
	report, err := (Deterministic{}).Analyze(context.Background(), testEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "deterministic" || len(report.Hypotheses) == 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	allowed := map[string]bool{"incident opened": true, "backlog fault enabled": true, "runbooks/reservation-dependency.md": true}
	for _, hypothesis := range report.Hypotheses {
		if hypothesis.Confidence < 0 || hypothesis.Confidence > 1 {
			t.Fatalf("confidence out of range: %+v", hypothesis)
		}
		for _, citation := range hypothesis.Evidence {
			if !allowed[citation] {
				t.Fatalf("invented citation %q", citation)
			}
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"Summary"`) || !strings.Contains(string(encoded), `"summary"`) {
		t.Fatalf("report is not lower-case JSON: %s", encoded)
	}
}

func TestOllamaAdapterParsesStrictJSONAndLowercaseSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"model":"test","created_at":"2026-08-16T00:00:00Z","response":"{\"summary\":\"ok\",\"hypotheses\":[{\"title\":\"Queue pressure\",\"confidence\":0.7,\"evidence\":[\"backlog fault enabled\"]}],\"checks\":[\"backlog fault enabled\"]}","done":true}`))
	}))
	defer srv.Close()
	t.Setenv("INCIDENT_ANALYST_ALLOW_REMOTE", "true")
	report, err := (Remote{Provider: "ollama", Endpoint: srv.URL, Model: "test"}).Analyze(context.Background(), testEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "ollama" || !report.Advisory || report.Hypotheses[0].Evidence[0] != "backlog fault enabled" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRemoteRejectsInvalidAdvisoryResponses(t *testing.T) {
	cases := map[string]string{
		"malformed":        `{"summary":`,
		"trailing":         `{"summary":"ok","hypotheses":[],"checks":["safe"]} trailing`,
		"unknown field":    `{"summary":"ok","hypotheses":[],"checks":["safe"],"extra":true}`,
		"missing summary":  `{"hypotheses":[],"checks":["safe"]}`,
		"blank check":      `{"summary":"ok","hypotheses":[],"checks":[" "]}`,
		"unsafe check":     `{"summary":"ok","hypotheses":[],"checks":["run command kubectl delete"]}`,
		"bad confidence":   `{"summary":"ok","hypotheses":[{"title":"x","confidence":1.1,"evidence":["incident opened"]}],"checks":["safe"]}`,
		"empty citation":   `{"summary":"ok","hypotheses":[{"title":"x","confidence":0.5,"evidence":[""]}],"checks":["safe"]}`,
		"unknown citation": `{"summary":"ok","hypotheses":[{"title":"x","confidence":0.5,"evidence":["not supplied"]}],"checks":["safe"]}`,
		"empty hypotheses": `{"summary":"ok","hypotheses":[],"checks":["safe"]}`,
		"empty checks":     `{"summary":"ok","hypotheses":[{"title":"x","confidence":0.5,"evidence":["incident opened"]}],"checks":[]}`,
	}
	for name, advisory := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"response":` + quote(advisory) + `}`))
			}))
			defer srv.Close()
			t.Setenv("INCIDENT_ANALYST_ALLOW_REMOTE", "true")
			if _, err := (Remote{Provider: "ollama", Endpoint: srv.URL}).Analyze(context.Background(), testEvidence()); err == nil {
				t.Fatal("expected invalid response error")
			}
		})
	}
}

func TestRemoteHTTPFailureReturnsErrorForServerFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusBadGateway) }))
	defer srv.Close()
	t.Setenv("INCIDENT_ANALYST_ALLOW_REMOTE", "true")
	if _, err := (Remote{Provider: "ollama", Endpoint: srv.URL}).Analyze(context.Background(), testEvidence()); err == nil {
		t.Fatal("expected provider failure")
	}
}

func quote(value string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}
