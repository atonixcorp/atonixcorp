package observability

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrForbidden = errors.New("observability project access denied")
	ErrNotFound  = errors.New("observability resource not found")
)

type Scope struct {
	UserID    uint
	TenantID  uint
	ProjectID uint
}

func (s *Service) markMonitored(ctx context.Context, scope Scope, resources []models.ObservabilityResource) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, resource := range resources {
			if err := tx.Model(&models.ObservabilityResource{}).
				Where(
					"tenant_id = ? AND project_id = ? AND resource_type = ? AND resource_id = ? AND resource_status <> ?",
					scope.TenantID, scope.ProjectID, resource.ResourceType, resource.ResourceID, "DELETED",
				).Update("monitoring_status", "connected").Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type Resource struct {
	ID               string    `json:"id"`
	ResourceType     string    `json:"resource_type"`
	Provider         string    `json:"provider"`
	Name             string    `json:"name"`
	Region           string    `json:"region,omitempty"`
	ProjectID        uint      `json:"project_id"`
	Status           string    `json:"status"`
	MonitoringStatus string    `json:"monitoring_status"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

type ResourceGroup struct {
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Health string `json:"health"`
	Source string `json:"source"`
}

type Overview struct {
	Health             string          `json:"health"`
	HealthyResources   *int            `json:"healthy_resources"`
	WarningResources   *int            `json:"warning_resources"`
	CriticalResources  *int            `json:"critical_resources"`
	UnknownHealth      int             `json:"unknown_health"`
	ActiveAlerts       *int            `json:"active_alerts"`
	TelemetryState     string          `json:"telemetry_state"`
	TelemetryFreshness *time.Time      `json:"telemetry_freshness"`
	ResourceGroups     []ResourceGroup `json:"resource_groups"`
	MetricAvailability string          `json:"metric_availability"`
}

type Integration struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	State       string `json:"state"`
	Description string `json:"description"`
}

type Metric struct {
	Name         string    `json:"name"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	ObservedAt   time.Time `json:"observed_at"`
	TenantID     uint      `json:"-"`
	ProjectID    uint      `json:"-"`
}

type MetricsResponse struct {
	ResourceID   string   `json:"resource_id"`
	ResourceType string   `json:"resource_type"`
	State        string   `json:"state"`
	Reason       string   `json:"reason,omitempty"`
	Metrics      []Metric `json:"metrics"`
}

type MetricsOverview struct {
	State   string   `json:"state"`
	Reason  string   `json:"reason"`
	Metrics []Metric `json:"metrics"`
}

type AvailabilityResponse struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type Dashboard struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ResourceKinds []string `json:"resource_kinds"`
}

