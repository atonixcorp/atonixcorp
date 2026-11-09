package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/models"
	platformcrypto "atonixcorp/api/internal/security/crypto"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testAccounts struct {
	accounts map[uint]cloud.CloudAccountCredentials
	err      error
	calls    int
}

func TestBlockDriverRegistryRegistersStaticAndScopedDrivers(t *testing.T) {
	registry := NewBlockDriverRegistry()
	local := &scopedTestDriver{name: "local-file"}
	if err := registry.Register(local); err != nil {
		t.Fatalf("register local driver: %v", err)
	}
	if err := registry.Register(local); !errors.Is(err, ErrDriverNameConflict) {
		t.Fatalf("duplicate driver error = %v, want ErrDriverNameConflict", err)
	}
	if err := registry.RegisterFactory(NewCinderDriverFactory()); err != nil {
		t.Fatalf("register Cinder factory: %v", err)
	}
	if registry.DefaultFactoryName() != "openstack-cinder" {
		t.Fatalf("default scoped driver = %q, want openstack-cinder", registry.DefaultFactoryName())
	}
	if _, err := registry.Driver("unknown"); !errors.Is(err, ErrDriverUnavailable) {
		t.Fatalf("unknown driver error = %v, want ErrDriverUnavailable", err)
	}
	if _, err := NewScopedDriversWithRegistry(
		&testAccounts{}, registry, "local-file",
	); err != nil {
		t.Fatalf("create scoped drivers from registry: %v", err)
	}
	if _, err := NewScopedDrivers(
		&testAccounts{}, &scopedTestDriver{name: "openstack-cinder"}, CinderConfig{},
	); !errors.Is(err, ErrDriverNameConflict) {
		t.Fatalf("conflicting driver registration error = %v, want ErrDriverNameConflict", err)
	}
}

