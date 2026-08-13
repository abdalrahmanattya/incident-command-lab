package server

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/abdalrahmanattya/incident-command-lab/internal/analysis"
	"github.com/abdalrahmanattya/incident-command-lab/internal/app"
	"github.com/abdalrahmanattya/incident-command-lab/internal/messaging"
	"github.com/abdalrahmanattya/incident-command-lab/internal/observability"
	"github.com/abdalrahmanattya/incident-command-lab/internal/store"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	app  *app.App
	log  *slog.Logger
	port string
}

func New(a *app.App) *Server {
	return &Server{app: a, log: slog.New(slog.NewJSONHandler(os.Stdout, nil)), port: env("PORT", "8080")}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
	mux.HandleFunc("GET /metrics", s.metrics)
	mux.HandleFunc("GET /v1/catalogue", s.catalogue)
	mux.HandleFunc("POST /v1/reservations", s.reserve)
	mux.HandleFunc("GET /v1/reservations/", s.reservation)
	mux.HandleFunc("POST /v1/reservations/", s.reservationAction)
	ops := http.NewServeMux()
	ops.HandleFunc("POST /ops/faults", s.fault)
	ops.HandleFunc("GET /ops/state", s.state)
	ops.HandleFunc("POST /ops/incidents", s.incident)
	ops.HandleFunc("GET /ops/incidents", s.incidents)
	ops.HandleFunc("GET /ops/incidents/", s.incidentAction)
	ops.HandleFunc("POST /ops/incidents/", s.incidentAction)
	mux.Handle("/ops/", s.operatorOnly(ops))
	return observability.Middleware(mux, s.log)
}

