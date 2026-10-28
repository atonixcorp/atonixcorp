package security

import (
	"context"
	"errors"
	"testing"

	"atonixcorp/api/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newScopedSecurityService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open security test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.TenantMembership{},
		&models.Project{},
		&models.ProjectMembership{},
		&models.SecurityPolicy{},
		&models.SecurityAuditEvent{},
	); err != nil {
		t.Fatalf("migrate security test database: %v", err)
	}
	if err := db.Create(&models.Tenant{ID: 1, Name: "tenant one", Slug: "tenant-one"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Project{ID: 1, TenantID: 1, Name: "project one", Slug: "project-one"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 11, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 11, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	return NewService(db), db
}

func validDraftRequest() CreatePolicyRequest {
	return CreatePolicyRequest{
		Name: "Require signed workload images",
		Type: "workload",
		Rules: []PolicyRule{{
			Field: "workload.image_signed", Operator: "equals", Value: "true", Effect: "require_review",
		}},
	}
}

func testRequestContext() RequestContext {
	return RequestContext{RequestID: "req-123", CorrelationID: "corr-123"}
}

func TestCreateDraftScopesRecordsAndAuditsInTransaction(t *testing.T) {
	service, db := newScopedSecurityService(t)
	scope := Scope{UserID: 11, TenantID: 1, ProjectID: 1}
	policy, err := service.CreateDraft(
		context.Background(), scope, testRequestContext(), "policy-create-001", validDraftRequest(),
	)
	if err != nil {
		t.Fatalf("create policy draft: %v", err)
	}
	if policy.ID == 0 || policy.State != "Draft" || policy.Version != 1 {
		t.Fatalf("unexpected policy response: %+v", policy)
	}

	var stored models.SecurityPolicy
	if err := db.First(&stored, policy.ID).Error; err != nil {
		t.Fatalf("load stored policy: %v", err)
	}
	if stored.TenantID != scope.TenantID || stored.ProjectID != scope.ProjectID || stored.OwnerID != scope.UserID {
		t.Fatalf("policy escaped authorized scope: %+v", stored)
	}
	var audit models.SecurityAuditEvent
	if err := db.Where("action = ?", "security.policy.create").First(&audit).Error; err != nil {
		t.Fatalf("load create audit: %v", err)
	}
	if audit.ActorID != scope.UserID || audit.Result != "success" || audit.RequestID != "req-123" {
		t.Fatalf("unexpected audit record: %+v", audit)
	}
}

func TestCreateDraftDeniesCrossTenantAndProjectSubstitution(t *testing.T) {
	service, db := newScopedSecurityService(t)
	tests := []Scope{
		{UserID: 11, TenantID: 2, ProjectID: 1},
		{UserID: 11, TenantID: 1, ProjectID: 2},
		{UserID: 12, TenantID: 1, ProjectID: 1},
	}
	for _, scope := range tests {
		_, err := service.CreateDraft(
			context.Background(), scope, testRequestContext(), "cross-scope-key-001", validDraftRequest(),
		)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected cross-scope create to be denied for %+v, got %v", scope, err)
		}
	}
	var count int64
	if err := db.Model(&models.SecurityPolicy{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unauthorized attempts created %d policies", count)
	}
	if err := db.Model(&models.SecurityAuditEvent{}).Where("result = ?", "denied").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(tests)) {
		t.Fatalf("expected all denied attempts audited, got %d", count)
	}
}

func TestCreateDraftEnforcesMembershipRolesAndIdempotency(t *testing.T) {
	service, db := newScopedSecurityService(t)
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 12, Role: "viewer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 12, Role: "deployer"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateDraft(
		context.Background(), Scope{UserID: 12, TenantID: 1, ProjectID: 1},
		testRequestContext(), "viewer-create-key", validDraftRequest(),
	); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer should not create a policy draft, got %v", err)
	}

	scope := Scope{UserID: 11, TenantID: 1, ProjectID: 1}
	request := validDraftRequest()
	first, err := service.CreateDraft(context.Background(), scope, testRequestContext(), "same-request-key", request)
	if err != nil {
		t.Fatalf("create initial draft: %v", err)
	}
	replayed, err := service.CreateDraft(context.Background(), scope, testRequestContext(), "same-request-key", request)
	if err != nil {
		t.Fatalf("replay create request: %v", err)
	}
	if replayed.ID != first.ID {
		t.Fatalf("idempotent replay returned a different policy: %d != %d", replayed.ID, first.ID)
	}
	changed := request
	changed.Name = "Different policy under same key"
	if _, err := service.CreateDraft(context.Background(), scope, testRequestContext(), "same-request-key", changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("reused key with a different request should conflict, got %v", err)
	}
}

func TestSimulationIsAuditedButUnavailableWithoutEngine(t *testing.T) {
	service, db := newScopedSecurityService(t)
	scope := Scope{UserID: 11, TenantID: 1, ProjectID: 1}
	policy, err := service.CreateDraft(
		context.Background(), scope, testRequestContext(), "simulation-create-key", validDraftRequest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = service.SimulatePolicy(context.Background(), scope, testRequestContext(), policy.ID)
	if !errors.Is(err, ErrSimulationUnavailable) {
		t.Fatalf("simulation should fail closed without an engine, got %v", err)
	}
	var audit models.SecurityAuditEvent
	if err := db.Where("action = ?", "security.policy.simulate").First(&audit).Error; err != nil {
		t.Fatalf("simulation attempt was not audited: %v", err)
	}
	if audit.Result != "unavailable" || audit.Reason != "SIMULATION_ENGINE_NOT_CONFIGURED" {
		t.Fatalf("unexpected simulation audit outcome: %+v", audit)
	}
}

func TestListAuditRequiresElevatedRoleAndIsScopeBound(t *testing.T) {
	service, db := newScopedSecurityService(t)
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 12, Role: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 12, Role: "viewer"}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := service.ListAudit(
		context.Background(), Scope{UserID: 12, TenantID: 1, ProjectID: 1},
		testRequestContext(), 20, 0, 24,
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer should not read audit records, got %v", err)
	}

	scope := Scope{UserID: 11, TenantID: 1, ProjectID: 1}
	if _, err := service.CreateDraft(
		context.Background(), scope, testRequestContext(), "audit-create-key", validDraftRequest(),
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SecurityAuditEvent{
		TenantID: 1, ProjectID: 2, ActorID: 11, Action: "security.other-project",
		ResourceType: "policy", ResourceID: "foreign-record", Result: "success",
		RequestID: "req-foreign", CorrelationID: "corr-foreign",
	}).Error; err != nil {
		t.Fatal(err)
	}
	page, err := service.ListAudit(context.Background(), scope, testRequestContext(), 20, 0, 24)
	if err != nil {
		t.Fatalf("list scoped audit: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected only the two events in the authorized project, got %d", len(page.Items))
	}
	for _, event := range page.Items {
		if event.ResourceID == "foreign-record" {
			t.Fatal("audit query returned an event from another project")
		}
	}
}
