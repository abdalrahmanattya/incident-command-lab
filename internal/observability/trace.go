package observability

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go.opentelemetry.io/otel"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
)

// Context carries the W3C trace id used to correlate HTTP, events and logs.
type Context struct{ TraceID, SpanID string }

func New() Context {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	s := make([]byte, 8)
	_, _ = rand.Read(s)
	return Context{TraceID: hex.EncodeToString(b), SpanID: hex.EncodeToString(s)}
}

func FromRequest(r *http.Request) Context {
	p := strings.TrimSpace(r.Header.Get("traceparent"))
	parts := strings.Split(p, "-")
	if len(parts) == 4 && len(parts[1]) == 32 && len(parts[2]) == 16 {
		return Context{TraceID: parts[1], SpanID: parts[2]}
	}
	return New()
}

func (c Context) Traceparent() string { return "00-" + c.TraceID + "-" + c.SpanID + "-01" }
func (c Context) Logger(l *slog.Logger) *slog.Logger {
	return l.With("trace_id", c.TraceID, "span_id", c.SpanID)
}

func Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests, _ := otel.Meter("incidentlab").Int64Counter("incidentlab_http_requests_total")
		requests.Add(r.Context(), 1)
		requestCount.Add(1)
		ctx := FromRequest(r)
		otelCtx, span := StartSpan(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		r = r.WithContext(otelCtx)
		w.Header().Set("traceparent", ctx.Traceparent())
		w.Header().Set("x-trace-id", ctx.TraceID)
		ctx.Logger(logger).Info("http.request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

var requestCount atomic.Uint64

func MetricsText() string {
	return "# HELP incidentlab_up Process health\n# TYPE incidentlab_up gauge\nincidentlab_up 1\n# HELP incidentlab_http_requests_total HTTP requests\n# TYPE incidentlab_http_requests_total counter\nincidentlab_http_requests_total " + fmt.Sprint(requestCount.Load()) + "\n"
}
