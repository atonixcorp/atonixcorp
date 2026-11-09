package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"atonixcorp/cloud/database/internal"
)

var ErrProviderUnavailable = errors.New("database provisioning is not configured")
var ErrInvalidRequest = errors.New("invalid database request")
var ErrIdempotencyConflict = errors.New("idempotency key was already used for a different request")

type Store interface {
	CreateInstance(*DBInstance) error
	UpdateInstance(*DBInstance) error
	GetInstance(id string) (*DBInstance, error)
	ListInstances(tenantID, projectID string) ([]DBInstance, error)
	ListInstancesByStatus(status string, limit int) ([]DBInstance, error)
	CanAccessProject(userID uint, tenantID, projectID string) (bool, error)
	CreateInstanceOperation(*DBInstance, *DatabaseOperation, string, string) (bool, error)
	GetOperation(id string) (*DatabaseOperation, error)
	GetOperationForDatabase(databaseID string) (*DatabaseOperation, error)
	UpdateOperation(*DatabaseOperation) error
}

type Controller struct {
	store         Store
	provider      DatabaseProvisioner
	policy        *Policy
	plans         map[string]Plan
	engineCatalog []EngineCatalogEntry
	defaultRegion string
}

func NewController(store Store, provider DatabaseProvisioner) *Controller {
	return &Controller{
		store: store, provider: provider, policy: NewPolicy(), plans: map[string]Plan{},
		engineCatalog: []EngineCatalogEntry{
			{ID: EnginePostgres, Name: "PostgreSQL", Category: "relational", Description: "Open-source relational database", Port: 5432, Versions: []string{"16", "17"}},
			{ID: EngineMySQL, Name: "MySQL", Category: "relational", Description: "Relational database", Port: 3306, Versions: []string{"8.4"}},
			{ID: EngineMariaDB, Name: "MariaDB", Category: "relational", Description: "Community relational database", Port: 3306, Versions: []string{"11.4"}},
			{ID: EngineRedis, Name: "Redis", Category: "key-value", Description: "In-memory key-value database", Port: 6379, Versions: []string{"7"}},
		},
	}
}

func (c *Controller) GetInstance(id string) (*DBInstance, error) {
	return c.store.GetInstance(id)
}

func (c *Controller) ListInstances(tenantID, projectID string) ([]DBInstance, error) {
	return c.store.ListInstances(tenantID, projectID)
}

func (c *Controller) SetPlans(plans map[string]Plan) {
	c.plans = plans
}

func (c *Controller) Plans() map[string]Plan {
	plans := make(map[string]Plan, len(c.plans))
	for key, plan := range c.plans {
		plan.Engines = append([]string(nil), plan.Engines...)
		plans[key] = plan
	}
	return plans
}

func (c *Controller) PlanCatalog() map[string]PlanCatalogEntry {
	catalog := make(map[string]PlanCatalogEntry, len(c.plans))
	enginesByID := make(map[string]bool)
	for _, entry := range c.EngineCatalog() {
		enginesByID[entry.ID] = entry.Enabled
	}
	for key, plan := range c.plans {
		engines := make([]string, 0, len(plan.Engines))
		for _, engine := range plan.Engines {
			if enginesByID[engine] {
				engines = append(engines, engine)
			}
		}
		catalog[key] = PlanCatalogEntry{
			ID: key, CPU: plan.CPU, MemoryMB: plan.MemoryMB, StorageGB: plan.StorageGB,
			Engines: engines, HighAvail: plan.HA,
			Enabled: c.planEnabled(key),
		}
	}
	return catalog
}

func (c *Controller) EngineCatalog() []EngineCatalogEntry {
	catalog := make([]EngineCatalogEntry, len(c.engineCatalog))
	for i, entry := range c.engineCatalog {
		entry.Versions = append([]string(nil), entry.Versions...)
		entry.Enabled = false
		if c.provider == nil {
			catalog[i] = entry
			continue
		}
		for _, version := range entry.Versions {
			for planID, plan := range c.plans {
				if contains(plan.Engines, entry.ID) && c.provider.Supports(entry.ID, version, planID, c.defaultRegion) {
					entry.Enabled = true
				}
			}
		}
		catalog[i] = entry
	}
	return catalog
}

