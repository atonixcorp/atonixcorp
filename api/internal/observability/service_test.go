package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"atonixcorp/api/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open observability test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Tenant{}, &models.TenantMembership{}, &models.Project{}, &models.ProjectMembership{},
		&models.StorageVolume{}, &models.ContainerDeployment{}, &models.ObservabilityResource{},
		&models.StorageAuditEvent{}, &models.ContainerDeploymentEvent{}, &models.SecurityAuditEvent{},
	); err != nil {
		t.Fatalf("migrate observability test database: %v", err)
	}
	if err := db.Create(&models.Tenant{ID: 1, Name: "tenant", Slug: "tenant"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Project{ID: 1, TenantID: 1, Name: "project", Slug: "project"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 7, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 7, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	return NewService(db), db
}

func TestResourcesAreDiscoveredAndSynchronizedFromSourceServices(t *testing.T) {
	service, db := newTestService(t)
	now := time.Now().UTC()
	if err := db.Create(&models.StorageVolume{
		ID: "0123456789abcdef0123456789abcdef", TenantID: 1, ProjectID: 1,
		OwnerID: 7, Name: "training-data", Region: "jhb-1", StorageClass: "standard",
		SizeGB: 20, State: "AVAILABLE", Driver: "cinder", IdempotencyKey: "create-volume",
		RequestHash: "hash", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContainerDeployment{
		ID: "abcdef0123456789abcdef0123456789", TenantID: 1, ProjectID: 1, OwnerID: 7,
		Name: "training-worker", State: "RUNNING", DesiredReplicas: 2,
		SpecJSON: []byte(`{}`), IdempotencyKey: "create-workload",
		RequestHash: "hash", CorrelationID: "corr", ProvisioningJobID: "job",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	resources, err := service.Resources(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil {
		t.Fatalf("discover resources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected storage and container resources to be discovered, got %d", len(resources))
	}
	for _, resource := range resources {
		if resource.MonitoringStatus != "not_configured" {
			t.Fatalf("resource must not imply telemetry when no adapter is connected: %+v", resource)
		}
	}
	if _, err := service.Resources(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1}); err != nil {
		t.Fatalf("repeat resource reconciliation: %v", err)
	}
	var count int64
	if err := db.Model(&models.ObservabilityResource{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("reconciliation should upsert, not duplicate records; got %d", count)
	}

	overview, err := service.Overview(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if overview.Health != "unknown" || overview.UnknownHealth != 2 || overview.TelemetryState != "unavailable" {
		t.Fatalf("overview should report unknown health without telemetry, got %+v", overview)
	}
	if overview.ResourceGroups[6].Count != 1 || overview.ResourceGroups[6].Kind != "storage.volume" {
		t.Fatalf("storage resource should be reflected in the type-specific inventory: %+v", overview.ResourceGroups[6])
	}
}

func TestResourcesAreProjectScopedAndRemovedResourcesAreHidden(t *testing.T) {
	service, db := newTestService(t)
	if _, err := service.Resources(context.Background(), Scope{UserID: 8, TenantID: 1, ProjectID: 1}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-member must not access the project inventory: %v", err)
	}
	if err := db.Create(&models.ObservabilityResource{
		TenantID: 1, ProjectID: 1, ResourceType: "storage.volume",
		ResourceID: "deleted-volume-id", Provider: "cinder", Name: "old-volume",
		ResourceStatus: "AVAILABLE", MonitoringStatus: "not_configured",
		FirstSeenAt: time.Now(), LastSeenAt: time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	resources, err := service.Resources(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 0 {
		t.Fatalf("resource absent from authoritative source should be marked removed and hidden, got %+v", resources)
	}
}

func TestMetricsAndLogsDoNotFabricateTelemetry(t *testing.T) {
	service, db := newTestService(t)
	if err := db.Create(&models.StorageVolume{
		ID: "0123456789abcdef0123456789abcdef", TenantID: 1, ProjectID: 1,
		OwnerID: 7, Name: "volume", State: "AVAILABLE", Driver: "cinder",
		IdempotencyKey: "key", RequestHash: "hash",
	}).Error; err != nil {
		t.Fatal(err)
	}
	scope := Scope{UserID: 7, TenantID: 1, ProjectID: 1}
	metrics, err := service.ResourceMetrics(context.Background(), scope, "storage.volume", "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.State != "unavailable" || len(metrics.Metrics) != 0 {
		t.Fatalf("unconfigured metric sources must return no samples: %+v", metrics)
	}
	logs, err := service.Logs(context.Background(), scope)
	if err != nil || logs.State != "unavailable" {
		t.Fatalf("logs should report their unavailable source explicitly: %+v, %v", logs, err)
	}
	alerts, err := service.Alerts(context.Background(), scope)
	if err != nil || alerts.State != "unavailable" {
		t.Fatalf("alerts should report their unavailable evaluator explicitly: %+v, %v", alerts, err)
	}
}

type testTelemetryAdapter struct {
	metrics []Metric
}

func (a testTelemetryAdapter) Name() string { return "test" }
func (a testTelemetryAdapter) Availability(context.Context) (string, error) {
	return "connected", nil
}
func (a testTelemetryAdapter) Metrics(context.Context, models.ObservabilityResource) ([]Metric, error) {
	return a.metrics, nil
}
func (a testTelemetryAdapter) ProjectMetrics(context.Context, Scope) ([]Metric, error) {
	return a.metrics, nil
}

func TestTelemetrySamplesAttachOnlyToDiscoveredResources(t *testing.T) {
	service, db := newTestService(t)
	if err := db.Create(&models.StorageVolume{
		ID: "0123456789abcdef0123456789abcdef", TenantID: 1, ProjectID: 1,
		OwnerID: 7, Name: "volume", State: "AVAILABLE", Driver: "cinder",
		IdempotencyKey: "key", RequestHash: "hash",
	}).Error; err != nil {
		t.Fatal(err)
	}
	service.adapters = []TelemetryAdapter{testTelemetryAdapter{metrics: []Metric{{
		Name: "atonix_resource_cpu_utilization_ratio", Value: 61, Unit: "%",
		ResourceID: "0123456789abcdef0123456789abcdef", ResourceType: "storage.volume",
		TenantID: 1, ProjectID: 1, ObservedAt: time.Now().UTC(),
	}}}}
	summary, err := service.MetricsSummary(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if summary.State != "available" || len(summary.Metrics) != 1 {
		t.Fatalf("expected the matching registered resource sample: %+v", summary)
	}
	var registry models.ObservabilityResource
	if err := db.First(&registry, "resource_id = ?", "0123456789abcdef0123456789abcdef").Error; err != nil {
		t.Fatal(err)
	}
	if registry.MonitoringStatus != "connected" {
		t.Fatalf("verified samples should update monitoring status, got %q", registry.MonitoringStatus)
	}

	service.adapters = []TelemetryAdapter{testTelemetryAdapter{metrics: []Metric{{
		Name: "atonix_resource_cpu_utilization_ratio", Value: 50, Unit: "%",
		ResourceID: "foreign-resource", ResourceType: "compute.instance",
		TenantID: 2, ProjectID: 1, ObservedAt: time.Now().UTC(),
	}}}}
	summary, err = service.MetricsSummary(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if summary.State != "unavailable" || len(summary.Metrics) != 0 {
		t.Fatalf("out-of-scope telemetry must not be returned: %+v", summary)
	}
}

func TestSecurityEventsRequireElevatedAuditPermissions(t *testing.T) {
	service, db := newTestService(t)
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 8, Role: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 8, Role: "viewer"}).Error; err != nil {
		t.Fatal(err)
	}
	events, err := service.Events(context.Background(), Scope{UserID: 8, TenantID: 1, ProjectID: 1})
	if err != nil || len(events) != 0 {
		t.Fatalf("project viewer should receive non-security project events only: %+v, %v", events, err)
	}
	if err := db.Create(&models.SecurityAuditEvent{
		TenantID: 1, ProjectID: 1, ActorID: 7, Action: "security.policy.create",
		ResourceType: "security_policy", ResourceID: "policy-1", Result: "success",
		Reason: "created", RequestID: "request", CorrelationID: "correlation",
		CreatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	events, err = service.Events(context.Background(), Scope{UserID: 7, TenantID: 1, ProjectID: 1})
	if err != nil || len(events) != 1 || events[0].Source != "security" {
		t.Fatalf("security owner should see scoped audit events: %+v, %v", events, err)
	}
}
