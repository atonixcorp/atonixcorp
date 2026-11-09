package cloud

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"atonixcorp/api/internal/authorization"
	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrProviderUnavailable         = errors.New("Nova is not configured")
	ErrProviderRequest             = errors.New("OpenStack provider request failed")
	ErrProviderNotFound            = errors.New("Nova instance not found")
	ErrSecurityGroupUnavailable    = errors.New("OpenStack security groups are not configured")
	ErrProjectProvisioningDisabled = errors.New("organization-scoped OpenStack provisioning is disabled")
	ErrForbidden                   = errors.New("project access denied")
	ErrNotFound                    = errors.New("instance not found")
	ErrInvalid                     = errors.New("invalid instance request")
	ErrConflict                    = errors.New("instance request conflicts with an existing resource")
)

var instanceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
var catalogResourceIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type ComputeProvider interface {
	Name() string
	Region() string
	GetInstance(context.Context, string) (ProviderInstance, error)
	StartInstance(context.Context, string) error
	StopInstance(context.Context, string) error
	RebootInstance(context.Context, string) error
	DeleteInstance(context.Context, string) error
}

type ComputeCapabilities struct {
	Compute       bool `json:"compute"`
	Network       bool `json:"network"`
	Image         bool `json:"image"`
	BlockStorage  bool `json:"blockStorage"`
	Identity      bool `json:"identity"`
	ObjectStorage bool `json:"objectStorage"`
	LoadBalancer  bool `json:"loadBalancer"`
	DNS           bool `json:"dns"`
	KeyManagement bool `json:"keyManagement"`
	Orchestration bool `json:"orchestration"`
}

type Provider = ComputeProvider

type ProjectScopedComputeProvider interface {
	CreateInstanceForProject(
		context.Context, uint, uint, ProviderCreateRequest,
	) (ProviderInstance, error)
}

type ProjectScopedProvider = ProjectScopedComputeProvider

type ProviderCreateRequest struct {
	ID        string
	Name      string
	ImageID   string
	SizeID    string
	NetworkID string
	KeyName   string
}

type ProviderInstance struct {
	ID        string
	Status    string
	PrivateIP string
	PublicIP  string
}

type CatalogImage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	MinDiskGB   int    `json:"min_disk_gb"`
	MinMemoryMB int    `json:"min_memory_mb"`
}

type ComputeSize struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	VCPUs       int    `json:"vcpus"`
	MemoryMB    int    `json:"memory_mb"`
	DiskGB      int    `json:"disk_gb"`
	EphemeralGB int    `json:"ephemeral_gb"`
	SwapMB      int    `json:"swap_mb"`
}

type CatalogFlavor = ComputeSize

type CatalogNetwork struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	AdminStateUp bool   `json:"admin_state_up"`
	Shared       bool   `json:"shared"`
	External     bool   `json:"external"`
}

type ComputeCatalog struct {
	Region       string              `json:"-"`
	RegionID     string              `json:"region_id"`
	Capabilities ComputeCapabilities `json:"capabilities"`
	Images       []CatalogImage      `json:"images"`
	Sizes        []ComputeSize       `json:"sizes"`
	Networks     []CatalogNetwork    `json:"networks"`
	Provisioning bool                `json:"provisioning_enabled"`
}

type ComputeCatalogProvider interface {
	ComputeCatalog(context.Context) (ComputeCatalog, error)
}

type CatalogProvider = ComputeCatalogProvider

type ComputeSecurityGroupProvider interface {
	ListSecurityGroups(context.Context) ([]SecurityGroup, error)
	CreateSecurityGroup(context.Context, string) (SecurityGroup, error)
}

type SecurityGroupProvider = ComputeSecurityGroupProvider

type SecurityGroup struct {
	ID          string              `json:"-"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	ProjectID   string              `json:"-"`
	Rules       []SecurityGroupRule `json:"rules"`
}

type SecurityGroupRule struct {
	ID             string `json:"-"`
	Direction      string `json:"direction"`
	Ethertype      string `json:"ethertype"`
	Protocol       string `json:"protocol"`
	PortRangeMin   *int   `json:"port_range_min,omitempty"`
	PortRangeMax   *int   `json:"port_range_max,omitempty"`
	RemoteIPPrefix string `json:"remote_ip_prefix,omitempty"`
}

type CreateSecurityGroupRequest struct {
	Name string `json:"name"`
}

type CreateInstanceRequest struct {
	TenantID  uint   `json:"tenant_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	RegionID  string `json:"region_id"`
	ImageID   string `json:"image_id"`
	SizeID    string `json:"size_id"`
	NetworkID string `json:"network_id"`
	KeyName   string `json:"key_name"`
}

