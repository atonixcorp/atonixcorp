package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthIsPublicAndResourceRoutesRequireAuthentication(t *testing.T) {
	handler := NewRouter("test-api-token")

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}

	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/projects/list", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthenticated.Code, http.StatusUnauthorized)
	}

	authenticatedRequest := httptest.NewRequest(http.MethodGet, "/projects/list", nil)
	authenticatedRequest.Header.Set("Authorization", "Bearer test-api-token")
	authenticated := httptest.NewRecorder()
	handler.ServeHTTP(authenticated, authenticatedRequest)
	if authenticated.Code != http.StatusServiceUnavailable || !strings.Contains(authenticated.Body.String(), "provider_unconfigured") {
		t.Fatalf("unconfigured provider response = %d %q", authenticated.Code, authenticated.Body.String())
	}
}