func TestScopedCinderUsesTenantTokenAndEndpoint(t *testing.T) {
	accounts := &testAccounts{accounts: map[uint]cloud.CloudAccountCredentials{
		1: cloud.OpenStackAccountCredentials{ProjectID: "project-one", CredentialID: "cred-one", CredentialSecret: "secret-one", Region: "region"},
		2: cloud.OpenStackAccountCredentials{ProjectID: "project-two", CredentialID: "cred-two", CredentialSecret: "secret-two", Region: "region"},
	}}
	resolver, err := NewScopedDrivers(accounts, nil, CinderConfig{
		AuthURL: "https://keystone.example/v3", Interface: "public", VolumeTypeIDs: []string{"fast"},
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	resolver.newCinder = func(cfg CinderConfig) (BlockDriver, error) {
		driver, err := NewCinderDriver(cfg)
		if err != nil {
			return nil, err
		}
		driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			respond := func(status int, body string) *http.Response {
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}
			}
			if request.URL.Host == "keystone.example" {
				var body struct {
					Auth struct {
						Identity struct {
							Credential struct {
								ID     string `json:"id"`
								Secret string `json:"secret"`
							} `json:"application_credential"`
						} `json:"identity"`
					} `json:"auth"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Auth.Identity.Credential.ID != cfg.ApplicationCredentialID || body.Auth.Identity.Credential.Secret != cfg.ApplicationCredentialSecret {
					t.Fatal("wrong Keystone application credential")
				}
				response := respond(http.StatusCreated, `{"token":{"expires_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`","project":{"id":"`+cfg.ProjectID+`"},"catalog":[{"type":"volumev3","endpoints":[{"region_id":"region","interface":"public","url":"https://cinder.example/v3/%(project_id)s"}]}]}}`)
				response.Header.Set("X-Subject-Token", "token-"+cfg.ProjectID)
				return response, nil
			}
			if request.URL.Path != "/v3/"+cfg.ProjectID+"/volumes/volume-id" || request.Header.Get("X-Auth-Token") != "token-"+cfg.ProjectID {
				t.Fatal("Cinder request used the wrong project or token")
			}
			seen[cfg.ProjectID] = true
			return respond(http.StatusOK, `{"volume":{"id":"volume-id","status":"available","size":8}}`), nil
		})}
		return driver, nil
	}
	for tenant := uint(1); tenant <= 2; tenant++ {
		driver, _, err := resolver.forTenant(context.Background(), tenant)
		if err != nil {
			t.Fatal(err)
		}
		volume, err := driver.InspectVolume(context.Background(), "volume-id")
		if err != nil || volume.SizeGB != 8 {
			t.Fatal("scoped Cinder inspection failed")
		}
	}
	if !seen["project-one"] || !seen["project-two"] {
		t.Fatal("both tenant endpoints were not used")
	}
}

func TestMappedAccountCreateWorkerUsesEncryptedCredentials(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CloudAccount{}, &models.TenantMembership{}, &models.Project{}, &models.ProjectMembership{},
		&models.StorageVolume{}, &models.StorageQuota{}, &models.StorageOperation{}, &models.StorageAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	secret, err := platformcrypto.EncryptAESGCM(key, []byte("tenant-secret"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []any{
		&models.CloudAccount{TenantID: 1, ScopeID: "tenant-project", CredentialID: "tenant-credential", EncryptedSecret: secret, Region: "region"},
		&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"},
		&models.Project{ID: 1, TenantID: 1, Name: "project", Slug: "project"},
		&models.ProjectMembership{ProjectID: 1, UserID: 1, Role: "owner"},
	} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	accounts := cloud.NewService(db, &cloud.OpenStackProvider{}, key)
	local := &scopedTestDriver{name: "local-file"}
	resolver, err := NewScopedDrivers(accounts, local, CinderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	provider := &scopedTestDriver{name: "openstack-cinder", volume: ProviderVolume{ID: "provider-id", Status: "available", SizeGB: 12}}
	resolver.newCinder = func(cfg CinderConfig) (BlockDriver, error) {
		if cfg.ProjectID != "tenant-project" || cfg.ApplicationCredentialID != "tenant-credential" || cfg.ApplicationCredentialSecret != "tenant-secret" {
			t.Fatal("wrong mapped provider credentials")
		}
		return provider, nil
	}
	service := NewService(db, local)
	service.SetScopedDrivers(resolver)
	request := CreateVolumeRequest{TenantID: 1, ProjectID: 1, Name: "test-volume", Region: "region", StorageClass: "fast", SizeGB: 12}
	result, err := service.CreateVolume(ctx, 1, "request-key", request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Volume.Driver != "openstack-cinder" || result.Volume.ProviderProjectID != "tenant-project" {
		t.Fatal("creation was not pinned to the mapped project")
	}
	if err := service.processNext(ctx); err != nil {
		t.Fatal(err)
	}
	volume, err := service.GetVolume(ctx, 1, result.Volume.ID)
	if err != nil || volume.ProviderID != "provider-id" || volume.ProviderStatus != "available" || volume.State != "AVAILABLE" {
		t.Fatalf("worker did not reconcile mapped volume: %v", err)
	}
	if err := db.Model(&models.CloudAccount{}).Where("tenant_id = ?", 1).Update("open_stack_project_id", "other-project").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetVolume(ctx, 1, result.Volume.ID); !errors.Is(err, ErrDriverUnavailable) {
		t.Fatal("changed mapping accessed a volume from the original project")
	}
}

func (a *testAccounts) StorageAccountCredentials(_ context.Context, tenant uint) (cloud.CloudAccountCredentials, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	account, ok := a.accounts[tenant]
	if !ok {
		return nil, cloud.ErrCloudAccountNotConfigured
	}
	return account, nil
}

type scopedTestDriver struct {
	name       string
	volume     ProviderVolume
	inspectErr error
}

func (d *scopedTestDriver) Name() string { return d.name }
func (d *scopedTestDriver) VolumeTypes(context.Context) ([]VolumeType, error) {
	return []VolumeType{{ID: "fast", Region: "region"}}, nil
}
func (d *scopedTestDriver) ValidateVolume(context.Context, string, string, string, bool) error {
	return nil
}
func (d *scopedTestDriver) EnsureVolume(context.Context, models.StorageVolume) (ProviderVolume, error) {
	return d.volume, nil
}
func (d *scopedTestDriver) InspectVolume(context.Context, string) (ProviderVolume, error) {
	return d.volume, d.inspectErr
}
func (d *scopedTestDriver) DeleteVolume(context.Context, models.StorageVolume) (bool, error) {
	return true, nil
}

func TestScopedDriversFailClosedAndPinProject(t *testing.T) {
	ctx := context.Background()
	accounts := &testAccounts{accounts: map[uint]cloud.CloudAccountCredentials{
		1: cloud.OpenStackAccountCredentials{ProjectID: "project-one", CredentialID: "cred-one", CredentialSecret: "secret-one", Region: "region-one"},
		2: cloud.OpenStackAccountCredentials{ProjectID: "project-two", CredentialID: "cred-two", CredentialSecret: "secret-two", Region: "region-two"},
	}}
	local := &scopedTestDriver{name: "local-file"}
	resolver, err := NewScopedDrivers(accounts, local, CinderConfig{
		ProjectID: "shared-project", Username: "shared-user", Password: "shared-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver.newCinder = func(cfg CinderConfig) (BlockDriver, error) {
		if cfg.ProjectID == "shared-project" || cfg.Username != "" || cfg.Password != "" {
			t.Fatal("shared credentials reached tenant driver")
		}
		if cfg.ProjectID == "project-one" && (cfg.ApplicationCredentialID != "cred-one" || cfg.ApplicationCredentialSecret != "secret-one" || cfg.Region != "region-one") {
			t.Fatal("tenant one credentials or region not used")
		}
		if cfg.ProjectID == "project-two" && (cfg.ApplicationCredentialID != "cred-two" || cfg.ApplicationCredentialSecret != "secret-two" || cfg.Region != "region-two") {
			t.Fatal("tenant two credentials or region not used")
		}
		return &scopedTestDriver{name: "openstack-cinder"}, nil
	}
	for tenant := uint(1); tenant <= 2; tenant++ {
		driver, project, err := resolver.forTenant(ctx, tenant)
		if err != nil || driver.Name() != "openstack-cinder" || project != fmt.Sprintf("project-%s", []string{"one", "two"}[tenant-1]) {
			t.Fatal("incorrect scoped driver")
		}
	}
	driver, _, err := resolver.forTenant(ctx, 3)
	if err != nil || driver != local {
		t.Fatal("unmapped tenant did not use explicitly configured local driver")
	}
	accounts.err = errors.New("decryption failure")
	if _, _, err := resolver.forTenant(ctx, 1); !errors.Is(err, ErrDriverUnavailable) {
		t.Fatal("mapped account failure must not fall back")
	}
	accounts.err = nil
	for _, project := range []string{"", "project-two"} {
		if _, err := resolver.forVolume(ctx, models.StorageVolume{TenantID: 1, Driver: "openstack-cinder", ProviderProjectID: project}); !errors.Is(err, ErrDriverUnavailable) {
			t.Fatal("missing or changed provider project was not rejected")
		}
	}
	if _, err := resolver.forVolume(ctx, models.StorageVolume{TenantID: 1, Driver: "local-file"}); err != nil {
		t.Fatal("existing local volume should retain its driver")
	}
}

func TestScopedVolumeReadReconcilesAndAuthorizes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.TenantMembership{}, &models.Project{}, &models.ProjectMembership{}, &models.StorageVolume{}); err != nil {
		t.Fatal(err)
	}
	for _, item := range []any{
		&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"},
		&models.Project{ID: 1, TenantID: 1, Name: "project", Slug: "project"},
		&models.ProjectMembership{ProjectID: 1, UserID: 1, Role: "owner"},
		&models.StorageVolume{ID: "volume", TenantID: 1, ProjectID: 1, OwnerID: 1, Name: "volume", Region: "region", StorageClass: "fast", SizeGB: 10, State: "AVAILABLE", Driver: "openstack-cinder", ProviderID: "cinder-id", ProviderProjectID: "project-one", IdempotencyKey: "key", RequestHash: "hash"},
	} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	accounts := &testAccounts{accounts: map[uint]cloud.CloudAccountCredentials{1: cloud.OpenStackAccountCredentials{ProjectID: "project-one", CredentialID: "cred", CredentialSecret: "secret"}}}
	resolver, err := NewScopedDrivers(accounts, nil, CinderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	provider := &scopedTestDriver{name: "openstack-cinder", volume: ProviderVolume{ID: "cinder-id", Status: "in-use", SizeGB: 20, AvailabilityZone: "zone-a", Attachments: []models.StorageAttachment{{ID: "attachment", ServerID: "server", Device: "/dev/vdb"}}}}
	resolver.newCinder = func(CinderConfig) (BlockDriver, error) { return provider, nil }
	service := NewService(db, nil)
	service.SetScopedDrivers(resolver)
	if _, err := service.GetVolume(context.Background(), 2, "volume"); !errors.Is(err, ErrNotFound) || accounts.calls != 0 {
		t.Fatal("unauthorized read accessed provider")
	}
	volume, err := service.GetVolume(context.Background(), 1, "volume")
	if err != nil || volume.ProviderStatus != "in-use" || volume.State != "IN-USE" || volume.SizeGB != 20 || volume.AvailabilityZone != "zone-a" || len(volume.Attachments) != 1 {
		t.Fatalf("reconciliation failed: %v", err)
	}
	var persisted models.StorageVolume
	if err := db.First(&persisted, "id = ?", "volume").Error; err != nil || persisted.SizeGB != 20 || persisted.State != "IN-USE" {
		t.Fatal("provider details were not reconciled")
	}
	provider.inspectErr = ErrProviderNotFound
	volume, err = service.GetVolume(context.Background(), 1, "volume")
	if err != nil || volume.State != "ERROR" || volume.ProviderStatus != "not_found" {
		t.Fatal("missing provider volume not reconciled")
	}
}
