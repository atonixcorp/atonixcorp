package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareValidatesAlgorithmAndTenantClaims(t *testing.T) {
	key := []byte("test-signing-key-with-at-least-32-characters")
	handler := NewAuthMiddleware(key).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, ok := AuthFromContext(r.Context())
		if !ok || auth.UserID != "user-1" || auth.TenantID != "tenant-1" || len(auth.Roles) != 1 {
			t.Errorf("unexpected authentication context: %+v", auth)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, testCase := range []struct {
		name       string
		method     *jwt.SigningMethodHMAC
		claims     jwt.MapClaims
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{
			name:       "unexpected algorithm",
			method:     jwt.SigningMethodHS512,
			claims:     validClaims(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing tenant",
			method:     jwt.SigningMethodHS256,
			claims:     jwt.MapClaims{"user_id": "user-1", "roles": []string{"hpc.submit"}, "exp": time.Now().Add(time.Minute).Unix()},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid",
			method:     jwt.SigningMethodHS256,
			claims:     validClaims(),
			wantStatus: http.StatusNoContent,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if testCase.method != nil {
				token, err := jwt.NewWithClaims(testCase.method, testCase.claims).SignedString(key)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, testCase.wantStatus)
			}
		})
	}
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"user_id":   "user-1",
		"tenant_id": "tenant-1",
		"roles":     []string{"hpc.submit"},
		"exp":       time.Now().Add(time.Minute).Unix(),
	}
}
