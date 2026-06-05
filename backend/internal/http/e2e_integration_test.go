//go:build integration

// End-to-end across the backend's full request path: a delegated token shaped exactly like the
// one the platform mints (RS256, iss/aud/exp/sub) is verified against a live JWKS, then drives the
// events API against a real Postgres. This proves the seam between the platform's token minting and
// this backend's verification, plus the data round-trip.
package http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/auth"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	calhttp "github.com/mucha17/flyspace-extensions-calendar/backend/internal/http"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/store"
)

const (
	e2eIssuer   = "https://core.test"
	e2eAudience = "net.flytegration.calendar"
	e2eKid      = "core-1"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("calendar"), postgres.WithUsername("calendar"), postgres.WithPassword("dev"),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		)),
	)
	if err != nil {
		panic(err)
	}
	url, _ := ctr.ConnectionString(ctx, "sslmode=disable")
	if pool, err = pgxpool.New(ctx, url); err != nil {
		panic(err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		panic(err)
	}

	code := m.Run()

	pool.Close()
	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

// serveJWKS publishes the RSA public key as a JWKS, exactly as the platform exposes its signing key.
func serveJWKS(t *testing.T, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	jwks := map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": e2eKid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}}
	body, _ := json.Marshal(jwks)
	return httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

func mintDelegated(t *testing.T, key *rsa.PrivateKey, aud, sub string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   sub,
		Issuer:    e2eIssuer,
		Audience:  jwt.ClaimStrings{aud},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	tok.Header["kid"] = e2eKid
	raw, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign delegated token: %v", err)
	}
	return raw
}

func TestDelegatedTokenRoundTrip(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwks := serveJWKS(t, &key.PublicKey)
	defer jwks.Close()

	verifier, err := auth.NewVerifier(context.Background(), jwks.URL, e2eIssuer, e2eAudience)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	srv := calhttp.NewRouter(calhttp.Deps{
		Events:   calendar.NewService(store.NewPgEventStore(pool)),
		Verifier: verifier,
	})

	token := mintDelegated(t, key, e2eAudience, "user-e2e")

	// Create an event with the delegated token.
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req("POST", "/events",
		`{"title":"Launch","start":"2026-06-10T09:00:00Z","end":"2026-06-10T10:00:00Z"}`, token))
	if rec.Code != stdhttp.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body.String())
	}

	// Read it back for the same user.
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req("GET", "/events?from=2026-06-01T00:00:00Z&to=2026-06-30T00:00:00Z", "", token))
	if rec.Code != stdhttp.StatusOK || !strings.Contains(rec.Body.String(), "Launch") {
		t.Fatalf("list: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestWrongAudienceRejectedEndToEnd(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwks := serveJWKS(t, &key.PublicKey)
	defer jwks.Close()

	verifier, err := auth.NewVerifier(context.Background(), jwks.URL, e2eIssuer, e2eAudience)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	srv := calhttp.NewRouter(calhttp.Deps{
		Events:   calendar.NewService(store.NewPgEventStore(pool)),
		Verifier: verifier,
	})

	// A token audienced to a different extension must be rejected by the verifier.
	token := mintDelegated(t, key, "net.flytegration.other", "user-e2e")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req("GET", "/events?from=2026-06-01T00:00:00Z&to=2026-06-30T00:00:00Z", "", token))
	if rec.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("wrong audience: status %d, want 401", rec.Code)
	}
}
