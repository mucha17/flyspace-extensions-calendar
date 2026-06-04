package http

import (
	"context"
	stdhttp "net/http"
	"strings"
)

type ctxKey int

const userKey ctxKey = 0

// Verifier validates the delegated bearer token and returns the acting user's subject.
type Verifier interface {
	Subject(token string) (string, error)
}

// authenticate verifies the delegated token core attached and places the user subject in the
// context. Requests without a valid token get 401 problem+json — this backend trusts only core's
// signature, never a client-set identity.
func authenticate(v Verifier) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				problem(w, stdhttp.StatusUnauthorized, "auth.required", "missing delegated token")
				return
			}
			sub, err := v.Subject(token)
			if err != nil {
				problem(w, stdhttp.StatusUnauthorized, "auth.invalid", "delegated token rejected")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, sub)))
		})
	}
}

func userFrom(ctx context.Context) string {
	sub, _ := ctx.Value(userKey).(string)
	return sub
}
