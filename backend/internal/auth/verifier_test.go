package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://core.test"
	testAudience = "net.flytegration.calendar"
)

func mint(t *testing.T, key *rsa.PrivateKey, claims jwt.RegisteredClaims) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return raw
}

func validClaims(now time.Time) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   "user-1",
		Issuer:    testIssuer,
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
}

func TestSubjectAcceptsValidToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(staticKey(&key.PublicKey), testIssuer, testAudience)

	sub, err := v.Subject(mint(t, key, validClaims(time.Now())))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sub != "user-1" {
		t.Fatalf("subject = %q, want user-1", sub)
	}
}

func TestSubjectRejects(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(staticKey(&key.PublicKey), testIssuer, testAudience)
	now := time.Now()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "wrong audience",
			token: mint(t, key, jwt.RegisteredClaims{Subject: "u", Issuer: testIssuer, Audience: jwt.ClaimStrings{"net.flytegration.other"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}),
		},
		{
			name:  "wrong issuer",
			token: mint(t, key, jwt.RegisteredClaims{Subject: "u", Issuer: "https://evil.test", Audience: jwt.ClaimStrings{testAudience}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}),
		},
		{
			name:  "expired",
			token: mint(t, key, jwt.RegisteredClaims{Subject: "u", Issuer: testIssuer, Audience: jwt.ClaimStrings{testAudience}, ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute))}),
		},
		{
			name:  "no expiry",
			token: mint(t, key, jwt.RegisteredClaims{Subject: "u", Issuer: testIssuer, Audience: jwt.ClaimStrings{testAudience}}),
		},
		{
			name:  "signed by another key",
			token: mint(t, other, validClaims(now)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := v.Subject(tt.token); err == nil {
				t.Fatalf("%s token was accepted, want rejection", tt.name)
			}
		})
	}
}

func staticKey(pub *rsa.PublicKey) jwt.Keyfunc {
	return func(*jwt.Token) (any, error) { return pub, nil }
}
