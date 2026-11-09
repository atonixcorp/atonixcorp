package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type AuthContext struct {
	UserID   string
	TenantID string
	Roles    []string
}

type authContextKey struct{}

type AuthMiddleware struct {
	SigningKey []byte
}

func NewAuthMiddleware(signingKey []byte) *AuthMiddleware {
	return &AuthMiddleware{SigningKey: signingKey}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing or invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return m.SigningKey, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
			return
		}
		userID, userOK := claims["user_id"].(string)
		tenantID, tenantOK := claims["tenant_id"].(string)
		roles, rolesOK := toStringSlice(claims["roles"])
		if !userOK || userID == "" || !tenantOK || tenantID == "" || !rolesOK {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), authContextKey{}, &AuthContext{
			UserID:   userID,
			TenantID: tenantID,
			Roles:    roles,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthFromContext(ctx context.Context) (*AuthContext, bool) {
	auth, ok := ctx.Value(authContextKey{}).(*AuthContext)
	return auth, ok && auth != nil
}

func toStringSlice(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, len(arr))
	for i, x := range arr {
		role, ok := x.(string)
		if !ok || role == "" {
			return nil, false
		}
		out[i] = role
	}
	return out, true
}