// operatorOnly keeps local operation credential-free while requiring a fully
// validated Entra bearer token carrying the configured operator group in cloud
// mode. It fails closed when any issuer, audience, expiry, signature, or group
// claim does not match the configured values.
func (s *Server) operatorOnly(next http.Handler) http.Handler {
	if strings.ToLower(os.Getenv("AUTH_MODE")) != "entra" {
		return next
	}
	group := os.Getenv("OPERATOR_GROUP_ID")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if group == "" {
			write(w, http.StatusForbidden, map[string]string{"error": "operator group is not configured"})
			return
		}
		claims, err := validateEntraToken(r)
		if err != nil {
			write(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		if !contains(claims.Groups, group) {
			write(w, http.StatusForbidden, map[string]string{"error": "operator group required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type entraClaims struct {
	Groups []string `json:"groups"`
}
type entraHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}
type jwks struct {
	Keys []jwk `json:"keys"`
}
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func validateEntraToken(r *http.Request) (entraClaims, error) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return entraClaims{}, fmt.Errorf("operator bearer token required")
	}
	segments := strings.Split(parts[1], ".")
	if len(segments) != 3 {
		return entraClaims{}, fmt.Errorf("invalid bearer token")
	}
	var header entraHeader
	var claims struct {
		Iss    string          `json:"iss"`
		Aud    json.RawMessage `json:"aud"`
		Exp    float64         `json:"exp"`
		Nbf    float64         `json:"nbf"`
		Groups []string        `json:"groups"`
	}
	decode := func(segment string, target any) error {
		b, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, target)
	}
	if err := decode(segments[0], &header); err != nil || header.Alg != "RS256" || header.Kid == "" {
		return entraClaims{}, fmt.Errorf("invalid JWT header")
	}
	if err := decode(segments[1], &claims); err != nil {
		return entraClaims{}, fmt.Errorf("invalid JWT claims")
	}
	issuer, audience, jwksURL := os.Getenv("AUTH_ISSUER"), os.Getenv("AUTH_AUDIENCE"), os.Getenv("AUTH_JWKS_URL")
	if issuer == "" || audience == "" || jwksURL == "" {
		return entraClaims{}, fmt.Errorf("Entra validation is not configured")
	}
	if claims.Iss != issuer || claims.Exp <= float64(time.Now().Unix()) || (claims.Nbf != 0 && claims.Nbf > float64(time.Now().Unix())) {
		return entraClaims{}, fmt.Errorf("invalid issuer or token lifetime")
	}
	var audString string
	var audList []string
	if json.Unmarshal(claims.Aud, &audString) != nil {
		_ = json.Unmarshal(claims.Aud, &audList)
	}
	if audString != audience && !contains(audList, audience) {
		return entraClaims{}, fmt.Errorf("invalid audience")
	}
	keyset, err := fetchJWKS(jwksURL)
	if err != nil {
		return entraClaims{}, err
	}
	var key jwk
	for _, candidate := range keyset.Keys {
		if candidate.Kid == header.Kid {
			key = candidate
			break
		}
	}
	if key.Kty != "RSA" || key.N == "" || key.E == "" {
		return entraClaims{}, fmt.Errorf("signing key not found")
	}
	n, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return entraClaims{}, fmt.Errorf("invalid signing modulus")
	}
	e, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return entraClaims{}, fmt.Errorf("invalid signing exponent")
	}
	var exponent int
	for _, b := range e {
		exponent = exponent*256 + int(b)
	}
	if exponent == 0 {
		return entraClaims{}, fmt.Errorf("invalid signing exponent")
	}
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
	sig, err := base64.RawURLEncoding.DecodeString(segments[2])
	if err != nil {
		return entraClaims{}, fmt.Errorf("invalid signature")
	}
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return entraClaims{}, fmt.Errorf("invalid signature")
	}
	return entraClaims{Groups: claims.Groups}, nil
}
func fetchJWKS(endpoint string) (jwks, error) {
	response, err := (&http.Client{Timeout: 3 * time.Second}).Get(endpoint)
	if err != nil {
		return jwks{}, fmt.Errorf("JWKS unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return jwks{}, fmt.Errorf("JWKS unavailable")
	}
	var keys jwks
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&keys); err != nil {
		return jwks{}, fmt.Errorf("invalid JWKS")
	}
	return keys, nil
}
func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
func (s *Server) ListenAndServe() error {
	s.log.Info("incident command lab listening", "port", s.port)
	return http.ListenAndServe(":"+s.port, s.Handler())
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("request body required")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]string{"status": "ok", "service": env("SERVICE_NAME", "gateway")})
}
func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(observability.MetricsText()))
}
func (s *Server) catalogue(w http.ResponseWriter, _ *http.Request) {
	write(w, 200, map[string]any{"products": s.app.ListProducts()})
}

type reserveRequest struct {
	CustomerID string `json:"customer_id"`
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
}

