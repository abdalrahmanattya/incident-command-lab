package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/abdalrahmanattya/incident-command-lab/internal/app"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEntraOperatorValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "test-key", "alg": "RS256", "n": n, "e": e}}})
	}))
	defer jwksServer.Close()
	t.Setenv("AUTH_MODE", "entra")
	t.Setenv("OPERATOR_GROUP_ID", "ops")
	t.Setenv("AUTH_ISSUER", "https://issuer.example/v2.0")
	t.Setenv("AUTH_AUDIENCE", "api://incidentlab")
	t.Setenv("AUTH_JWKS_URL", jwksServer.URL)
	valid := makeToken(t, key, "https://issuer.example/v2.0", "api://incidentlab", time.Now().Add(5*time.Minute), []string{"ops"})
	for _, tc := range []struct {
		name, token string
		status      int
	}{
		{"missing", "", http.StatusUnauthorized}, {"forged", valid + "x", http.StatusUnauthorized},
		{"wrong issuer", makeToken(t, key, "https://wrong", "api://incidentlab", time.Now().Add(time.Minute), []string{"ops"}), http.StatusUnauthorized},
		{"wrong audience", makeToken(t, key, "https://issuer.example/v2.0", "api://wrong", time.Now().Add(time.Minute), []string{"ops"}), http.StatusUnauthorized},
		{"expired", makeToken(t, key, "https://issuer.example/v2.0", "api://incidentlab", time.Now().Add(-time.Minute), []string{"ops"}), http.StatusUnauthorized},
		{"wrong group", makeToken(t, key, "https://issuer.example/v2.0", "api://incidentlab", time.Now().Add(time.Minute), []string{"readers"}), http.StatusForbidden},
		{"valid", valid, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(appForAuthTest()).Handler()
			req := httptest.NewRequest(http.MethodGet, "/ops/state", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("got %d want %d body=%s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}
func appForAuthTest() *app.App { return app.New() }
func makeToken(t *testing.T, key *rsa.PrivateKey, issuer, audience string, expiry time.Time, groups []string) string {
	t.Helper()
	enc := base64.RawURLEncoding.EncodeToString
	header := enc([]byte(`{"alg":"RS256","kid":"test-key","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(map[string]any{"iss": issuer, "aud": audience, "exp": expiry.Unix(), "groups": groups})
	payload := enc(payloadBytes)
	input := header + "." + payload
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + enc(sig)
}
