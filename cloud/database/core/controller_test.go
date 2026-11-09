package core_test

import (
	. "atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/openstack"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type memoryStore struct {
	instances   map[string]DBInstance
	operations  map[string]DatabaseOperation
	idempotency map[string]memoryIdempotency
	failUpdate  bool
}

type memoryIdempotency struct {
	hash        string
	instanceID  string
	operationID string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		instances:   make(map[string]DBInstance),
		operations:  make(map[string]DatabaseOperation),
		idempotency: make(map[string]memoryIdempotency),
	}
}

func (s *memoryStore) CreateInstance(instance *DBInstance) error {
	if _, exists := s.instances[instance.ID]; exists {
		return errors.New("instance already exists")
	}
	s.instances[instance.ID] = *instance
	return nil
}

func (s *memoryStore) UpdateInstance(instance *DBInstance) error {
	if s.failUpdate {
		s.failUpdate = false
		return errors.New("temporary metadata-store failure")
	}
	if _, exists := s.instances[instance.ID]; !exists {
		return errors.New("instance not found")
	}
	s.instances[instance.ID] = *instance
	return nil
}

func (s *memoryStore) GetInstance(id string) (*DBInstance, error) {
	instance, exists := s.instances[id]
	if !exists {
		return nil, errors.New("instance not found")
	}
	return &instance, nil
}

func (s *memoryStore) ListInstances(tenantID, projectID string) ([]DBInstance, error) {
	var result []DBInstance
	for _, instance := range s.instances {
		if instance.TenantID == tenantID && instance.ProjectID == projectID {
			result = append(result, instance)
		}
	}
	return result, nil
}

func (s *memoryStore) ListInstancesByStatus(status string, _ int) ([]DBInstance, error) {
	var result []DBInstance
	for _, instance := range s.instances {
		if instance.Status == status {
			result = append(result, instance)
		}
	}
	return result, nil
}

func (*memoryStore) CanAccessProject(uint, string, string) (bool, error) {
	return true, nil
}

func (s *memoryStore) CreateInstanceOperation(
	instance *DBInstance, operation *DatabaseOperation, key, requestHash string,
) (bool, error) {
	scopedKey := instance.TenantID + "/" + instance.ProjectID + "/" + key
	if key != "" {
		if previous, exists := s.idempotency[scopedKey]; exists {
			if previous.hash != requestHash {
				return false, ErrIdempotencyConflict
			}
			instance.ID = previous.instanceID
			operation.ID = previous.operationID
			return true, nil
		}
	}
	if err := s.CreateInstance(instance); err != nil {
		return false, err
	}
	s.operations[operation.ID] = *operation
	if key != "" {
		s.idempotency[scopedKey] = memoryIdempotency{
			hash: requestHash, instanceID: instance.ID, operationID: operation.ID,
		}
	}
	return false, nil
}

func (s *memoryStore) GetOperation(id string) (*DatabaseOperation, error) {
	operation, exists := s.operations[id]
	if !exists {
		return nil, errors.New("operation not found")
	}
	return &operation, nil
}

func (s *memoryStore) GetOperationForDatabase(databaseID string) (*DatabaseOperation, error) {
	for _, operation := range s.operations {
		if operation.DatabaseID == databaseID {
			return &operation, nil
		}
	}
	return nil, errors.New("operation not found")
}

func (s *memoryStore) UpdateOperation(operation *DatabaseOperation) error {
	if _, exists := s.operations[operation.ID]; !exists {
		return errors.New("operation not found")
	}
	s.operations[operation.ID] = *operation
	return nil
}

type fakeOpenStack struct {
	serverStatus  string
	serverIP      string
	volumeStatus  string
	volumeLinked  bool
	attachCalls   int
	servers       map[string]string
	volumes       map[string]string
	serverCreates int
	volumeCreates int
}

