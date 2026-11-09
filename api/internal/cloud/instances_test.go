package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"atonixcorp/api/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeProvider struct {
	region        string
	instance      ProviderInstance
	catalog       ComputeCatalog
	createRequest ProviderCreateRequest
	createCalls   int
	stopCalls     int
	deleteCalls   int
	createErr     error
	getErr        error
}

type fakeSecurityGroupProvider struct {
	*fakeProvider
	groups      []SecurityGroup
	createCalls int
}

func (p *fakeSecurityGroupProvider) ListSecurityGroups(context.Context) ([]SecurityGroup, error) {
	return p.groups, nil
}

func (p *fakeSecurityGroupProvider) CreateSecurityGroup(
	_ context.Context, name string,
) (SecurityGroup, error) {
	p.createCalls++
	group := SecurityGroup{
		ID: "provider-group-id", Name: name, ProjectID: "openstack-project-id",
		Rules: []SecurityGroupRule{{
			ID: "provider-rule-id", Direction: "ingress", Ethertype: "IPv4",
			Protocol: "tcp", RemoteIPPrefix: "192.0.2.0/24",
		}},
	}
	p.groups = append(p.groups, group)
	return group, nil
}

func (p *fakeProvider) Name() string   { return "openstack" }
func (p *fakeProvider) Region() string { return p.region }

func (p *fakeProvider) ComputeCatalog(context.Context) (ComputeCatalog, error) {
	return p.catalog, nil
}

func (p *fakeProvider) CreateInstanceForProject(
	ctx context.Context, tenantID, projectID uint, request ProviderCreateRequest,
) (ProviderInstance, error) {
	if tenantID == 0 || projectID == 0 {
		return ProviderInstance{}, ErrProjectProvisioningDisabled
	}
	return p.CreateInstance(ctx, request)
}

func (p *fakeProvider) CreateInstance(_ context.Context, request ProviderCreateRequest) (ProviderInstance, error) {
	p.createCalls++
	p.createRequest = request
	if p.createErr != nil {
		return ProviderInstance{}, p.createErr
	}
	p.instance = ProviderInstance{
		ID: "nova-" + request.ID, Status: "BUILD", PrivateIP: "10.0.0.10",
	}
	return p.instance, nil
}

func (p *fakeProvider) GetInstance(context.Context, string) (ProviderInstance, error) {
	return p.instance, p.getErr
}

func (p *fakeProvider) StartInstance(context.Context, string) error { return nil }

func (p *fakeProvider) StopInstance(context.Context, string) error {
	p.stopCalls++
	return nil
}

func (p *fakeProvider) RebootInstance(context.Context, string) error { return nil }

func (p *fakeProvider) DeleteInstance(context.Context, string) error {
	p.deleteCalls++
	return nil
}

func TestCreateInstanceIsIdempotentAndStoresNovaReference(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	request := validCreateRequest()

	first, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0001", request)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	second, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0001", request)
	if err != nil {
		t.Fatalf("idempotent CreateInstance() error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent request returned different IDs: %q, %q", first.ID, second.ID)
	}
	if first.ProviderInstanceID == "" || first.Provider != "openstack" ||
		first.Status != "PROVISIONING" || first.ProviderStatus != "BUILD" {
		t.Fatalf("unexpected persisted provider instance: %+v", first)
	}
	if first.RegionResourceID != testRegionResourceID ||
		first.ImageResourceID != testImageResourceID ||
		first.SizeResourceID != testSizeResourceID ||
		first.NetworkResourceID != testNetworkResourceID {
		t.Fatalf("instance response did not retain AtonixCorp catalog IDs: %+v", first)
	}
	if provider.createRequest.ImageID != "provider-image-id" ||
		provider.createRequest.SizeID != "provider-size-id" ||
		provider.createRequest.NetworkID != "provider-network-id" {
		t.Fatalf("provider received untranslated catalog IDs: %+v", provider.createRequest)
	}
	var persisted models.CloudInstance
	if err := db.First(&persisted, "id = ?", first.ID).Error; err != nil {
		t.Fatalf("load persisted instance: %v", err)
	}
	if persisted.ProviderStatus != "BUILD" {
		t.Fatalf("persisted provider status = %q, want BUILD", persisted.ProviderStatus)
	}
	if provider.createCalls != 1 {
		t.Fatalf("Nova create called %d times, want 1", provider.createCalls)
	}
	request.ImageID = "00000000000000000000000000000005"
	if _, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0001", request); !errors.Is(err, ErrConflict) {
		t.Fatalf("reused key with changed request returned %v, want ErrConflict", err)
	}
}

