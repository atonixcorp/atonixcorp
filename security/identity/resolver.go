package identity

import (
	"atonixcorp/security/auth"
)

// Resolver resolves identity from a token or raw input.
type Resolver struct {
	TokenService *auth.TokenService
}

// NewResolver creates a new identity resolver.
func NewResolver(ts *auth.TokenService) *Resolver {
	return &Resolver{TokenService: ts}
}

// ResolveIdentity resolves identity from a raw token string.
func (r *Resolver) ResolveIdentity(token string) (auth.Identity, error) {
	t, err := auth.ParseRawToken(token)
	if err != nil {
		return auth.Identity{}, err
	}

	if err := r.TokenService.VerifyToken(t); err != nil {
		return auth.Identity{}, err
	}

	return t.Identity, nil
}
