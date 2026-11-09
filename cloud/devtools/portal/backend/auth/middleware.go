package auth

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		expected := os.Getenv("ATCLOUD_PORTAL_TOKEN")

		const prefix = "Bearer "
		if expected == "" || !strings.HasPrefix(token, prefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(token, prefix)), []byte(expected)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":"unauthorized","error":"authentication required"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