func TestCatalogReturnsProjectScopedProviderResourcesAndRequiresMembership(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{
		region: "RegionOne",
		catalog: ComputeCatalog{
			Region:       "RegionOne",
			Capabilities: ComputeCapabilities{Compute: true, Image: true, Network: true},
			Images:       []CatalogImage{{ID: "image-real", Name: "Ubuntu", Status: "ACTIVE"}},
			Sizes:        []ComputeSize{{ID: "flavor-real", Name: "small", VCPUs: 2, MemoryMB: 2048}},
			Networks:     []CatalogNetwork{{ID: "network-real", Name: "private", Status: "ACTIVE", AdminStateUp: true}},
		},
	}
	service := NewService(db, provider)
	catalog, err := service.Catalog(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if !catalog.Provisioning || catalog.RegionID == "RegionOne" ||
		catalog.Images[0].ID == "image-real" ||
		catalog.Sizes[0].ID == "flavor-real" ||
		catalog.Sizes[0].Name != "AX Compute 2" ||
		catalog.Sizes[0].MemoryMB != 2048 ||
		catalog.Networks[0].ID == "network-real" {
		t.Fatalf("Catalog() = %+v, want mapped catalog IDs and enabled provisioning", catalog)
	}
	secondCatalog, err := service.Catalog(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatalf("second Catalog() error = %v", err)
	}
	if catalog.RegionID != secondCatalog.RegionID ||
		catalog.Images[0].ID != secondCatalog.Images[0].ID ||
		catalog.Sizes[0].ID != secondCatalog.Sizes[0].ID ||
		catalog.Networks[0].ID != secondCatalog.Networks[0].ID {
		t.Fatalf("catalog mappings were not stable across refreshes: first=%+v second=%+v", catalog, secondCatalog)
	}
	if _, err := service.Catalog(context.Background(), 2, 1, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Catalog() for non-member returned %v, want ErrForbidden", err)
	}
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal customer catalog: %v", err)
	}
	for _, providerID := range []string{"RegionOne", "image-real", "flavor-real", "network-real", "small"} {
		if strings.Contains(string(encoded), providerID) {
			t.Errorf("customer catalog exposed provider ID %q: %s", providerID, encoded)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("decode customer catalog: %v", err)
	}
	if _, ok := response["sizes"]; !ok {
		t.Errorf("customer catalog omitted sizes: %s", encoded)
	}
	if _, ok := response["flavors"]; ok {
		t.Errorf("customer catalog exposed provider terminology: %s", encoded)
	}
}

func TestCatalogRefreshDeactivatesRemovedMappings(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{
		region: "RegionOne",
		catalog: ComputeCatalog{
			Region:       "RegionOne",
			Capabilities: ComputeCapabilities{Compute: true, Image: true, Network: true},
			Images:       []CatalogImage{{ID: "image-real", Name: "Ubuntu", Status: "ACTIVE"}},
			Sizes:        []ComputeSize{{ID: "flavor-real", Name: "small", VCPUs: 2, MemoryMB: 2048}},
			Networks:     []CatalogNetwork{{ID: "network-real", Name: "private", Status: "ACTIVE", AdminStateUp: true}},
		},
	}
	service := NewService(db, provider)
	catalog, err := service.Catalog(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	provider.catalog.Networks = []CatalogNetwork{}
	if _, err := service.Catalog(context.Background(), 1, 1, 1); err != nil {
		t.Fatalf("Catalog() after network removal error = %v", err)
	}
	request := CreateInstanceRequest{
		TenantID: 1, ProjectID: 1, Name: "stale-network-vm",
		RegionID: catalog.RegionID, ImageID: catalog.Images[0].ID,
		SizeID: catalog.Sizes[0].ID, NetworkID: catalog.Networks[0].ID,
	}
	if _, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0015", request,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateInstance() with a removed catalog mapping returned %v, want ErrInvalid", err)
	}
	if provider.createCalls != 0 {
		t.Fatalf("provider was called %d times for a removed catalog mapping, want 0", provider.createCalls)
	}
}