type Service struct {
	db        *gorm.DB
	provider  ComputeProvider
	registry  *ComputeProviderRegistry
	secretKey []byte
}

func NewService(db *gorm.DB, provider ComputeProvider, secretKey ...[]byte) *Service {
	service := &Service{db: db, provider: provider}
	if len(secretKey) > 0 {
		service.secretKey = secretKey[0]
	}
	return service
}

func NewServiceWithRegistry(
	db *gorm.DB,
	provider ComputeProvider,
	registry *ComputeProviderRegistry,
	secretKey ...[]byte,
) *Service {
	service := NewService(db, provider, secretKey...)
	service.registry = registry
	return service
}

func (s *Service) Region() string {
	if s.provider == nil {
		return ""
	}
	return s.provider.Region()
}

func (s *Service) ProviderCapabilities() ComputeCapabilities {
	provider, ok := s.provider.(interface {
		Capabilities() ComputeCapabilities
	})
	if !ok {
		return ComputeCapabilities{}
	}
	return provider.Capabilities()
}

func (s *Service) Catalog(
	ctx context.Context, userID, tenantID, projectID uint,
) (ComputeCatalog, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return ComputeCatalog{}, ErrForbidden
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return ComputeCatalog{}, err
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return ComputeCatalog{}, err
	}
	catalogProvider, ok := provider.(ComputeCatalogProvider)
	if !ok {
		return ComputeCatalog{}, ErrProviderUnavailable
	}
	catalog, err := catalogProvider.ComputeCatalog(ctx)
	if err != nil {
		return ComputeCatalog{}, err
	}
	if err := s.mapCatalogResources(ctx, provider, tenantID, projectID, &catalog); err != nil {
		return ComputeCatalog{}, err
	}
	catalog.Provisioning = catalog.Capabilities.Compute &&
		catalog.Capabilities.Image && catalog.Capabilities.Network &&
		len(catalog.Images) > 0 && len(catalog.Sizes) > 0 && len(catalog.Networks) > 0
	return catalog, nil
}

func (s *Service) ListSecurityGroups(
	ctx context.Context, userID, tenantID, projectID uint,
) ([]SecurityGroup, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return nil, ErrForbidden
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	groupProvider, ok := provider.(ComputeSecurityGroupProvider)
	if !ok {
		return nil, ErrSecurityGroupUnavailable
	}
	return groupProvider.ListSecurityGroups(ctx)
}

func (s *Service) CreateSecurityGroup(
	ctx context.Context, userID, tenantID, projectID uint, request CreateSecurityGroupRequest,
) (SecurityGroup, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return SecurityGroup{}, ErrForbidden
	}
	request.Name = strings.TrimSpace(request.Name)
	if !instanceNamePattern.MatchString(request.Name) {
		return SecurityGroup{}, fmt.Errorf("%w: security group name is invalid", ErrInvalid)
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, true); err != nil {
		return SecurityGroup{}, err
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return SecurityGroup{}, err
	}
	groupProvider, ok := provider.(ComputeSecurityGroupProvider)
	if !ok {
		return SecurityGroup{}, ErrSecurityGroupUnavailable
	}
	return groupProvider.CreateSecurityGroup(ctx, request.Name)
}

func providerComputeAvailable(provider Provider) bool {
	if provider == nil {
		return false
	}
	computeProvider, ok := provider.(interface {
		ComputeAvailable() bool
	})
	return !ok || computeProvider.ComputeAvailable()
}