type Event struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resource_id"`
	Source     string    `json:"source"`
	Severity   string    `json:"severity"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

type ResourceCollector interface {
	Name() string
	ResourceTypes() []string
	Discover(context.Context, Scope) ([]models.ObservabilityResource, bool, error)
}

type TelemetryAdapter interface {
	Name() string
	Metrics(context.Context, models.ObservabilityResource) ([]Metric, error)
	Availability(context.Context) (string, error)
}

type ProjectTelemetryAdapter interface {
	ProjectMetrics(context.Context, Scope) ([]Metric, error)
}

type Service struct {
	db                    *gorm.DB
	collectors            []ResourceCollector
	adapters              []TelemetryAdapter
	prometheusConfigError error
}

func NewService(db *gorm.DB) *Service {
	service := &Service{
		db: db,
		collectors: []ResourceCollector{
			&storageCollector{db: db},
			&containerCollector{db: db},
		},
	}
	prometheus, err := newPrometheusAdapterFromEnvironment()
	if err != nil {
		service.prometheusConfigError = err
	} else if prometheus != nil {
		service.adapters = append(service.adapters, prometheus)
	}
	return service
}

func (s *Service) Overview(ctx context.Context, scope Scope) (Overview, error) {
	resources, err := s.Resources(ctx, scope)
	if err != nil {
		return Overview{}, err
	}
	metricSummary, err := s.MetricsSummary(ctx, scope)
	if err != nil {
		return Overview{}, err
	}
	groups := defaultResourceGroups()
	for _, resource := range resources {
		for i := range groups {
			if groups[i].Kind == resource.ResourceType {
				groups[i].Count++
				break
			}
		}
	}
	var freshness *time.Time
	for _, metric := range metricSummary.Metrics {
		if freshness == nil || metric.ObservedAt.After(*freshness) {
			observedAt := metric.ObservedAt
			freshness = &observedAt
		}
	}
	return Overview{
		Health: "unknown", UnknownHealth: len(resources),
		TelemetryState: metricSummary.State, TelemetryFreshness: freshness,
		ResourceGroups: groups, MetricAvailability: metricSummary.State,
	}, nil
}

func (s *Service) Resources(ctx context.Context, scope Scope) ([]Resource, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	if err := s.syncInventory(ctx, scope); err != nil {
		return nil, err
	}
	var records []models.ObservabilityResource
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ? AND resource_status <> ?", scope.TenantID, scope.ProjectID, "DELETED").
		Order("resource_type ASC, name ASC").Limit(1000).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]Resource, 0, len(records))
	for _, record := range records {
		result = append(result, Resource{
			ID: record.ResourceID, ResourceType: record.ResourceType,
			Provider: record.Provider, Name: record.Name, Region: record.Region,
			ProjectID: record.ProjectID, Status: record.ResourceStatus,
			MonitoringStatus: record.MonitoringStatus, LastSeenAt: record.LastSeenAt,
		})
	}
	return result, nil
}

func (s *Service) Metrics(ctx context.Context, scope Scope, resourceID string) (MetricsResponse, error) {
	return s.ResourceMetrics(ctx, scope, "", resourceID)
}

func (s *Service) MetricsSummary(ctx context.Context, scope Scope) (MetricsOverview, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return MetricsOverview{}, err
	}
	if s.prometheusConfigError != nil {
		return MetricsOverview{
			State:   "invalid_configuration",
			Reason:  "The configured Prometheus endpoint is invalid; correct OBSERVABILITY_PROMETHEUS_URL.",
			Metrics: []Metric{},
		}, nil
	}
	resources, err := s.Resources(ctx, scope)
	if err != nil {
		return MetricsOverview{}, err
	}
	knownResources := make(map[string]models.ObservabilityResource, len(resources))
	for _, resource := range resources {
		knownResources[resource.ResourceType+"\x00"+resource.ID] = models.ObservabilityResource{
			TenantID: scope.TenantID, ProjectID: scope.ProjectID,
			ResourceType: resource.ResourceType, ResourceID: resource.ID,
		}
	}
	for _, adapter := range s.adapters {
		projectAdapter, ok := adapter.(ProjectTelemetryAdapter)
		if !ok {
			continue
		}
		metrics, err := projectAdapter.ProjectMetrics(ctx, scope)
		if err != nil {
			log.Printf("observability adapter %s project metrics failed: %v", adapter.Name(), err)
			return MetricsOverview{
				State: "unavailable", Reason: "The configured telemetry adapter request failed; see API logs.",
				Metrics: []Metric{},
			}, nil
		}
		if len(metrics) > 0 {
			knownSamples := make([]Metric, 0, len(metrics))
			discovered := make([]models.ObservabilityResource, 0)
			for _, metric := range metrics {
				if metric.TenantID != scope.TenantID || metric.ProjectID != scope.ProjectID {
					log.Printf("observability adapter %s returned project metrics outside the requested scope", adapter.Name())
					return MetricsOverview{
						State: "unavailable", Reason: "The configured telemetry adapter returned an invalid project scope.",
						Metrics: []Metric{},
					}, nil
				}
				resource, exists := knownResources[metric.ResourceType+"\x00"+metric.ResourceID]
				if exists {
					knownSamples = append(knownSamples, metric)
					discovered = append(discovered, resource)
				}
			}
			if len(knownSamples) > 0 {
				if err := s.markMonitored(ctx, scope, discovered); err != nil {
					return MetricsOverview{}, err
				}
				return MetricsOverview{
					State: "available", Reason: "Metrics were returned by the configured telemetry adapter.",
					Metrics: knownSamples,
				}, nil
			}
		}
	}
	return MetricsOverview{
		State: "unavailable", Reason: "No telemetry adapter is currently supplying project metrics for this project.",
		Metrics: []Metric{},
	}, nil
}

func (s *Service) ResourceMetrics(ctx context.Context, scope Scope, resourceType, resourceID string) (MetricsResponse, error) {
	resource, err := s.resource(ctx, scope, resourceType, resourceID)
	if err != nil {
		return MetricsResponse{}, err
	}
	for _, adapter := range s.adapters {
		metrics, adapterErr := adapter.Metrics(ctx, resource)
		if adapterErr != nil {
			log.Printf("observability adapter %s failed for resource type %s: %v", adapter.Name(), resource.ResourceType, adapterErr)
			return MetricsResponse{
				ResourceID: resource.ResourceID, ResourceType: resource.ResourceType,
				State: "unavailable", Reason: "The configured telemetry adapter request failed; see API logs.",
				Metrics: []Metric{},
			}, nil
		}
		if metrics != nil {
			for _, metric := range metrics {
				if metric.ResourceID != resource.ResourceID || metric.ResourceType != resource.ResourceType ||
					metric.TenantID != resource.TenantID || metric.ProjectID != resource.ProjectID {
					log.Printf("observability adapter %s returned a metric outside the requested resource scope", adapter.Name())
					return MetricsResponse{
						ResourceID: resource.ResourceID, ResourceType: resource.ResourceType,
						State: "unavailable", Reason: "The configured telemetry adapter returned an invalid resource scope.",
						Metrics: []Metric{},
					}, nil
				}
			}
			if len(metrics) > 0 {
				if err := s.markMonitored(ctx, scope, []models.ObservabilityResource{resource}); err != nil {
					return MetricsResponse{}, err
				}
			}
			return MetricsResponse{
				ResourceID: resource.ResourceID, ResourceType: resource.ResourceType,
				State: "available", Metrics: metrics,
			}, nil
		}
	}
	return MetricsResponse{
		ResourceID: resource.ResourceID, ResourceType: resource.ResourceType,
		State: "unavailable", Reason: "No configured telemetry adapter returned samples for this resource.",
		Metrics: []Metric{},
	}, nil
}

func (s *Service) GetResource(ctx context.Context, scope Scope, resourceType, resourceID string) (Resource, error) {
	record, err := s.resource(ctx, scope, resourceType, resourceID)
	if err != nil {
		return Resource{}, err
	}
	return Resource{
		ID: record.ResourceID, ResourceType: record.ResourceType,
		Provider: record.Provider, Name: record.Name, Region: record.Region,
		ProjectID: record.ProjectID, Status: record.ResourceStatus,
		MonitoringStatus: record.MonitoringStatus, LastSeenAt: record.LastSeenAt,
	}, nil
}

func (s *Service) Integrations(ctx context.Context, scope Scope) ([]Integration, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	configured := func(name string) string {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return "configured_unverified"
		}
		return "not_configured"
	}
	prometheusState := configured("OBSERVABILITY_PROMETHEUS_URL")
	if s.prometheusConfigError != nil {
		prometheusState = "invalid_configuration"
	} else if len(s.adapters) > 0 {
		for _, adapter := range s.adapters {
			if adapter.Name() != "prometheus" {
				continue
			}
			state, err := adapter.Availability(ctx)
			if err != nil {
				prometheusState = "unavailable"
			} else {
				prometheusState = state
			}
		}
	}
	return []Integration{
		{Name: "OpenStack Ceilometer", Provider: "openstack", State: configured("CEILOMETER_API_URL"), Description: "OpenStack service meters and resource samples; no live collection is claimed until authenticated collection is verified."},
		{Name: "Prometheus-compatible metrics", Provider: "prometheus", State: prometheusState, Description: "Prometheus-compatible adapter queries only AtonixCorp resource-labeled metric series; credentials remain server-side."},
		{Name: "OpenHPC / Slurm", Provider: "slurm", State: configured("OBSERVABILITY_SLURM_API_URL"), Description: "Slurm scheduler, partition, job, and node telemetry adapter."},
		{Name: "Kubernetes metrics", Provider: "kubernetes", State: "not_configured", Description: "Runtime-specific pod metrics are not yet connected to this workspace."},
		{Name: "Database telemetry", Provider: "database", State: "not_configured", Description: "Database service metrics are not yet exposed to the observability pipeline."},
	}, nil
}

func (s *Service) Logs(ctx context.Context, scope Scope) (AvailabilityResponse, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return AvailabilityResponse{}, err
	}
	return AvailabilityResponse{State: "unavailable", Reason: "No authorized log collector is configured."}, nil
}

func (s *Service) Alerts(ctx context.Context, scope Scope) (AvailabilityResponse, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return AvailabilityResponse{}, err
	}
	return AvailabilityResponse{State: "unavailable", Reason: "No telemetry-backed alert evaluator is configured."}, nil
}

func (s *Service) Dashboards(ctx context.Context, scope Scope) ([]Dashboard, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	return []Dashboard{
		{ID: "cloud-overview", Name: "Cloud Overview", Description: "Cross-platform resource health and telemetry.", ResourceKinds: []string{"compute.instance", "container.workload", "storage.volume", "database.instance", "network.network", "hpc.node"}},
		{ID: "compute-overview", Name: "Compute Overview", Description: "Compute resources and provider-reported utilization.", ResourceKinds: []string{"compute.instance", "compute.hypervisor", "compute.node"}},
		{ID: "network-overview", Name: "Network Overview", Description: "Network resources and provider-reported traffic.", ResourceKinds: []string{"network.network", "network.router", "network.interface", "network.load_balancer"}},
		{ID: "storage-overview", Name: "Storage Overview", Description: "Storage inventory and provider-reported capacity.", ResourceKinds: []string{"storage.volume", "storage.object"}},
		{ID: "containers-overview", Name: "Containers Overview", Description: "Kubernetes workloads and node metrics.", ResourceKinds: []string{"container.cluster", "container.node", "container.workload"}},
		{ID: "database-overview", Name: "Database Overview", Description: "Database service inventory and metrics.", ResourceKinds: []string{"database.instance"}},
		{ID: "hpc-overview", Name: "HPC Overview", Description: "OpenHPC nodes, Slurm jobs, and scheduler state.", ResourceKinds: []string{"hpc.node", "hpc.job"}},
		{ID: "gpu-training", Name: "GPU / AI Training", Description: "GPU capacity and training workload telemetry.", ResourceKinds: []string{"hpc.node", "hpc.job", "container.workload"}},
		{ID: "security-overview", Name: "Security Overview", Description: "Security audit events and platform security telemetry.", ResourceKinds: []string{"security.event"}},
	}, nil
}

func (s *Service) Events(ctx context.Context, scope Scope) ([]Event, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return nil, err
	}
	result := make([]Event, 0, 100)
	var containerEvents []models.ContainerDeploymentEvent
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", scope.TenantID, scope.ProjectID).
		Order("id DESC").Limit(100).Find(&containerEvents).Error; err != nil {
		return nil, err
	}
	for _, event := range containerEvents {
		result = append(result, Event{
			ID: fmt.Sprintf("container-%d", event.ID), ResourceID: event.DeploymentID,
			Source: "containers", Severity: eventSeverity(event.NewState),
			Summary: event.Reason, CreatedAt: event.CreatedAt,
		})
	}
	var storageEvents []models.StorageAuditEvent
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", scope.TenantID, scope.ProjectID).
		Order("id DESC").Limit(100).Find(&storageEvents).Error; err != nil {
		return nil, err
	}
	for _, event := range storageEvents {
		result = append(result, Event{
			ID: fmt.Sprintf("storage-%d", event.ID), ResourceID: event.ResourceID,
			Source: "storage", Severity: eventSeverity(event.Result),
			Summary: event.Action + ": " + event.Result, CreatedAt: event.CreatedAt,
		})
	}
	canReadAudit, err := s.canReadAudit(ctx, scope)
	if err != nil {
		return nil, err
	}
	if !canReadAudit {
		sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
		if len(result) > 100 {
			result = result[:100]
		}
		return result, nil
	}
	var audits []models.SecurityAuditEvent
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", scope.TenantID, scope.ProjectID).
		Order("id DESC").Limit(100).Find(&audits).Error; err != nil {
		return nil, err
	}
	for _, audit := range audits {
		result = append(result, Event{
			ID: fmt.Sprintf("security-%d", audit.ID), ResourceID: audit.ResourceID,
			Source: "security", Severity: eventSeverity(audit.Result),
			Summary: audit.Action + ": " + audit.Result, CreatedAt: audit.CreatedAt,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if len(result) > 100 {
		result = result[:100]
	}
	return result, nil
}

func (s *Service) canReadAudit(ctx context.Context, scope Scope) (bool, error) {
	if scope.UserID == 0 || scope.TenantID == 0 || scope.ProjectID == 0 {
		return false, ErrForbidden
	}
	var membership struct {
		TenantRole  string
		ProjectRole string
	}
	err := s.db.WithContext(ctx).Table("projects").
		Select("tenant_memberships.role AS tenant_role, project_memberships.role AS project_role").
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = projects.tenant_id AND tenant_memberships.user_id = ?", scope.UserID).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id AND project_memberships.user_id = ?", scope.UserID).
		Where("projects.id = ? AND projects.tenant_id = ?", scope.ProjectID, scope.TenantID).
		Take(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	tenantCanRead := membership.TenantRole == "owner" || membership.TenantRole == "admin"
	projectCanRead := membership.ProjectRole == "owner" ||
		membership.ProjectRole == "admin" ||
		membership.ProjectRole == "security_auditor"
	if !tenantCanRead || !projectCanRead {
		return false, nil
	}
	return true, nil
}

func (s *Service) resource(ctx context.Context, scope Scope, resourceType, resourceID string) (models.ObservabilityResource, error) {
	if err := s.authorize(ctx, scope); err != nil {
		return models.ObservabilityResource{}, err
	}
	if err := s.syncInventory(ctx, scope); err != nil {
		return models.ObservabilityResource{}, err
	}
	var resource models.ObservabilityResource
	query := s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND resource_id = ? AND resource_status <> ?",
		scope.TenantID, scope.ProjectID, resourceID, "DELETED",
	)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	err := query.First(&resource).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ObservabilityResource{}, ErrNotFound
	}
	return resource, err
}

func (s *Service) authorize(ctx context.Context, scope Scope) error {
	if scope.UserID == 0 || scope.TenantID == 0 || scope.ProjectID == 0 {
		return ErrForbidden
	}
	var count int64
	err := s.db.WithContext(ctx).Table("projects").
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = projects.tenant_id AND tenant_memberships.user_id = ?", scope.UserID).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id AND project_memberships.user_id = ?", scope.UserID).
		Where("projects.id = ? AND projects.tenant_id = ?", scope.ProjectID, scope.TenantID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Service) syncInventory(ctx context.Context, scope Scope) error {
	var discovered []models.ObservabilityResource
	sourceComplete := make(map[string]bool)
	for _, collector := range s.collectors {
		items, complete, err := collector.Discover(ctx, scope)
		if err != nil {
			return fmt.Errorf("%s resource discovery: %w", collector.Name(), err)
		}
		discovered = append(discovered, items...)
		for _, resourceType := range collector.ResourceTypes() {
			sourceComplete[resourceType] = complete
		}
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range discovered {
			record := discovered[i]
			record.LastSeenAt = now
			if err := upsertResource(tx, &record); err != nil {
				return err
			}
		}
		for resourceType, complete := range sourceComplete {
			if !complete {
				continue
			}
			var ids []string
			for _, resource := range discovered {
				if resource.ResourceType == resourceType {
					ids = append(ids, resource.ResourceID)
				}
			}
			query := tx.Model(&models.ObservabilityResource{}).
				Where("tenant_id = ? AND project_id = ? AND resource_type = ?", scope.TenantID, scope.ProjectID, resourceType)
			if len(ids) > 0 {
				query = query.Where("resource_id NOT IN ?", ids)
			}
			if err := query.Update("resource_status", "DELETED").Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertResource(tx *gorm.DB, resource *models.ObservabilityResource) error {
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "project_id"}, {Name: "resource_type"}, {Name: "resource_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"provider", "name", "region", "resource_status", "last_seen_at", "updated_at",
		}),
	}).Create(resource).Error
}

func defaultResourceGroups() []ResourceGroup {
	return []ResourceGroup{
		{Kind: "compute.instance", Health: "unknown", Source: "openstack_ceilometer"},
		{Kind: "compute.hypervisor", Health: "unknown", Source: "openstack_ceilometer"},
		{Kind: "compute.node", Health: "unknown", Source: "openstack_ceilometer"},
		{Kind: "container.cluster", Health: "unknown", Source: "kubernetes"},
		{Kind: "container.node", Health: "unknown", Source: "kubernetes"},
		{Kind: "container.workload", Health: "unknown", Source: "kubernetes"},
		{Kind: "storage.volume", Health: "unknown", Source: "cinder"},
		{Kind: "storage.object", Health: "unknown", Source: "swift"},
		{Kind: "database.instance", Health: "unknown", Source: "database"},
		{Kind: "network.network", Health: "unknown", Source: "neutron"},
		{Kind: "network.router", Health: "unknown", Source: "neutron"},
		{Kind: "network.interface", Health: "unknown", Source: "neutron"},
		{Kind: "network.load_balancer", Health: "unknown", Source: "octavia"},
		{Kind: "hpc.node", Health: "unknown", Source: "slurm"},
		{Kind: "hpc.job", Health: "unknown", Source: "slurm"},
	}
}

func eventSeverity(result string) string {
	switch strings.ToLower(result) {
	case "denied", "failed", "error", "degraded", "error_state":
		return "warning"
	default:
		return "info"
	}
}

type storageCollector struct{ db *gorm.DB }

func (c *storageCollector) Name() string            { return "storage" }
func (c *storageCollector) ResourceTypes() []string { return []string{"storage.volume"} }

func (c *storageCollector) Discover(ctx context.Context, scope Scope) ([]models.ObservabilityResource, bool, error) {
	var volumes []models.StorageVolume
	if err := c.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ? AND deleted_at IS NULL AND state <> ?", scope.TenantID, scope.ProjectID, "DELETED").
		Order("id ASC").Limit(1001).Find(&volumes).Error; err != nil {
		return nil, false, err
	}
	complete := len(volumes) <= 1000
	if !complete {
		volumes = volumes[:1000]
	}
	result := make([]models.ObservabilityResource, 0, len(volumes))
	for _, volume := range volumes {
		result = append(result, models.ObservabilityResource{
			TenantID: scope.TenantID, ProjectID: scope.ProjectID,
			ResourceType: "storage.volume", ResourceID: volume.ID,
			Provider: volume.Driver, Name: volume.Name, Region: volume.Region,
			ResourceStatus: volume.State, MonitoringStatus: "not_configured",
			FirstSeenAt: volume.CreatedAt,
		})
	}
	return result, complete, nil
}

type containerCollector struct{ db *gorm.DB }

func (c *containerCollector) Name() string            { return "kubernetes" }
func (c *containerCollector) ResourceTypes() []string { return []string{"container.workload"} }

func (c *containerCollector) Discover(ctx context.Context, scope Scope) ([]models.ObservabilityResource, bool, error) {
	var deployments []models.ContainerDeployment
	if err := c.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ? AND deleted_at IS NULL", scope.TenantID, scope.ProjectID).
		Order("id ASC").Limit(1001).Find(&deployments).Error; err != nil {
		return nil, false, err
	}
	complete := len(deployments) <= 1000
	if !complete {
		deployments = deployments[:1000]
	}
	result := make([]models.ObservabilityResource, 0, len(deployments))
	for _, deployment := range deployments {
		result = append(result, models.ObservabilityResource{
			TenantID: scope.TenantID, ProjectID: scope.ProjectID,
			ResourceType: "container.workload", ResourceID: deployment.ID,
			Provider: "kubernetes", Name: deployment.Name,
			ResourceStatus: deployment.State, MonitoringStatus: "not_configured",
			FirstSeenAt: deployment.CreatedAt,
		})
	}
	return result, complete, nil
}
