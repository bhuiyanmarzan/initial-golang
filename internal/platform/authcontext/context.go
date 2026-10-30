// Package authcontext holds only verified identity data. Only authentication
// adapters may call WithPrincipal after token/session verification.
package authcontext

import (
	"context"
	"errors"
	"net/http"
)

type key struct{}
type Principal struct{ UserID string }

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, key{}, p)
}
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(key{}).(Principal)
	return p, ok && p.UserID != ""
}
func UserKey(r *http.Request) (string, error) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		return "", errors.New("verified identity missing")
	}
	return p.UserID, nil
}