func TestSecurityGroupsAreProjectScopedAndProviderBacked(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeSecurityGroupProvider{fakeProvider: &fakeProvider{region: "RegionOne"}}
	service := NewService(db, provider)

	group, err := service.CreateSecurityGroup(
		context.Background(), 1, 1, 1, CreateSecurityGroupRequest{Name: "safe-test-group"},
	)
	if err != nil {
		t.Fatalf("CreateSecurityGroup() error = %v", err)
	}
	if group.ID != "provider-group-id" || group.ProjectID != "openstack-project-id" ||
		len(group.Rules) != 1 || group.Rules[0].RemoteIPPrefix != "192.0.2.0/24" {
		t.Fatalf("CreateSecurityGroup() = %+v, want provider-returned group and safe test rule", group)
	}

	groups, err := service.ListSecurityGroups(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatalf("ListSecurityGroups() error = %v", err)
	}
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Fatalf("ListSecurityGroups() = %+v, want provider-created group", groups)
	}
	if _, err := service.ListSecurityGroups(context.Background(), 2, 1, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListSecurityGroups() for non-member returned %v, want ErrForbidden", err)
	}
	if _, err := service.CreateSecurityGroup(
		context.Background(), 1, 1, 1, CreateSecurityGroupRequest{Name: "unsafe name"},
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateSecurityGroup() with invalid name returned %v, want ErrInvalid", err)
	}
	if provider.createCalls != 1 {
		t.Fatalf("provider created %d groups, want exactly 1", provider.createCalls)
	}
}

func TestCreateInstancePersistsAndReconcilesCloudOperation(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	ctx := WithRequestID(context.Background(), "request-123")
	instance, err := service.CreateInstance(ctx, 1, 1, 1, "request-key-0010", validCreateRequest())
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	var operation models.CloudOperation
	if err := db.Where("resource_id = ? AND operation = ?", instance.ID, "create").
		First(&operation).Error; err != nil {
		t.Fatalf("load create operation: %v", err)
	}
	if operation.Status != "RUNNING" || operation.RequestID != "request-123" {
		t.Fatalf("create operation = %+v, want RUNNING with request ID", operation)
	}

	provider.instance.Status = "ACTIVE"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("ReconcilePendingOperations() error = %v", err)
	}
	if err := db.First(&operation, "id = ?", operation.ID).Error; err != nil {
		t.Fatalf("reload operation: %v", err)
	}
	if operation.Status != "SUCCEEDED" || operation.CompletedAt == nil {
		t.Fatalf("reconciled operation = %+v, want completed SUCCEEDED", operation)
	}
	var auditCount int64
	if err := db.Model(&models.CloudAuditEvent{}).
		Where("operation_id = ?", operation.ID).Count(&auditCount).Error; err != nil {
		t.Fatalf("count operation audit records: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("audit record count = %d, want pending and succeeded records", auditCount)
	}
}

func TestCloudOperationAccessUsesProjectMembership(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	instance, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0011", validCreateRequest(),
	)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	var operation models.CloudOperation
	if err := db.Where("resource_id = ?", instance.ID).First(&operation).Error; err != nil {
		t.Fatalf("load operation: %v", err)
	}
	if _, err := service.GetOperation(context.Background(), 2, operation.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOperation() for non-member returned %v, want ErrNotFound", err)
	}
	operations, err := service.ListOperations(context.Background(), 1, 1, 1)
	if err != nil {
		t.Fatalf("ListOperations() error = %v", err)
	}
	if len(operations) != 1 || operations[0].ID != operation.ID {
		t.Fatalf("ListOperations() = %+v, want operation %s", operations, operation.ID)
	}
}

