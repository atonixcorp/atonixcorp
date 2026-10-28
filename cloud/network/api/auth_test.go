package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type testProjectAuthorizer struct {
	allowed bool
	userID  string
	tenant  string
	project string
}

func (a *testProjectAuthorizer) AuthorizeProject(userID, tenantID, projectID string) (bool, error) {
	a.userID, a.tenant, a.project = userID, tenantID, projectID
	return a.allowed, nil
}

func TestNetworkingRoutesRequireAuthentication(t *testing.T) {
	authorizer := &testProjectAuthorizer{allowed: true}
	router := NewRouter(nil, nil, authorizer, "test-secret")
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/networking/tenants/12/projects/34/networks",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if authorizer.userID != "" {
		t.Fatal("project authorization ran before authentication")
	}
}

func TestNetworkingRoutesRequireProjectMembership(t *testing.T) {
	authorizer := &testProjectAuthorizer{}
	router := NewRouter(nil, nil, authorizer, "test-secret")
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/networking/tenants/12/projects/34/networks",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, 7, "test-secret"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if authorizer.userID != "7" || authorizer.tenant != "12" || authorizer.project != "34" {
		t.Fatalf("authorization scope = (%q, %q, %q), want (7, 12, 34)",
			authorizer.userID, authorizer.tenant, authorizer.project)
	}
}

func TestNetworkingRoutesRejectInvalidScope(t *testing.T) {
	authorizer := &testProjectAuthorizer{allowed: true}
	router := NewRouter(nil, nil, authorizer, "test-secret")
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/networking/tenants/0/projects/34/networks",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+signedToken(t, 7, "test-secret"))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if authorizer.userID != "" {
		t.Fatal("invalid scope reached project authorization")
	}
}

func signedToken(t *testing.T, userID uint, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	value, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return value
}