func (os *fakeOpenStack) FindServerByName(name string) (string, bool, error) {
	id, found := os.servers[name]
	return id, found, nil
}

func (os *fakeOpenStack) CreateServer(name, _, _, _, _ string) (string, error) {
	if os.servers == nil {
		os.servers = make(map[string]string)
	}
	os.serverCreates++
	os.servers[name] = "server-1"
	return "server-1", nil
}

func (os *fakeOpenStack) FindVolumeByName(name string) (string, bool, error) {
	id, found := os.volumes[name]
	return id, found, nil
}

func (os *fakeOpenStack) CreateVolume(name string, _ int) (string, error) {
	if os.volumes == nil {
		os.volumes = make(map[string]string)
	}
	os.volumeCreates++
	os.volumes[name] = "volume-1"
	return "volume-1", nil
}

func (os *fakeOpenStack) GetServerState(string) (string, string, error) {
	return os.serverStatus, os.serverIP, nil
}

func TestProvisioningFindsNovaResourceAfterMetadataWriteFailure(t *testing.T) {
	store := newMemoryStore()
	provider := &fakeOpenStack{serverStatus: "ACTIVE", volumeStatus: "creating"}
	controller := NewController(store, openstack.NewDatabaseProvisioner(provider, openstack.ProvisionerConfig{
		EngineImages: map[string]string{"postgres:16": "image-1"}, PlanFlavors: map[string]string{"basic": "flavor-1"}, NetworkID: "network-1",
	}))
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}},
	})
	instance, err := controller.CreateInstance(CreateInstanceRequest{
		TenantID: "1", ProjectID: "2", Name: "recovery-db", Engine: EnginePostgres,
		EngineVersion: "16", Plan: "basic",
	})
	if err != nil {
		t.Fatalf("create provisioning request: %v", err)
	}

	store.failUpdate = true
	if err := controller.ProvisionInstance(instance.ID); err == nil {
		t.Fatal("expected metadata persistence failure")
	}
	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("retry after metadata failure: %v", err)
	}
	if provider.serverCreates != 1 || provider.volumeCreates != 1 {
		t.Fatalf("expected one Nova server and one Cinder volume, got %d and %d",
			provider.serverCreates, provider.volumeCreates)
	}
}

func (os *fakeOpenStack) IsVolumeAttached(string, string) (bool, error) {
	return os.volumeLinked, nil
}

func (os *fakeOpenStack) GetVolumeState(string) (string, error) {
	return os.volumeStatus, nil
}

func (os *fakeOpenStack) AttachVolume(string, string) error {
	os.attachCalls++
	os.volumeLinked = true
	return nil
}

func TestProvisioningWaitsForOpenStackResources(t *testing.T) {
	store := newMemoryStore()
	provider := &fakeOpenStack{serverStatus: "BUILD", volumeStatus: "creating"}
	controller := NewController(store, openstack.NewDatabaseProvisioner(provider, openstack.ProvisionerConfig{
		EngineImages: map[string]string{"postgres:16": "image-1"}, PlanFlavors: map[string]string{"basic": "flavor-1"}, NetworkID: "network-1", SecurityGroupID: "db-sg",
	}))
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}},
	})
	instance, err := controller.CreateInstance(CreateInstanceRequest{
		TenantID: "1", ProjectID: "2", Name: "analytics-db", Engine: EnginePostgres,
		EngineVersion: "16", Plan: "basic", WorkloadProfile: "hpc",
	})
	if err != nil {
		t.Fatalf("create provisioning request: %v", err)
	}

	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("wait for Nova BUILD state: %v", err)
	}
	provisioned, _ := store.GetInstance(instance.ID)
	state := readProviderState(t, provisioned)
	if provisioned.Status != "provisioning" || state.ServerID != "server-1" ||
		state.ServerStatus != "BUILD" || state.VolumeID != "" {
		t.Fatalf("expected only the Nova server while it is building, got %+v", provisioned)
	}

	provider.serverStatus = "ACTIVE"
	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("wait for Cinder volume: %v", err)
	}
	provisioned, _ = store.GetInstance(instance.ID)
	state = readProviderState(t, provisioned)
	if provisioned.Status != "provisioning" || state.ServerStatus != "ACTIVE" ||
		state.VolumeStatus != "creating" || provider.attachCalls != 0 {
		t.Fatalf("volume should not be attached before Cinder is available: %+v", provisioned)
	}

	provider.volumeStatus = "available"
	provider.serverIP = "10.0.0.15"
	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("attach available volume: %v", err)
	}
	provisioned, _ = store.GetInstance(instance.ID)
	state = readProviderState(t, provisioned)
	if provisioned.Status != "starting" || provisioned.Endpoint != "" ||
		state.ServerIP != "10.0.0.15" || state.VolumeStatus != "available" ||
		!state.VolumeAttached || provider.attachCalls != 1 {
		t.Fatalf("expected VM-ready starting state with one volume attachment, got %+v", provisioned)
	}
}

