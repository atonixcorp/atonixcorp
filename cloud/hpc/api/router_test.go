package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"atonixcorp/cloud/hpc/metrics"
)

func TestHealthEndpointsArePublicAndProtectedRoutesRequireAuthentication(t *testing.T) {
	handler := NewRouterWithChecks(nil, nil, nil, nil, metrics.NewExporter(), []byte("test-key"),
		nil, RuntimeChecks{
			PingDatabase: func(context.Context) error { return nil },
		})

	for _, path := range []string{"/health", "/ready"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusOK)
		}
	}

	protected := httptest.NewRecorder()
	handler.ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/hpc/nodes", nil))
	if protected.Code != http.StatusUnauthorized {
		t.Fatalf("protected status = %d, want %d", protected.Code, http.StatusUnauthorized)
	}
}

func TestReadyReportsDatabaseAndStorageFailures(t *testing.T) {
	handler := NewRouterWithChecks(nil, nil, nil, nil, metrics.NewExporter(), []byte("test-key"),
		nil, RuntimeChecks{
			PingDatabase: func(context.Context) error { return errors.New("database down") },
		})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("database failure status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
