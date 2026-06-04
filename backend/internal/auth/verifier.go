// Package auth verifies the short-lived delegated JWTs core mints when it proxies a frontend call
// to this backend. The token is signed by core and verified against core's published JWKS; its
// subject is the acting user. This backend never sees a raw user token, only the delegated one.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Verifier validates a bearer token against core's JWKS, checking the issuer (core) and audience
// (this extension), and returns the authenticated user's subject.
type Verifier struct {
	keyfunc  jwt.Keyfunc
	issuer   string
	audience string
}

// NewVerifier fetches core's JWKS (auto-refreshing in the background) and binds the expected
// issuer and audience.
func NewVerifier(ctx context.Context, jwksURL, issuer, audience string) (*Verifier, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("load core JWKS from %s: %w", jwksURL, err)
	}
	return newVerifier(k.Keyfunc, issuer, audience), nil
}

// newVerifier builds a verifier from any jwt.Keyfunc, so tests can supply a static key.
func newVerifier(kf jwt.Keyfunc, issuer, audience string) *Verifier {
	return &Verifier{keyfunc: kf, issuer: issuer, audience: audience}
}

// Subject parses and verifies a raw bearer token and returns the `sub` claim. It rejects any token
// not signed with RS256, not issued by core, or not audienced to this extension, and any expired
// token.
func (v *Verifier) Subject(token string) (string, error) {
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(token, &claims, v.keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	); err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", errors.New("token has no subject")
	}
	return claims.Subject, nil
}