func TestDatabaseJSONHidesProviderResourceMetadata(t *testing.T) {
	instance := DBInstance{
		Provider: "openstack", ProviderMetadata: json.RawMessage(`{"server_id":"nova-server-id","volume_id":"cinder-volume-id"}`),
	}
	encoded, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("marshal database instance: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("decode database instance JSON: %v", err)
	}
	for _, privateField := range []string{
		"Provider", "ProviderMetadata", "provider", "provider_metadata",
		"openstack_server_id", "openstack_server_status", "openstack_server_ip",
		"openstack_volume_id", "openstack_volume_status", "openstack_volume_attached",
		"openstack_network_id",
	} {
		if _, exists := result[privateField]; exists {
			t.Errorf("database JSON exposed provider field %q: %s", privateField, encoded)
		}
	}
}

func TestCreateInstanceRejectsUnconfiguredEngineVersion(t *testing.T) {
	store := newMemoryStore()
	controller := NewController(store, openstack.NewDatabaseProvisioner(&fakeOpenStack{}, openstack.ProvisionerConfig{
		EngineImages: map[string]string{"postgres:16": "image-1"}, PlanFlavors: map[string]string{"basic": "flavor-1"}, NetworkID: "network-1",
	}))
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}},
	})
	_, err := controller.CreateInstance(CreateInstanceRequest{
		TenantID: "1", ProjectID: "2", Name: "analytics-db", Engine: EnginePostgres,
		EngineVersion: "99", Plan: "basic",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request for unsupported engine version, got %v", err)
	}
}

func TestCreateInstanceOperationIsIdempotent(t *testing.T) {
	store := newMemoryStore()
	controller := NewController(store, openstack.NewDatabaseProvisioner(&fakeOpenStack{}, openstack.ProvisionerConfig{
		EngineImages: map[string]string{"postgres:16": "image-1"}, PlanFlavors: map[string]string{"basic": "flavor-1"}, NetworkID: "network-1",
	}))
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}},
	})
	request := CreateInstanceRequest{
		TenantID: "1", ProjectID: "2", Name: "repeatable-db", Engine: EnginePostgres,
		EngineVersion: "16", Plan: "basic",
	}
	first, err := controller.CreateInstanceOperation(request, "client-key-12345678")
	if err != nil {
		t.Fatalf("create first database request: %v", err)
	}
	replayed, err := controller.CreateInstanceOperation(request, "client-key-12345678")
	if err != nil {
		t.Fatalf("replay database request: %v", err)
	}
	if first.Database.ID != replayed.Database.ID ||
		first.Operation.ID != replayed.Operation.ID || len(store.instances) != 1 {
		t.Fatalf("idempotent retry created a duplicate resource: first=%+v retry=%+v",
			first, replayed)
	}

	request.Name = "different-db"
	if _, err := controller.CreateInstanceOperation(request, "client-key-12345678"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected conflicting request body to be rejected, got %v", err)
	}
}

