package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"atonixcorp/security/crypto"
)

// Token contains identity + expiry + signature.
type Token struct {
	Identity  Identity
	Expires   time.Time
	Signature []byte
}

// TokenService issues and verifies tokens.
type TokenService struct {
	KM crypto.KeyManager
}

// NewTokenService creates a token service using your crypto keys.
func NewTokenService(km crypto.KeyManager) *TokenService {
	return &TokenService{KM: km}
}

// IssueToken creates a signed token for an identity.
func (ts *TokenService) IssueToken(id Identity, ttl time.Duration) (Token, error) {
	t := Token{
		Identity: id,
		Expires:  time.Now().UTC().Add(ttl),
	}

	// Serialize minimal token payload
	payload := []byte(id.ID + "|" + id.ProjectID + "|" + t.Expires.Format(time.RFC3339Nano))

	// Sign payload
	sig, err := crypto.Sign(id.ID, payload, ts.KM)
	if err != nil {
		return Token{}, err
	}

	t.Signature = sig
	return t, nil
}

// VerifyToken checks signature + expiry.
func (ts *TokenService) VerifyToken(t Token) error {
	if time.Now().UTC().After(t.Expires) {
		return errors.New("token expired")
	}

	payload := []byte(t.Identity.ID + "|" + t.Identity.ProjectID + "|" + t.Expires.Format(time.RFC3339Nano))

	return crypto.Verify(t.Identity.ID, payload, t.Signature, ts.KM)
}

type tokenParts struct {
	ID        string
	Project   string
	Expires   time.Time
	Signature []byte
	Roles     []string
}

func SplitToken(raw string) (*tokenParts, error) {
	if raw == "" {
		return nil, errors.New("invalid token format")
	}

	parts := strings.Split(raw, "|")
	if len(parts) != 4 {
		return nil, errors.New("invalid token format")
	}

	expires, err := time.Parse(time.RFC3339Nano, parts[2])
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, err
	}

	return &tokenParts{
		ID:        parts[0],
		Project:   parts[1],
		Expires:   expires,
		Signature: sig,
	}, nil
}

func ParseRawToken(raw string) (Token, error) {
	parts, err := SplitToken(raw)
	if err != nil {
		return Token{}, err
	}
	return Token{
		Identity: Identity{
			ID:        parts.ID,
			ProjectID: parts.Project,
			Roles:     parts.Roles,
		},
		Expires:   parts.Expires,
		Signature: parts.Signature,
	}, nil
}
