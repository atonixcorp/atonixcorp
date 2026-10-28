package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"atonixcorp/cloud/database/core"

	"github.com/golang-jwt/jwt/v5"
)

type testStore struct {
	allowed bool
}

func (*testStore) CreateInstance(*core.DBInstance) error { return nil }
func (*testStore) UpdateInstance(*core.DBInstance) error { return nil }
func (*testStore) GetInstance(string) (*core.DBInstance, error) {
	return nil, errors.New("not found")
}
func (*testStore) ListInstances(string, string) ([]core.DBInstance, error) {
	return []core.DBInstance{}, nil
}
func (*testStore) ListInstancesByStatus(string, int) ([]core.DBInstance, error) {
	return []core.DBInstance{}, nil
}
func (s *testStore) CanAccessProject(uint, string, string) (bool, error) {
	return s.allowed, nil
}
func (*testStore) CreateInstanceOperation(
	*core.DBInstance, *core.DatabaseOperation, string, string,
) (bool, error) {
	return false, nil
}
func (*testStore) GetOperation(string) (*core.DatabaseOperation, error) {
	return nil, errors.New("not found")
}
func (*testStore) GetOperationForDatabase(string) (*core.DatabaseOperation, error) {
	return nil, errors.New("not found")
}
func (*testStore) UpdateOperation(*core.DatabaseOperation) error { return nil }

func TestRouterRequiresAuthenticationAndProjectMembership(t *testing.T) {
	secret := "test-only-signing-key"
	store := &testStore{}
	server := NewAPIServer(core.NewController(store, nil), store, secret)
	path := "/api/v1/tenants/1/projects/2/databases"

	unauthenticated := httptest.NewRecorder()
	server.Router().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, path, nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated request to be rejected, got %d", unauthenticated.Code)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &apiClaims{UserID: 7})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)
	forbidden := httptest.NewRecorder()
	server.Router().ServeHTTP(forbidden, request)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected non-member request to be forbidden, got %d", forbidden.Code)
	}

	store.allowed = true
	request = httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)
	authorized := httptest.NewRecorder()
	server.Router().ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK {
		t.Fatalf("expected project member request to succeed, got %d", authorized.Code)
	}
}
