package core

import (
	"errors"
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
	controller := NewController(store, provider)
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, FlavorID: "flavor-1", Engines: []string{EnginePostgres}},
	})
	controller.SetProvisioningConfig(
		map[string]string{"postgres:16": "image-1"}, "network-1", "",
	)
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
	controller := NewController(store, provider)
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, FlavorID: "flavor-1", Engines: []string{EnginePostgres}},
	})
	controller.SetProvisioningConfig(
		map[string]string{"postgres:16": "image-1"}, "network-1", "db-sg",
	)
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
	if provisioned.Status != "provisioning" || provisioned.OpenStackServerID != "server-1" ||
		provisioned.OpenStackVolumeID != "" {
		t.Fatalf("expected only the Nova server while it is building, got %+v", provisioned)
	}

	provider.serverStatus = "ACTIVE"
	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("wait for Cinder volume: %v", err)
	}
	provisioned, _ = store.GetInstance(instance.ID)
	if provisioned.Status != "provisioning" || provider.attachCalls != 0 {
		t.Fatalf("volume should not be attached before Cinder is available: %+v", provisioned)
	}

	provider.volumeStatus = "available"
	provider.serverIP = "10.0.0.15"
	if err := controller.ProvisionInstance(instance.ID); err != nil {
		t.Fatalf("attach available volume: %v", err)
	}
	provisioned, _ = store.GetInstance(instance.ID)
	if provisioned.Status != "starting" || provisioned.Endpoint != "" ||
		!provisioned.OpenStackVolumeAttached || provider.attachCalls != 1 {
		t.Fatalf("expected VM-ready starting state with one volume attachment, got %+v", provisioned)
	}
}

func TestCreateInstanceRejectsUnconfiguredEngineVersion(t *testing.T) {
	store := newMemoryStore()
	controller := NewController(store, &fakeOpenStack{})
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, FlavorID: "flavor-1", Engines: []string{EnginePostgres}},
	})
	controller.SetProvisioningConfig(
		map[string]string{"postgres:16": "image-1"}, "network-1", "",
	)
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
	controller := NewController(store, &fakeOpenStack{})
	controller.SetPlans(map[string]Plan{
		"basic": {CPU: 1, MemoryMB: 2048, StorageGB: 20, FlavorID: "flavor-1", Engines: []string{EnginePostgres}},
	})
	controller.SetProvisioningConfig(
		map[string]string{"postgres:16": "image-1"}, "network-1", "",
	)
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
