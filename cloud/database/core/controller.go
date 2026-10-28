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

type OpenStackClient interface {
	FindServerByName(name string) (id string, found bool, err error)
	CreateServer(name, imageID, flavorID, networkID, securityGroupID string) (string, error)
	FindVolumeByName(name string) (id string, found bool, err error)
	CreateVolume(name string, sizeGB int) (string, error)
	GetServerState(serverID string) (status, ip string, err error)
	IsVolumeAttached(serverID, volumeID string) (bool, error)
	GetVolumeState(volumeID string) (string, error)
	AttachVolume(serverID, volumeID string) error
}

type Controller struct {
	store         Store
	os            OpenStackClient
	policy        *Policy
	plans         map[string]Plan
	engineImages  map[string]string
	engineCatalog []EngineCatalogEntry
	networkID     string
	securityGroup string
	defaultRegion string
}

func NewController(store Store, os OpenStackClient) *Controller {
	return &Controller{
		store: store, os: os, policy: NewPolicy(), plans: map[string]Plan{},
		engineImages: map[string]string{}, engineCatalog: []EngineCatalogEntry{
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
			Enabled: c.os != nil && c.networkID != "" && plan.FlavorID != "" && len(engines) > 0,
		}
	}
	return catalog
}

func (c *Controller) EngineCatalog() []EngineCatalogEntry {
	catalog := make([]EngineCatalogEntry, len(c.engineCatalog))
	for i, entry := range c.engineCatalog {
		entry.Versions = append([]string(nil), entry.Versions...)
		entry.Enabled = false
		if c.os == nil || c.networkID == "" {
			catalog[i] = entry
			continue
		}
		for _, version := range entry.Versions {
			if c.engineImages[engineImageKey(entry.ID, version)] != "" {
				entry.Enabled = true
				break
			}
		}
		catalog[i] = entry
	}
	return catalog
}

func engineImageKey(engine, version string) string {
	return engine + ":" + version
}

func (c *Controller) SetProvisioningConfig(
	engineImages map[string]string, networkID, securityGroupID string,
) {
	c.engineImages = engineImages
	c.networkID = strings.TrimSpace(networkID)
	c.securityGroup = strings.TrimSpace(securityGroupID)
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
	if c.os == nil || c.networkID == "" {
		return nil, ErrProviderUnavailable
	}
	if err := c.policy.ValidateCreate(req); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	plan, ok := c.plans[req.Plan]
	if !ok {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("unsupported database plan %q", req.Plan))
	}
	if !contains(plan.Engines, req.Engine) || plan.FlavorID == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf(
			"database engine or service plan is not enabled for plan %q", req.Plan,
		))
	}
	imageID := strings.TrimSpace(c.engineImages[engineImageKey(req.Engine, req.EngineVersion)])
	if imageID == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf(
			"no prepared database image is configured for engine %q version %q",
			req.Engine, req.EngineVersion,
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
		Status: "provisioning", Port: port, NetworkID: c.networkID,
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
	if c.os == nil {
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
	plan, ok := c.plans[instance.Plan]
	if !ok || plan.FlavorID == "" || c.engineImages[engineImageKey(instance.Engine, instance.EngineVersion)] == "" {
		return c.failProvision(instance, operation, errors.New("database plan or engine image is no longer configured"))
	}
	resourceName := instance.Name + "-" + instance.ID
	if instance.OpenStackServerID == "" {
		serverID, found, err := c.os.FindServerByName(resourceName)
		if err != nil {
			return c.retryOrFail(instance, operation, fmt.Errorf("find Nova server: %w", err))
		}
		if !found {
			serverID, err = c.os.CreateServer(
				resourceName, c.engineImages[engineImageKey(instance.Engine, instance.EngineVersion)],
				plan.FlavorID, c.networkID, c.securityGroup,
			)
			if err != nil {
				return c.retryOrFail(instance, operation, fmt.Errorf("create Nova server: %w", err))
			}
		}
		instance.OpenStackServerID = serverID
		if err := c.store.UpdateInstance(instance); err != nil {
			return fmt.Errorf("persist Nova server ID: %w", err)
		}
	}
	serverStatus, ip, err := c.os.GetServerState(instance.OpenStackServerID)
	if err != nil {
		return c.retryOrFail(instance, operation, fmt.Errorf("check Nova server status: %w", err))
	}
	if serverStatus == "ERROR" {
		return c.failProvision(instance, operation, errors.New("Nova server entered ERROR state"))
	}
	if serverStatus != "ACTIVE" {
		return nil
	}
	operation.Stage = "PROVISIONING_STORAGE"
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return fmt.Errorf("persist database storage stage: %w", err)
	}
	if instance.OpenStackVolumeID == "" {
		volumeName := resourceName + "-data"
		volumeID, found, err := c.os.FindVolumeByName(volumeName)
		if err != nil {
			return c.retryOrFail(instance, operation, fmt.Errorf("find Cinder volume: %w", err))
		}
		if !found {
			volumeID, err = c.os.CreateVolume(volumeName, instance.StorageGB)
			if err != nil {
				return c.retryOrFail(instance, operation, fmt.Errorf("create Cinder volume: %w", err))
			}
		}
		instance.OpenStackVolumeID = volumeID
		if err := c.store.UpdateInstance(instance); err != nil {
			return fmt.Errorf("persist Cinder volume ID: %w", err)
		}
	}
	volumeStatus, err := c.os.GetVolumeState(instance.OpenStackVolumeID)
	if err != nil {
		return c.retryOrFail(instance, operation, fmt.Errorf("check Cinder volume status: %w", err))
	}
	switch volumeStatus {
	case "error":
		return c.failProvision(instance, operation, errors.New("Cinder volume entered error state"))
	case "available":
	case "in-use":
		attached, err := c.os.IsVolumeAttached(instance.OpenStackServerID, instance.OpenStackVolumeID)
		if err != nil {
			return c.retryOrFail(instance, operation, fmt.Errorf("check Nova volume attachment: %w", err))
		}
		if !attached {
			return c.failProvision(instance, operation, errors.New("Cinder volume is in-use but not attached to the expected server"))
		}
		instance.OpenStackVolumeAttached = true
	default:
		return nil
	}
	if !instance.OpenStackVolumeAttached {
		attached, err := c.os.IsVolumeAttached(instance.OpenStackServerID, instance.OpenStackVolumeID)
		if err != nil {
			return c.retryOrFail(instance, operation, fmt.Errorf("check Nova volume attachment: %w", err))
		}
		if !attached {
			if err := c.os.AttachVolume(instance.OpenStackServerID, instance.OpenStackVolumeID); err != nil {
				return c.retryOrFail(instance, operation, fmt.Errorf("attach Cinder volume: %w", err))
			}
		}
		instance.OpenStackVolumeAttached = true
	}
	if ip == "" {
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
	operation.Reason = "A provider request could not complete; the operation will retry."
	operation.UpdatedAt = time.Now().UTC()
	if err := c.store.UpdateOperation(operation); err != nil {
		return errors.Join(cause, fmt.Errorf("persist retryable operation failure: %w", err))
	}
	return cause
}

func (c *Controller) ProcessPendingInstances(limit int) error {
	if c.os == nil {
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