func TestRebootOperationWaitsForProviderTransitionAndReturnToActive(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	instance, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0012", validCreateRequest(),
	)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	provider.instance.Status = "ACTIVE"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("reconcile create operation: %v", err)
	}
	if _, err := service.OperateInstance(context.Background(), 1, instance.ID, "reboot"); err != nil {
		t.Fatalf("OperateInstance(reboot) error = %v", err)
	}

	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("ReconcilePendingOperations() error = %v", err)
	}
	var operation models.CloudOperation
	if err := db.Where("resource_id = ? AND operation = ?", instance.ID, "reboot").
		First(&operation).Error; err != nil {
		t.Fatalf("load reboot operation: %v", err)
	}
	if operation.Status != "RUNNING" {
		t.Fatalf("reboot operation completed before Nova reported a reboot: %+v", operation)
	}

	provider.instance.Status = "REBOOT"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("reconcile reboot transition: %v", err)
	}
	if err := db.First(&operation, "id = ?", operation.ID).Error; err != nil {
		t.Fatalf("reload reboot operation: %v", err)
	}
	if !operation.ProviderTransitionObserved || operation.Status != "RUNNING" {
		t.Fatalf("reboot transition state = %+v, want observed and still running", operation)
	}

	provider.instance.Status = "ACTIVE"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("reconcile reboot completion: %v", err)
	}
	if err := db.First(&operation, "id = ?", operation.ID).Error; err != nil {
		t.Fatalf("reload completed reboot operation: %v", err)
	}
	if operation.Status != "SUCCEEDED" {
		t.Fatalf("reboot operation = %+v, want SUCCEEDED after returning ACTIVE", operation)
	}
}

func TestCreateInstanceRequiresProjectScopedProvider(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, unscopedProvider{Provider: provider})

	if _, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0007", validCreateRequest(),
	); !errors.Is(err, ErrProjectProvisioningDisabled) {
		t.Fatalf("CreateInstance() without a project-scoped provider returned %v, want ErrProjectProvisioningDisabled", err)
	}
	var count int64
	if err := db.Model(&models.CloudInstance{}).Count(&count).Error; err != nil {
		t.Fatalf("count instances: %v", err)
	}
	if count != 0 {
		t.Fatalf("unmapped create persisted %d instance records, want 0", count)
	}
}

func TestCreateInstanceRejectsProviderAndCrossProjectCatalogIDs(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	request := validCreateRequest()
	request.ImageID = "provider-image-id"
	if _, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0013", request); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateInstance() with provider image ID returned %v, want ErrInvalid", err)
	}

	if err := db.Create(&models.Project{ID: 2, TenantID: 1, Name: "Other", Slug: "other"}).Error; err != nil {
		t.Fatalf("create second project: %v", err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 2, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatalf("create second project membership: %v", err)
	}
	request = validCreateRequest()
	request.ProjectID = 2
	if _, err := service.CreateInstance(context.Background(), 1, 1, 2, "request-key-0014", request); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateInstance() with another project's catalog IDs returned %v, want ErrInvalid", err)
	}
	if provider.createCalls != 0 {
		t.Fatalf("provider was called %d times for invalid catalog IDs, want 0", provider.createCalls)
	}
}

func TestInstanceLookupRequiresProjectMembership(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	instance, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0002", validCreateRequest(),
	)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if _, err := service.GetInstance(context.Background(), 2, instance.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetInstance() for non-member returned %v, want ErrNotFound", err)
	}
	if _, err := service.ListInstances(context.Background(), 2, 1, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListInstances() for non-member returned %v, want ErrForbidden", err)
	}
}

func TestOperateInstanceRequiresActiveServerAndProjectRole(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	instance, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0003", validCreateRequest(),
	)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	provider.instance.Status = "ACTIVE"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("reconcile create operation: %v", err)
	}
	result, err := service.OperateInstance(context.Background(), 1, instance.ID, "stop")
	if err != nil {
		t.Fatalf("OperateInstance(stop) error = %v", err)
	}
	if result.Status != "STOPPING" || result.ProviderStatus != "ACTIVE" || provider.stopCalls != 1 {
		t.Fatalf("stop result = %+v, stop calls = %d", result, provider.stopCalls)
	}

	provider.instance.Status = "BUILD"
	if _, err := service.OperateInstance(context.Background(), 1, instance.ID, "stop"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stopping BUILD instance returned %v, want ErrConflict", err)
	}
}