func (s *Service) mapCatalogResources(
	ctx context.Context,
	provider Provider,
	tenantID, projectID uint,
	catalog *ComputeCatalog,
) error {
	providerName := strings.TrimSpace(provider.Name())
	if providerName == "" || strings.TrimSpace(catalog.Region) == "" {
		return ErrProviderUnavailable
	}

	type catalogResource struct {
		resourceType string
		providerID   string
		name         string
		setID        func(string)
	}
	resources := []catalogResource{{
		resourceType: "region",
		providerID:   catalog.Region,
		name:         "Primary region",
		setID:        func(id string) { catalog.RegionID = id },
	}}
	for i := range catalog.Images {
		image := &catalog.Images[i]
		resources = append(resources, catalogResource{
			resourceType: "image", providerID: image.ID, name: image.Name,
			setID: func(id string) { image.ID = id },
		})
	}
	sizeCounts := make(map[int]int, len(catalog.Sizes))
	for _, size := range catalog.Sizes {
		sizeCounts[size.VCPUs]++
	}
	for i := range catalog.Sizes {
		size := &catalog.Sizes[i]
		size.Name = fmt.Sprintf("AX Compute %d", size.VCPUs)
		if size.VCPUs < 1 {
			size.Name = "AX Compute Size"
		}
		if sizeCounts[size.VCPUs] > 1 {
			size.Name += fmt.Sprintf(" · %d MiB RAM · %d GiB disk", size.MemoryMB, size.DiskGB)
		}
		resources = append(resources, catalogResource{
			resourceType: "size", providerID: size.ID, name: size.Name,
			setID: func(id string) { size.ID = id },
		})
	}
	for i := range catalog.Networks {
		network := &catalog.Networks[i]
		resources = append(resources, catalogResource{
			resourceType: "network", providerID: network.ID, name: network.Name,
			setID: func(id string) { network.ID = id },
		})
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activeTypes := map[string]struct{}{
			"region": {}, "image": {}, "size": {}, "network": {},
		}
		for resourceType := range activeTypes {
			if err := tx.Model(&models.ProviderCatalogResource{}).
				Where(
					"tenant_id = ? AND project_id = ? AND provider = ? AND resource_type = ?",
					tenantID, projectID, providerName, resourceType,
				).Update("active", false).Error; err != nil {
				return fmt.Errorf("deactivate stale catalog mappings: %w", err)
			}
		}
		for _, resource := range resources {
			if resource.providerID == "" {
				return fmt.Errorf("%w: provider catalog entry is missing an identifier", ErrProviderUnavailable)
			}
			mapping, err := s.upsertCatalogResource(
				tx, tenantID, projectID, providerName,
				resource.resourceType, resource.providerID, resource.name,
			)
			if err != nil {
				return err
			}
			resource.setID(mapping.ID)
			if err := backfillInstanceCatalogID(
				tx, tenantID, projectID, resource.resourceType,
				resource.providerID, mapping.ID,
			); err != nil {
				return fmt.Errorf("update existing resource catalog mapping: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) upsertCatalogResource(
	tx *gorm.DB,
	tenantID, projectID uint,
	provider, resourceType, providerResourceID, name string,
) (models.ProviderCatalogResource, error) {
	id, err := newInstanceID()
	if err != nil {
		return models.ProviderCatalogResource{}, err
	}
	mapping := models.ProviderCatalogResource{
		ID: id, TenantID: tenantID, ProjectID: projectID,
		Provider: provider, ResourceType: resourceType,
		ProviderResourceID: providerResourceID, Name: name, Active: true,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"}, {Name: "project_id"}, {Name: "provider"},
			{Name: "resource_type"}, {Name: "provider_resource_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"name", "active", "updated_at"}),
	}).Create(&mapping).Error; err != nil {
		return models.ProviderCatalogResource{}, fmt.Errorf("save catalog mapping: %w", err)
	}
	var persisted models.ProviderCatalogResource
	if err := tx.Where(
		"tenant_id = ? AND project_id = ? AND provider = ? AND resource_type = ? AND provider_resource_id = ?",
		tenantID, projectID, provider, resourceType, providerResourceID,
	).First(&persisted).Error; err != nil {
		return models.ProviderCatalogResource{}, fmt.Errorf("load catalog mapping: %w", err)
	}
	return persisted, nil
}

func backfillInstanceCatalogID(
	tx *gorm.DB,
	tenantID, projectID uint,
	resourceType, providerResourceID, catalogResourceID string,
) error {
	var providerColumn, resourceColumn string
	switch resourceType {
	case "region":
		providerColumn, resourceColumn = "region", "region_resource_id"
	case "image":
		providerColumn, resourceColumn = "image_id", "image_resource_id"
	case "size":
		providerColumn, resourceColumn = "flavor_id", "size_resource_id"
	case "network":
		providerColumn, resourceColumn = "network_id", "network_resource_id"
	default:
		return fmt.Errorf("unsupported catalog resource type %q", resourceType)
	}
	return tx.Model(&models.CloudInstance{}).
		Where(
			"tenant_id = ? AND project_id = ? AND "+providerColumn+" = ? AND ("+resourceColumn+" = '' OR "+resourceColumn+" IS NULL)",
			tenantID, projectID, providerResourceID,
		).Update(resourceColumn, catalogResourceID).Error
}

func (s *Service) resolveCatalogResource(
	ctx context.Context,
	tenantID, projectID uint,
	provider, resourceType, id string,
) (models.ProviderCatalogResource, error) {
	if !catalogResourceIDPattern.MatchString(id) {
		return models.ProviderCatalogResource{}, fmt.Errorf("%w: invalid catalog resource", ErrInvalid)
	}
	var mapping models.ProviderCatalogResource
	err := s.db.WithContext(ctx).Where(
		"id = ? AND tenant_id = ? AND project_id = ? AND provider = ? AND resource_type = ? AND active = ?",
		id, tenantID, projectID, provider, resourceType, true,
	).First(&mapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ProviderCatalogResource{}, fmt.Errorf("%w: unavailable catalog resource", ErrInvalid)
	}
	if err != nil {
		return models.ProviderCatalogResource{}, err
	}
	return mapping, nil
}

func (s *Service) ListInstances(
	ctx context.Context, userID, tenantID, projectID uint,
) ([]models.CloudInstance, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return nil, ErrForbidden
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !providerComputeAvailable(provider) {
		return nil, ErrProviderUnavailable
	}

	var instances []models.CloudInstance
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ? AND deleted_at IS NULL", tenantID, projectID).
		Order("created_at DESC").Limit(500).Find(&instances).Error; err != nil {
		return nil, err
	}
	for index := range instances {
		if err := s.refreshInstance(ctx, &instances[index], provider); err != nil {
			return nil, err
		}
	}
	result := instances[:0]
	for _, instance := range instances {
		if instance.DeletedAt == nil {
			result = append(result, instance)
		}
	}
	return result, nil
}

func (s *Service) CreateInstance(
	ctx context.Context,
	userID, tenantID, projectID uint,
	idempotencyKey string,
	request CreateInstanceRequest,
) (models.CloudInstance, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return models.CloudInstance{}, ErrForbidden
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, true); err != nil {
		return models.CloudInstance{}, err
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	if !providerComputeAvailable(provider) {
		return models.CloudInstance{}, ErrProviderUnavailable
	}
	projectProvider, ok := provider.(ProjectScopedComputeProvider)
	if !ok {
		return models.CloudInstance{}, ErrProjectProvisioningDisabled
	}
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return models.CloudInstance{}, fmt.Errorf("%w: an Idempotency-Key between 8 and 128 characters is required", ErrInvalid)
	}
	if err := normalizeCreateRequest(&request); err != nil {
		return models.CloudInstance{}, err
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return models.CloudInstance{}, err
	}
	requestHash := sha256.Sum256(requestJSON)
	hash := hex.EncodeToString(requestHash[:])

	var existing models.CloudInstance
	err = s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
		tenantID, projectID, idempotencyKey,
	).First(&existing).Error
	if err == nil {
		if existing.OwnerID != userID || existing.RequestHash != hash {
			return models.CloudInstance{}, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.CloudInstance{}, err
	}
	var named models.CloudInstance
	if err := s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND name = ? AND deleted_at IS NULL",
		tenantID, projectID, request.Name,
	).First(&named).Error; err == nil {
		return models.CloudInstance{}, ErrConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.CloudInstance{}, err
	}

	region, err := s.resolveCatalogResource(ctx, tenantID, projectID, provider.Name(), "region", request.RegionID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	if region.ProviderResourceID != provider.Region() {
		return models.CloudInstance{}, fmt.Errorf("%w: selected region is unavailable", ErrInvalid)
	}
	image, err := s.resolveCatalogResource(ctx, tenantID, projectID, provider.Name(), "image", request.ImageID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	size, err := s.resolveCatalogResource(ctx, tenantID, projectID, provider.Name(), "size", request.SizeID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	network, err := s.resolveCatalogResource(ctx, tenantID, projectID, provider.Name(), "network", request.NetworkID)
	if err != nil {
		return models.CloudInstance{}, err
	}

	id, err := newInstanceID()
	if err != nil {
		return models.CloudInstance{}, err
	}
	instance := models.CloudInstance{
		ID: id, TenantID: tenantID, ProjectID: projectID, OwnerID: userID,
		Name: request.Name, Provider: provider.Name(), Region: region.ProviderResourceID,
		ImageID: image.ProviderResourceID, FlavorID: size.ProviderResourceID, NetworkID: network.ProviderResourceID,
		RegionResourceID: request.RegionID, ImageResourceID: request.ImageID,
		SizeResourceID: request.SizeID, NetworkResourceID: request.NetworkID,
		KeyName: request.KeyName, Status: "PROVISIONING",
		IdempotencyKey: idempotencyKey, RequestHash: hash,
	}
	if err := s.db.WithContext(ctx).Create(&instance).Error; err != nil {
		var concurrent models.CloudInstance
		lookupErr := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
			tenantID, projectID, idempotencyKey,
		).First(&concurrent).Error
		if lookupErr == nil && concurrent.OwnerID == userID && concurrent.RequestHash == hash {
			return concurrent, nil
		}
		if lookupErr == nil {
			return models.CloudInstance{}, ErrConflict
		}
		var duplicateName models.CloudInstance
		nameErr := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND project_id = ? AND name = ? AND deleted_at IS NULL",
			tenantID, projectID, request.Name,
		).First(&duplicateName).Error
		if nameErr == nil {
			return models.CloudInstance{}, ErrConflict
		}
		if !errors.Is(nameErr, gorm.ErrRecordNotFound) {
			return models.CloudInstance{}, nameErr
		}
		return models.CloudInstance{}, err
	}

	operation, err := s.createOperation(ctx, instance, userID, "create")
	if err != nil {
		if updateErr := s.db.WithContext(ctx).Model(&instance).Updates(map[string]any{
			"status": "ERROR", "error_code": "OPERATION_RECORD_FAILED",
		}).Error; updateErr != nil {
			return models.CloudInstance{}, fmt.Errorf("create operation record failed and instance state could not be saved: %w", updateErr)
		}
		return models.CloudInstance{}, fmt.Errorf("create cloud operation record: %w", err)
	}
	providerInstance, err := projectProvider.CreateInstanceForProject(
		ctx, tenantID, projectID, ProviderCreateRequest{
			ID: instance.ID, Name: request.Name, ImageID: image.ProviderResourceID,
			SizeID: size.ProviderResourceID, NetworkID: network.ProviderResourceID, KeyName: request.KeyName,
		})
	if err != nil {
		stateErr := s.db.WithContext(ctx).Model(&instance).Updates(map[string]any{
			"status": "ERROR", "error_code": "PROVISIONING_FAILED",
		}).Error
		operationErr := s.completeOperation(ctx, &operation, "FAILED", "PROVISIONING_FAILED", "failed")
		if stateErr != nil {
			return models.CloudInstance{}, fmt.Errorf("Nova create failed and provisioning state could not be saved: %w", stateErr)
		}
		if operationErr != nil {
			return models.CloudInstance{}, fmt.Errorf("%w: create server; record operation failure: %v", ErrProviderRequest, operationErr)
		}
		return models.CloudInstance{}, fmt.Errorf("%w: create server", ErrProviderRequest)
	}
	if providerInstance.ID == "" {
		stateErr := s.db.WithContext(ctx).Model(&instance).Updates(map[string]any{
			"status": "ERROR", "error_code": "COMPUTE_RESPONSE_INVALID",
		}).Error
		operationErr := s.completeOperation(ctx, &operation, "FAILED", "COMPUTE_RESPONSE_INVALID", "failed")
		if stateErr != nil {
			return models.CloudInstance{}, fmt.Errorf("Nova omitted the server ID and provisioning state could not be saved: %w", stateErr)
		}
		if operationErr != nil {
			return models.CloudInstance{}, fmt.Errorf("%w: Nova returned no server ID; record operation failure: %v", ErrProviderRequest, operationErr)
		}
		return models.CloudInstance{}, fmt.Errorf("%w: Nova returned no server ID", ErrProviderRequest)
	}
	instance.ProviderInstanceID = providerInstance.ID
	applyProviderState(&instance, providerInstance)
	operation.ProviderOperationID = providerInstance.ID
	if err := s.db.WithContext(ctx).Model(&operation).
		Update("provider_operation_id", providerInstance.ID).Error; err != nil {
		return models.CloudInstance{}, err
	}
	if err := s.db.WithContext(ctx).Model(&instance).Updates(map[string]any{
		"provider_instance_id": instance.ProviderInstanceID,
		"status":               instance.Status,
		"provider_status":      instance.ProviderStatus,
		"private_ip":           instance.PrivateIP,
		"public_ip":            instance.PublicIP,
	}).Error; err != nil {
		return models.CloudInstance{}, err
	}
	if instance.Status == "RUNNING" {
		if err := s.completeOperation(ctx, &operation, "SUCCEEDED", "", "succeeded"); err != nil {
			return models.CloudInstance{}, err
		}
	}
	return instance, nil
}

func (s *Service) GetInstance(
	ctx context.Context, userID uint, id string,
) (models.CloudInstance, error) {
	var instance models.CloudInstance
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CloudInstance{}, ErrNotFound
		}
		return models.CloudInstance{}, err
	}
	if err := s.authorizeProject(ctx, userID, instance.TenantID, instance.ProjectID, false); err != nil {
		if errors.Is(err, ErrForbidden) {
			return models.CloudInstance{}, ErrNotFound
		}
		return models.CloudInstance{}, err
	}
	provider, err := s.providerForTenant(ctx, instance.TenantID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	if !providerComputeAvailable(provider) {
		return models.CloudInstance{}, ErrProviderUnavailable
	}
	if err := s.refreshInstance(ctx, &instance, provider); err != nil {
		return models.CloudInstance{}, err
	}
	if instance.DeletedAt != nil || instance.Status == "DELETED" {
		return models.CloudInstance{}, ErrNotFound
	}
	return instance, nil
}

func (s *Service) OperateInstance(
	ctx context.Context, userID uint, id, action string,
) (models.CloudInstance, error) {
	instance, err := s.findAuthorized(ctx, userID, id, true)
	if err != nil {
		return models.CloudInstance{}, err
	}
	if err := s.ensureNoRunningOperation(ctx, instance.ID); err != nil {
		return models.CloudInstance{}, err
	}
	provider, err := s.providerForTenant(ctx, instance.TenantID)
	if err != nil {
		return models.CloudInstance{}, err
	}
	if !providerComputeAvailable(provider) {
		return models.CloudInstance{}, ErrProviderUnavailable
	}
	if instance.ProviderInstanceID == "" {
		return models.CloudInstance{}, fmt.Errorf("%w: instance has no Nova resource", ErrConflict)
	}
	current, err := provider.GetInstance(ctx, instance.ProviderInstanceID)
	if errors.Is(err, ErrProviderNotFound) {
		markDeleted(&instance)
		if saveErr := s.saveInstance(ctx, &instance); saveErr != nil {
			return models.CloudInstance{}, saveErr
		}
		return models.CloudInstance{}, ErrNotFound
	}
	if err != nil {
		if errors.Is(err, ErrProviderUnavailable) {
			return models.CloudInstance{}, ErrProviderUnavailable
		}
		return models.CloudInstance{}, fmt.Errorf("%w: inspect server", ErrProviderRequest)
	}
	applyProviderState(&instance, current)

	switch action {
	case "start":
		if instance.ProviderStatus != "SHUTOFF" && instance.ProviderStatus != "STOPPED" {
			return models.CloudInstance{}, fmt.Errorf("%w: only stopped instances can be started", ErrConflict)
		}
		instance.Status = "STARTING"
	case "stop":
		if instance.ProviderStatus != "ACTIVE" {
			return models.CloudInstance{}, fmt.Errorf("%w: only active instances can be stopped", ErrConflict)
		}
		instance.Status = "STOPPING"
	case "reboot":
		if instance.ProviderStatus != "ACTIVE" {
			return models.CloudInstance{}, fmt.Errorf("%w: only active instances can be rebooted", ErrConflict)
		}
		instance.Status = "REBOOTING"
	default:
		return models.CloudInstance{}, fmt.Errorf("%w: unsupported instance action", ErrInvalid)
	}
	operation, err := s.createOperation(ctx, instance, userID, action)
	if err != nil {
		return models.CloudInstance{}, fmt.Errorf("create cloud operation record: %w", err)
	}
	switch action {
	case "start":
		err = provider.StartInstance(ctx, instance.ProviderInstanceID)
	case "stop":
		err = provider.StopInstance(ctx, instance.ProviderInstanceID)
	case "reboot":
		err = provider.RebootInstance(ctx, instance.ProviderInstanceID)
	}
	if err != nil {
		if operationErr := s.completeOperation(ctx, &operation, "FAILED", "COMPUTE_REQUEST_FAILED", "failed"); operationErr != nil {
			return models.CloudInstance{}, fmt.Errorf("%w: %s server; record operation failure: %v", ErrProviderRequest, action, operationErr)
		}
		if errors.Is(err, ErrProviderUnavailable) {
			return models.CloudInstance{}, ErrProviderUnavailable
		}
		return models.CloudInstance{}, fmt.Errorf("%w: %s server", ErrProviderRequest, action)
	}
	if err := s.saveInstance(ctx, &instance); err != nil {
		return models.CloudInstance{}, err
	}
	return instance, nil
}

func (s *Service) DeleteInstance(ctx context.Context, userID uint, id string) error {
	instance, err := s.findAuthorized(ctx, userID, id, true)
	if err != nil {
		return err
	}
	if instance.Status == "DELETING" {
		return nil
	}
	if err := s.ensureNoRunningOperation(ctx, instance.ID); err != nil {
		return err
	}
	provider, err := s.providerForTenant(ctx, instance.TenantID)
	if err != nil {
		return err
	}
	if !providerComputeAvailable(provider) {
		return ErrProviderUnavailable
	}
	if instance.ProviderInstanceID != "" {
		operation, err := s.createOperation(ctx, instance, userID, "delete")
		if err != nil {
			return fmt.Errorf("create cloud operation record: %w", err)
		}
		if err := provider.DeleteInstance(ctx, instance.ProviderInstanceID); err != nil {
			if operationErr := s.completeOperation(ctx, &operation, "FAILED", "COMPUTE_REQUEST_FAILED", "failed"); operationErr != nil {
				return fmt.Errorf("%w: delete server; record operation failure: %v", ErrProviderRequest, operationErr)
			}
			if errors.Is(err, ErrProviderUnavailable) {
				return ErrProviderUnavailable
			}
			return fmt.Errorf("%w: delete server", ErrProviderRequest)
		}
		instance.Status = "DELETING"
		return s.saveInstance(ctx, &instance)
	}
	operation, err := s.createOperation(ctx, instance, userID, "delete")
	if err != nil {
		return fmt.Errorf("create cloud operation record: %w", err)
	}
	markDeleted(&instance)
	if err := s.saveInstance(ctx, &instance); err != nil {
		return err
	}
	return s.completeOperation(ctx, &operation, "SUCCEEDED", "", "succeeded")
}

func (s *Service) findAuthorized(
	ctx context.Context, userID uint, id string, write bool,
) (models.CloudInstance, error) {
	if userID == 0 {
		return models.CloudInstance{}, ErrForbidden
	}
	var instance models.CloudInstance
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CloudInstance{}, ErrNotFound
		}
		return models.CloudInstance{}, err
	}
	if err := s.authorizeProject(ctx, userID, instance.TenantID, instance.ProjectID, write); err != nil {
		if errors.Is(err, ErrForbidden) {
			return models.CloudInstance{}, ErrNotFound
		}
		return models.CloudInstance{}, err
	}
	return instance, nil
}