func readProviderState(t *testing.T, instance *DBInstance) openstack.ProvisioningState {
	t.Helper()
	var state openstack.ProvisioningState
	if err := json.Unmarshal(instance.ProviderMetadata, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

type domainProvisioner struct{ calls int }

func (*domainProvisioner) Name() string { return "test-only" }
func (*domainProvisioner) Supports(engine, version, plan, region string) bool {
	return engine == EnginePostgres && version == "16" && plan == "basic" && region == "test-region"
}
func (provider *domainProvisioner) Provision(instance *DBInstance, checkpoint func(string) error) (ProvisioningProgress, error) {
	provider.calls++
	instance.ProviderMetadata = json.RawMessage(`{"remote_database":"test-resource"}`)
	if err := checkpoint("PROVISIONING_STORAGE"); err != nil {
		return ProvisioningProgress{}, err
	}
	return ProvisioningProgress{InfrastructureReady: true}, nil
}

func TestDatabaseDomainDoesNotRequireInfrastructureServerOrVolumeModels(t *testing.T) {
	store := newMemoryStore()
	provider := &domainProvisioner{}
	controller := NewController(store, provider)
	controller.SetDefaultRegion("test-region")
	controller.SetPlans(map[string]Plan{"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}}})
	if !controller.PlanCatalog()["basic"].Enabled {
		t.Fatal("domain plan required OpenStack configuration")
	}
	request := CreateInstanceRequest{TenantID: "1", ProjectID: "2", Name: "domain-db", Engine: EnginePostgres, EngineVersion: "16", Plan: "basic"}
	result, err := controller.CreateInstanceOperation(request, "domain-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.ProvisionInstance(result.Database.ID); err != nil {
		t.Fatal(err)
	}
	instance, _ := store.GetInstance(result.Database.ID)
	if instance.Status != "starting" || instance.Provider != "test-only" || provider.calls != 1 {
		t.Fatalf("incorrect domain result: %+v", instance)
	}
	instance.Status = "provisioning"
	instance.Provider = "another-adapter"
	if err := store.UpdateInstance(instance); err != nil {
		t.Fatal(err)
	}
	if err := controller.ProvisionInstance(instance.ID); err == nil || provider.calls != 1 {
		t.Fatal("provider mismatch did not fail closed")
	}
}

func TestAdapterFailurePersistsPrivateStatusAndDomainSafeReason(t *testing.T) {
	store := newMemoryStore()
	adapter := openstack.NewDatabaseProvisioner(&fakeOpenStack{serverStatus: "ERROR"}, openstack.ProvisionerConfig{
		EngineImages: map[string]string{"postgres:16": "image"}, PlanFlavors: map[string]string{"basic": "flavor"}, NetworkID: "network",
	})
	controller := NewController(store, adapter)
	controller.SetPlans(map[string]Plan{"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, Engines: []string{EnginePostgres}}})
	result, err := controller.CreateInstanceOperation(CreateInstanceRequest{
		TenantID: "1", ProjectID: "2", Name: "failed-db", Engine: EnginePostgres, EngineVersion: "16", Plan: "basic",
	}, "failure-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.ProvisionInstance(result.Database.ID); !errors.Is(err, ErrProvisioningFailed) {
		t.Fatalf("provider failure not classified: %v", err)
	}
	instance, _ := store.GetInstance(result.Database.ID)
	operation, _ := store.GetOperation(result.Operation.ID)
	if instance.Status != "error" || readProviderState(t, instance).ServerStatus != "ERROR" || operation.Status != "failed" {
		t.Fatal("failure state was not persisted")
	}
	if strings.Contains(operation.Reason, "Nova") || strings.Contains(operation.Reason, "server-1") {
		t.Fatal("domain operation exposed provider diagnostics")
	}
}