func (c *Controller) planEnabled(planID string) bool {
	if c.provider == nil {
		return false
	}
	plan := c.plans[planID]
	for _, engine := range c.engineCatalog {
		if !contains(plan.Engines, engine.ID) {
			continue
		}
		for _, version := range engine.Versions {
			if c.provider.Supports(engine.ID, version, planID, c.defaultRegion) {
				return true
			}
		}
	}
	return false
}

func (c *Controller) SetDefaultRegion(region string) {
	c.defaultRegion = strings.TrimSpace(region)
}

func (c *Controller) CreateInstance(req CreateInstanceRequest) (*DBInstance, error) {
	result, err := c.CreateInstanceOperation(req, "")
	if err != nil {
		return nil, err
	}
	return result.Database, nil
}

func (c *Controller) CreateInstanceOperation(
	req CreateInstanceRequest, idempotencyKey string,
) (*CreateDatabaseResult, error) {
	if c.provider == nil {
		return nil, ErrProviderUnavailable
	}
	if err := c.policy.ValidateCreate(req); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	plan, ok := c.plans[req.Plan]
	if !ok {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("unsupported database plan %q", req.Plan))
	}
	if strings.TrimSpace(req.Region) == "" {
		req.Region = c.defaultRegion
	}
	if !contains(plan.Engines, req.Engine) || !c.provider.Supports(req.Engine, req.EngineVersion, req.Plan, req.Region) {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf(
			"database engine or service plan is not enabled for plan %q", req.Plan,
		))
	}
	if req.CPU != 0 && req.CPU != plan.CPU ||
		req.MemoryMB != 0 && req.MemoryMB != plan.MemoryMB ||
		req.StorageGB != 0 && req.StorageGB != plan.StorageGB {
		return nil, errors.Join(
			ErrInvalidRequest,
			fmt.Errorf("compute and storage values must match the selected backend plan"),
		)
	}
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if req.WorkloadProfile == "" {
		req.WorkloadProfile = "general-purpose"
	}
	if strings.TrimSpace(req.Region) == "" {
		req.Region = c.defaultRegion
	}
	port := map[string]int{
		EnginePostgres: 5432, EngineMySQL: 3306, EngineMariaDB: 3306, EngineRedis: 6379,
	}[req.Engine]
	if req.EngineVersion == "" {
		return nil, errors.Join(
			ErrInvalidRequest,
			fmt.Errorf("engine_version is required and must be offered by the database catalog"),
		)
	}
	now := time.Now().UTC()
	instance := &DBInstance{
		ID: internal.GenerateID("db"), TenantID: req.TenantID, ProjectID: req.ProjectID,
		Name: req.Name, Engine: req.Engine, EngineVersion: req.EngineVersion,
		Plan: req.Plan, WorkloadProfile: req.WorkloadProfile, CPU: plan.CPU,
		MemoryMB: plan.MemoryMB, StorageGB: plan.StorageGB, Region: req.Region,
		Status: "provisioning", Port: port,
		Provider:  c.provider.Name(),
		CreatedAt: now, UpdatedAt: now,
	}
	operation := &DatabaseOperation{
		ID: internal.GenerateID("op"), DatabaseID: instance.ID, TenantID: req.TenantID,
		ProjectID: req.ProjectID, Status: "queued", Stage: "QUEUED",
		CreatedAt: now, UpdatedAt: now,
	}
	requestHash := sha256.Sum256([]byte(strings.Join([]string{
		req.TenantID, req.ProjectID, req.Name, req.Engine, req.EngineVersion, req.Plan,
		req.WorkloadProfile, req.Region,
	}, "\x00")))
	replayed, err := c.store.CreateInstanceOperation(
		instance, operation, idempotencyKey, hex.EncodeToString(requestHash[:]),
	)
	if err != nil {
		return nil, fmt.Errorf("save database provisioning request: %w", err)
	}
	if replayed {
		instance, err = c.store.GetInstance(instance.ID)
		if err != nil {
			return nil, fmt.Errorf("load idempotent database request: %w", err)
		}
		operation, err = c.store.GetOperation(operation.ID)
		if err != nil {
			return nil, fmt.Errorf("load idempotent database operation: %w", err)
		}
	}
	baseURL := "/api/v1/tenants/" + req.TenantID + "/projects/" + req.ProjectID + "/operations/" + operation.ID
	return &CreateDatabaseResult{
		Database: instance, Operation: operation,
		StatusURL: baseURL, EventStreamURL: baseURL + "/events",
	}, nil
}