func (s *Service) refreshInstance(
	ctx context.Context, instance *models.CloudInstance, provider Provider,
) error {
	if instance.ProviderInstanceID == "" || instance.Status == "ERROR" {
		return nil
	}
	providerInstance, err := provider.GetInstance(ctx, instance.ProviderInstanceID)
	if errors.Is(err, ErrProviderNotFound) {
		if instance.Status == "DELETING" {
			markDeleted(instance)
		} else {
			instance.Status = "ERROR"
			instance.ErrorCode = "RESOURCE_NOT_FOUND"
		}
		if saveErr := s.db.WithContext(ctx).Model(instance).Updates(map[string]any{
			"status": instance.Status, "error_code": instance.ErrorCode, "deleted_at": instance.DeletedAt,
		}).Error; saveErr != nil {
			return saveErr
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: inspect server", ErrProviderRequest)
	}
	applyProviderState(instance, providerInstance)
	if instance.Status == "DELETED" {
		markDeleted(instance)
	}
	return s.saveInstance(ctx, instance)
}

func (s *Service) saveInstance(ctx context.Context, instance *models.CloudInstance) error {
	return s.db.WithContext(ctx).Model(instance).Updates(map[string]any{
		"status":          instance.Status,
		"provider_status": instance.ProviderStatus,
		"private_ip":      instance.PrivateIP,
		"public_ip":       instance.PublicIP,
		"error_code":      instance.ErrorCode,
		"deleted_at":      instance.DeletedAt,
	}).Error
}

func (s *Service) authorizeProject(
	ctx context.Context, userID, tenantID, projectID uint, write bool,
) error {
	var project models.Project
	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", projectID, tenantID).
		First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	isAdmin, err := authorization.IsPlatformSuperAdmin(ctx, s.db, userID)
	if err != nil {
		return err
	}
	if isAdmin {
		return nil
	}

	var membership struct {
		TenantRole  string
		ProjectRole string
	}
	err = s.db.WithContext(ctx).Table("projects").
		Select("tenant_memberships.role AS tenant_role, project_memberships.role AS project_role").
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = projects.tenant_id AND tenant_memberships.user_id = ?", userID).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id AND project_memberships.user_id = ?", userID).
		Where("projects.id = ? AND projects.tenant_id = ?", projectID, tenantID).
		Take(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if write && membership.ProjectRole != "owner" && membership.ProjectRole != "admin" &&
		membership.ProjectRole != "deployer" {
		return ErrForbidden
	}
	return nil
}

func normalizeCreateRequest(request *CreateInstanceRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	request.RegionID = strings.TrimSpace(request.RegionID)
	request.ImageID = strings.TrimSpace(request.ImageID)
	request.SizeID = strings.TrimSpace(request.SizeID)
	request.NetworkID = strings.TrimSpace(request.NetworkID)
	request.KeyName = strings.TrimSpace(request.KeyName)
	if !instanceNamePattern.MatchString(request.Name) {
		return fmt.Errorf("%w: name must use 1-63 letters, numbers, dots, underscores, or hyphens", ErrInvalid)
	}
	if !catalogResourceIDPattern.MatchString(request.RegionID) ||
		!catalogResourceIDPattern.MatchString(request.ImageID) ||
		!catalogResourceIDPattern.MatchString(request.SizeID) ||
		!catalogResourceIDPattern.MatchString(request.NetworkID) {
		return fmt.Errorf("%w: region, image, size, and network must be selected from the current catalog", ErrInvalid)
	}
	if len(request.KeyName) > 255 {
		return fmt.Errorf("%w: key name is too long", ErrInvalid)
	}
	return nil
}

func applyProviderState(instance *models.CloudInstance, providerInstance ProviderInstance) {
	if providerInstance.Status != "" {
		instance.ProviderStatus = strings.ToUpper(providerInstance.Status)
		instance.Status = normalizeProviderStatus(instance.ProviderStatus)
	}
	instance.PrivateIP = providerInstance.PrivateIP
	instance.PublicIP = providerInstance.PublicIP
}

func normalizeProviderStatus(status string) string {
	switch strings.ToUpper(status) {
	case "ACTIVE":
		return "RUNNING"
	case "SHUTOFF", "STOPPED", "SHELVED", "SHELVED_OFFLOADED":
		return "STOPPED"
	case "BUILD", "REBUILD", "REBUILDING", "RESCUE", "UNRESCUE", "SUSPENDED", "PAUSED":
		return "PROVISIONING"
	case "REBOOT", "HARD_REBOOT":
		return "REBOOTING"
	case "MIGRATING", "RESIZE", "RESIZE_MIGRATING", "RESIZE_PREP",
		"RESIZE_VERIFY", "RESIZE_REVERT", "VERIFY_RESIZE", "REVERT_RESIZE":
		return "RESIZING"
	case "DELETING", "SOFT_DELETED":
		return "DELETING"
	case "DELETED":
		return "DELETED"
	case "ERROR":
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func newInstanceID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate instance ID: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func markDeleted(instance *models.CloudInstance) {
	now := time.Now().UTC()
	instance.Status = "DELETED"
	instance.DeletedAt = &now
}