func (s *Server) reserve(w http.ResponseWriter, r *http.Request) {
	var in reserveRequest
	if err := decode(r, &in); err != nil {
		write(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if in.CustomerID == "" {
		write(w, 400, map[string]string{"error": "customer_id is required"})
		return
	}
	res, err := s.app.Reserve(r.Context(), key, in.CustomerID, in.ProductID, in.Quantity)
	if err != nil {
		write(w, 422, map[string]string{"error": err.Error()})
		return
	}
	write(w, 201, res)
}
func (s *Server) reservation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/reservations/")
	if id == "" {
		write(w, 404, map[string]string{"error": "not found"})
		return
	}
	res, ok := s.app.GetReservation(id)
	if !ok {
		write(w, 404, map[string]string{"error": "reservation not found"})
		return
	}
	write(w, 200, res)
}
func (s *Server) reservationAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		write(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/reservations/")
	if !strings.HasSuffix(id, "/cancel") {
		write(w, 404, map[string]string{"error": "not found"})
		return
	}
	id = strings.TrimSuffix(id, "/cancel")
	res, err := s.app.Cancel(id)
	if err != nil {
		write(w, 404, map[string]string{"error": err.Error()})
		return
	}
	write(w, 200, res)
}

type faultRequest struct {
	Fault   string `json:"fault"`
	Enabled bool   `json:"enabled"`
}

func (s *Server) fault(w http.ResponseWriter, r *http.Request) {
	var in faultRequest
	if err := decode(r, &in); err != nil {
		write(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	valid := map[app.Fault]bool{app.FaultLatency: true, app.FaultDependency: true, app.FaultBacklog: true, app.FaultDuplicate: true, app.FaultDB: true, app.FaultBadRelease: true}
	f := app.Fault(in.Fault)
	if !valid[f] {
		write(w, 400, map[string]string{"error": "unknown fault"})
		return
	}
	s.app.SetFault(f, in.Enabled)
	write(w, 200, map[string]any{"fault": f, "enabled": in.Enabled, "active": s.app.Faults()})
}
func (s *Server) state(w http.ResponseWriter, _ *http.Request) { write(w, 200, s.app.Snapshot()) }

type incidentRequest struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
}

func (s *Server) incident(w http.ResponseWriter, r *http.Request) {
	var in incidentRequest
	if err := decode(r, &in); err != nil || !app.ValidateTitle(in.Title) {
		write(w, 400, map[string]string{"error": "title must be 3-120 characters"})
		return
	}
	if in.Severity == "" {
		in.Severity = "SEV3"
	}
	write(w, 201, s.app.CreateIncident(in.Title, in.Severity))
}
func (s *Server) incidents(w http.ResponseWriter, _ *http.Request) {
	write(w, 200, map[string]any{"incidents": s.app.ListIncidents()})
}
func (s *Server) incidentAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ops/incidents/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		write(w, 404, map[string]string{"error": "not found"})
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		if i, ok := s.app.Incident(id); ok {
			write(w, 200, i)
		} else {
			write(w, 404, map[string]string{"error": "incident not found"})
		}
		return
	}
	if parts[1] == "evidence" {
		e, err := s.app.Evidence(id)
		if err != nil {
			write(w, 404, map[string]string{"error": err.Error()})
			return
		}
		write(w, 200, e)
		return
	}
	if parts[1] == "analyze" {
		e, err := s.app.Evidence(id)
		if err != nil {
			write(w, 404, map[string]string{"error": err.Error()})
			return
		}
		report, err := analysis.Select().Analyze(r.Context(), e)
		if err != nil {
			report, _ = analysis.Deterministic{}.Analyze(r.Context(), e)
			report.Provider = "deterministic-fallback"
		}
		write(w, 200, report)
		return
	}
	write(w, 404, map[string]string{"error": "not found"})
}

// Run starts a named service. Service binaries share the same contract so they can
// be composed independently in Kubernetes while the gateway exposes the API.
func Run(name string) error {
	os.Setenv("SERVICE_NAME", name)
	telCtx, telCancel := context.WithTimeout(context.Background(), 5*time.Second)
	telemetry, telErr := observability.Setup(telCtx)
	telCancel()
	if telErr != nil {
		return fmt.Errorf("telemetry: %w", telErr)
	}
	defer telemetry.Shutdown(context.Background())
	a := app.New()
	var repo *store.Postgres
	if url := os.Getenv("DATABASE_URL"); url != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		var err error
		for {
			repo, err = store.Open(ctx, url)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return fmt.Errorf("database: %w", err)
			}
			time.Sleep(time.Second)
		}
		defer repo.Close()
		a = app.NewWithStore(repo)
	}
	if nurl := os.Getenv("NATS_URL"); nurl != "" {
		bus, err := messaging.Connect(nurl)
		if err != nil {
			return fmt.Errorf("nats: %w", err)
		}
		defer bus.Close()
		_ = bus.EnsureConsumer()
		if repo != nil {
			go func() {
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()
				for range ticker.C {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_ = repo.PublishPending(ctx, 50, bus.Publish)
					cancel()
				}
			}()
		}
	}
	return New(a).ListenAndServe()
}