func TestNormalizeProviderStatus(t *testing.T) {
	tests := map[string]string{
		"ACTIVE": "RUNNING", "SHUTOFF": "STOPPED", "BUILD": "PROVISIONING",
		"REBOOT": "REBOOTING", "MIGRATING": "RESIZING", "VERIFY_RESIZE": "RESIZING",
		"SOFT_DELETED": "DELETING", "DELETED": "DELETED", "ERROR": "ERROR",
		"provider-new-state": "UNKNOWN",
	}
	for providerStatus, want := range tests {
		if got := normalizeProviderStatus(providerStatus); got != want {
			t.Errorf("normalizeProviderStatus(%q) = %q, want %q", providerStatus, got, want)
		}
	}
}

func TestOperateMissingNovaInstanceMarksRecordDeleted(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	service := NewService(db, provider)
	instance, err := service.CreateInstance(
		context.Background(), 1, 1, 1, "request-key-0006", validCreateRequest(),
	)
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	provider.instance.Status = "ACTIVE"
	if err := service.ReconcilePendingOperations(context.Background()); err != nil {
		t.Fatalf("reconcile create operation: %v", err)
	}
	provider.getErr = ErrProviderNotFound
	if _, err := service.OperateInstance(context.Background(), 1, instance.ID, "stop"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("OperateInstance() for missing Nova resource returned %v, want ErrNotFound", err)
	}
	var persisted models.CloudInstance
	if err := db.Unscoped().First(&persisted, "id = ?", instance.ID).Error; err != nil {
		t.Fatalf("load soft-deleted instance: %v", err)
	}
	if persisted.DeletedAt == nil || persisted.Status != "DELETED" {
		t.Fatalf("missing Nova resource was not marked deleted: %+v", persisted)
	}
}

func TestDeleteFailedProvisioningReleasesInstanceName(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne", createErr: errors.New("provider detail")}
	service := NewService(db, provider)
	request := validCreateRequest()
	if _, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0004", request); !errors.Is(err, ErrProviderRequest) {
		t.Fatalf("CreateInstance() error = %v, want ErrProviderRequest", err)
	}
	var failed models.CloudInstance
	if err := db.Where("idempotency_key = ?", "request-key-0004").First(&failed).Error; err != nil {
		t.Fatalf("load failed instance: %v", err)
	}
	if err := service.DeleteInstance(context.Background(), 1, failed.ID); err != nil {
		t.Fatalf("DeleteInstance() error = %v", err)
	}
	provider.createErr = nil
	if _, err := service.CreateInstance(context.Background(), 1, 1, 1, "request-key-0005", request); err != nil {
		t.Fatalf("CreateInstance() after deleting failed resource error = %v", err)
	}
}