func (c *Controller) ProvisionInstance(id string) error {
	if c.provider == nil {
		return ErrProviderUnavailable
	}
	instance, err := c.store.GetInstance(id)
	if err != nil {
		return err
	}
	if instance.Status != "provisioning" {
		return nil
	}
	operation, err := c.store.GetOperationForDatabase(instance.ID)
	if err != nil {
		return fmt.Errorf("load database operation: %w", err)
	}
	operation.Status = "running"
	operation.Stage = "ALLOCATING_COMPUTE"
	operation.Attempt++
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return fmt.Errorf("persist database operation stage: %w", err)
	}
	if _, ok := c.plans[instance.Plan]; !ok || instance.Provider != c.provider.Name() || !c.provider.Supports(instance.Engine, instance.EngineVersion, instance.Plan, instance.Region) {
		return c.failProvision(instance, operation, ErrProviderUnavailable)
	}
	progress, err := c.provider.Provision(instance, func(stage string) error {
		instance.UpdatedAt = time.Now().UTC()
		if err := c.store.UpdateInstance(instance); err != nil {
			return err
		}
		operation.Stage = stage
		operation.UpdatedAt = instance.UpdatedAt
		return c.store.UpdateOperation(operation)
	})
	if err != nil {
		if errors.Is(err, ErrProvisioningFailed) {
			return c.failProvision(instance, operation, err)
		}
		return c.retryOrFail(instance, operation, err)
	}
	if !progress.InfrastructureReady {
		return nil
	}
	instance.Status = "starting"
	operation.Status = "blocked"
	operation.Stage = "BOOTSTRAP_ENGINE"
	operation.Reason = "Compute resources are ready; managed engine bootstrap and health verification are not configured."
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return fmt.Errorf("persist database bootstrap status: %w", err)
	}
	instance.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateInstance(instance); err != nil {
		return fmt.Errorf("persist provisioned database: %w", err)
	}
	return nil
}

func (c *Controller) retryOrFail(instance *DBInstance, operation *DatabaseOperation, cause error) error {
	var responseError interface{ HTTPStatusCode() int }
	if errors.As(cause, &responseError) &&
		responseError.HTTPStatusCode() >= 400 && responseError.HTTPStatusCode() < 500 &&
		responseError.HTTPStatusCode() != 408 && responseError.HTTPStatusCode() != 409 &&
		responseError.HTTPStatusCode() != 429 {
		return c.failProvision(instance, operation, cause)
	}
	operation.Reason = "A database infrastructure request could not complete; the operation will retry."
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return errors.Join(cause, fmt.Errorf("persist retryable operation failure: %w", err))
	}
	return cause
}

func (c *Controller) ProcessPendingInstances(limit int) error {
	if c.provider == nil {
		return ErrProviderUnavailable
	}
	instances, err := c.store.ListInstancesByStatus("provisioning", limit)
	if err != nil {
		return err
	}
	var failures []error
	for _, instance := range instances {
		if err := c.ProvisionInstance(instance.ID); err != nil {
			failures = append(failures, fmt.Errorf("database %s: %w", instance.ID, err))
		}
	}
	return errors.Join(failures...)
}

func (c *Controller) failProvision(
	instance *DBInstance, operation *DatabaseOperation, cause error,
) error {
	log.Printf("database provisioning failed operation=%s database=%s: %v", operation.ID, instance.ID, cause)
	instance.Status = "error"
	instance.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateInstance(instance); err != nil {
		return errors.Join(cause, fmt.Errorf("persist database provisioning failure: %w", err))
	}
	operation.Status = "failed"
	operation.Stage = "FAILED"
	operation.Reason = "The database could not be provisioned. Contact support with this operation ID."
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return errors.Join(cause, fmt.Errorf("persist database operation failure: %w", err))
	}
	return cause
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
