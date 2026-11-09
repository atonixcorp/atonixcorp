package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAuthRequiresConfiguredMatchingToken(t *testing.T) {
	t.Setenv("ATCLOUD_PORTAL_TOKEN", "test-portal-token")
	handler := WithAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, testCase := range []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer other-token", wantStatus: http.StatusUnauthorized},
		{name: "valid", header: "Bearer test-portal-token", wantStatus: http.StatusNoContent},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", testCase.header)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, testCase.wantStatus)
			}
		})
	}
}