func TestCloudInstanceJSONHidesProviderDetailsAndKeepsIdempotencyPrivate(t *testing.T) {
	encoded, err := json.Marshal(models.CloudInstance{
		ID: "atonixcorp-id", Provider: "openstack", ProviderInstanceID: "nova-id",
		ProviderStatus: "BUILD", PrivateIP: "10.0.0.10",
		Region: "RegionOne", ImageID: "provider-image-id",
		FlavorID: "provider-size-id", NetworkID: "provider-network-id",
		RegionResourceID: testRegionResourceID, ImageResourceID: testImageResourceID,
		SizeResourceID: testSizeResourceID, NetworkResourceID: testNetworkResourceID,
		IdempotencyKey: "private-key", RequestHash: "private-hash",
	})
	if err != nil {
		t.Fatalf("marshal CloudInstance: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("decode CloudInstance JSON: %v", err)
	}
	for key, expected := range map[string]any{
		"private_ip": "10.0.0.10",
		"region_id":  testRegionResourceID,
		"image_id":   testImageResourceID,
		"size_id":    testSizeResourceID,
		"network_id": testNetworkResourceID,
	} {
		if response[key] != expected {
			t.Errorf("CloudInstance JSON %s = %v, want %v", key, response[key], expected)
		}
	}
	for _, privateField := range []string{
		"provider", "provider_id", "provider_status", "region", "flavor_id",
		"idempotency_key", "request_hash",
	} {
		if _, exists := response[privateField]; exists {
			t.Errorf("CloudInstance JSON exposed private field %q: %s", privateField, encoded)
		}
	}
}

func TestCloudOperationJSONDoesNotExposeProviderOperationIDOrActor(t *testing.T) {
	encoded, err := json.Marshal(models.CloudOperation{
		ID: "atonixcorp-operation", ProviderOperationID: "nova-server-id",
		ActorID: 73, ProviderStatusAtStart: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("marshal CloudOperation: %v", err)
	}
	for _, privateValue := range []string{"nova-server-id", "73", "ACTIVE"} {
		if strings.Contains(string(encoded), privateValue) {
			t.Errorf("CloudOperation JSON exposed %q: %s", privateValue, encoded)
		}
	}
}

func newCloudTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Tenant{}, &models.TenantMembership{}, &models.Project{},
		&models.ProjectMembership{}, &models.CloudAccount{}, &models.CloudInstance{},
		&models.ProviderCatalogResource{}, &models.CloudOperation{}, &models.CloudAuditEvent{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if err := db.Create(&models.Tenant{ID: 1, Name: "Tenant", Slug: "tenant"}).Error; err != nil {
		t.Fatalf("create test tenant: %v", err)
	}
	if err := db.Create(&models.Project{ID: 1, TenantID: 1, Name: "Project", Slug: "project"}).Error; err != nil {
		t.Fatalf("create test project: %v", err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatalf("create test tenant membership: %v", err)
	}
	if err := db.Create(&models.ProjectMembership{ProjectID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatalf("create test project membership: %v", err)
	}
	mappings := []models.ProviderCatalogResource{
		{
			ID: testRegionResourceID, TenantID: 1, ProjectID: 1, Provider: "openstack",
			ResourceType: "region", ProviderResourceID: "RegionOne", Active: true,
		},
		{
			ID: testImageResourceID, TenantID: 1, ProjectID: 1, Provider: "openstack",
			ResourceType: "image", ProviderResourceID: "provider-image-id", Active: true,
		},
		{
			ID: testSizeResourceID, TenantID: 1, ProjectID: 1, Provider: "openstack",
			ResourceType: "size", ProviderResourceID: "provider-size-id", Active: true,
		},
		{
			ID: testNetworkResourceID, TenantID: 1, ProjectID: 1, Provider: "openstack",
			ResourceType: "network", ProviderResourceID: "provider-network-id", Active: true,
		},
	}
	if err := db.Create(&mappings).Error; err != nil {
		t.Fatalf("seed provider catalog mappings: %v", err)
	}
	return db
}

func TestPlatformSuperAdminCanAuthorizeAcrossTenantAndProject(t *testing.T) {
	db := newCloudTestDB(t)
	if err := db.Create(&models.User{
		ID: 99, FullName: "Platform Admin", Email: "platform@example.invalid",
		PlatformRole: models.PlatformRoleSuperAdmin,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Tenant{ID: 2, Name: "Other tenant", Slug: "other-tenant"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Project{ID: 2, TenantID: 2, Name: "Other project", Slug: "other-project"}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewService(db, nil)
	if err := service.authorizeProject(context.Background(), 99, 2, 2, true); err != nil {
		t.Fatalf("super admin cross-tenant write authorization: %v", err)
	}
	if err := service.authorizeProject(context.Background(), 99, 1, 2, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("mismatched tenant/project scope error = %v, want ErrForbidden", err)
	}
	if err := service.authorizeProject(context.Background(), 1, 2, 2, false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ordinary user cross-tenant access error = %v, want ErrForbidden", err)
	}
}

type unscopedProvider struct {
	Provider
}

func validCreateRequest() CreateInstanceRequest {
	return CreateInstanceRequest{
		TenantID: 1, ProjectID: 1, Name: "test-vm", RegionID: testRegionResourceID,
		ImageID: testImageResourceID, SizeID: testSizeResourceID, NetworkID: testNetworkResourceID,
	}
}

const (
	testRegionResourceID  = "00000000000000000000000000000001"
	testImageResourceID   = "00000000000000000000000000000002"
	testSizeResourceID    = "00000000000000000000000000000003"
	testNetworkResourceID = "00000000000000000000000000000004"
)
