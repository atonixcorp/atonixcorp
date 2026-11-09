package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"atonixcorp/api/internal/models"
	platformcrypto "atonixcorp/api/internal/security/crypto"
)

type fakeAccountProviderFactory struct {
	Provider
	account CloudAccountCredentials
}

func (f *fakeAccountProviderFactory) DecodeStoredAccount(record models.CloudAccount, secret []byte) (CloudAccountCredentials, error) {
	return (&OpenStackProvider{}).DecodeStoredAccount(record, secret)
}

func (f *fakeAccountProviderFactory) ForAccount(
	_ context.Context, account CloudAccountCredentials,
) (Provider, error) {
	f.account = account
	return f.Provider, nil
}

func TestSetCloudAccountPersistsEncryptedOrganizationMapping(t *testing.T) {
	db := newCloudTestDB(t)
	provider := &fakeProvider{region: "RegionOne"}
	factory := &fakeAccountProviderFactory{Provider: provider}
	registry := NewComputeProviderRegistry()
	if err := registry.Register(factory); err != nil {
		t.Fatalf("register compute provider: %v", err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	service := NewServiceWithRegistry(db, provider, registry, key)
	account := OpenStackAccountCredentials{
		ProjectID: "openstack-project", CredentialID: "application-credential",
		CredentialSecret: "private-credential-secret", Region: "RegionOne",
	}

	if err := service.SetCloudAccount(context.Background(), 1, account); err != nil {
		t.Fatalf("SetCloudAccount(): %v", err)
	}
	validated, ok := factory.account.(OpenStackAccountCredentials)
	if !ok || validated.ProjectID != account.ProjectID ||
		validated.CredentialSecret != account.CredentialSecret {
		t.Fatalf("provider did not validate the requested account: %+v", factory.account)
	}
	if _, err := service.providerForTenant(context.Background(), 1); err != nil {
		t.Fatalf("resolve tenant compute provider through registry: %v", err)
	}

	var stored models.CloudAccount
	if err := db.Where("tenant_id = ?", 1).First(&stored).Error; err != nil {
		t.Fatalf("load cloud account mapping: %v", err)
	}
	if string(stored.EncryptedSecret) == account.CredentialSecret {
		t.Fatal("cloud account credential was stored in plaintext")
	}
	secret, err := platformcrypto.DecryptAESGCM(key, stored.EncryptedSecret)
	if err != nil {
		t.Fatalf("decrypt stored cloud account credential: %v", err)
	}
	var decoded OpenStackAccountCredentials
	if err := json.Unmarshal(secret, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != account || stored.CredentialVersion != 1 || stored.Provider != "openstack" {
		t.Fatal("stored cloud account credential did not decrypt to the supplied secret")
	}
	configured, err := service.CloudAccountConfigured(context.Background(), 1)
	if err != nil || !configured {
		t.Fatalf("CloudAccountConfigured() = %v, %v; want true", configured, err)
	}
}

func TestComputeProviderRegistryRejectsDuplicatesAndUnknownNames(t *testing.T) {
	provider := &fakeProvider{region: "RegionOne"}
	factory := &fakeAccountProviderFactory{Provider: provider}
	registry := NewComputeProviderRegistry()
	if err := registry.Register(factory); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if registry.DefaultName() != "openstack" {
		t.Fatalf("registry default = %q, want openstack", registry.DefaultName())
	}
	if err := registry.Register(factory); !errors.Is(err, ErrProviderNameConflict) {
		t.Fatalf("duplicate registration error = %v, want ErrProviderNameConflict", err)
	}
	if _, err := registry.ForAccount(
		context.Background(), "unknown", OpenStackAccountCredentials{},
	); !errors.Is(err, ErrProviderNotRegistered) {
		t.Fatalf("unknown provider error = %v, want ErrProviderNotRegistered", err)
	}
	resolved, err := registry.ForAccount(context.Background(), "openstack", OpenStackAccountCredentials{})
	if err != nil {
		t.Fatalf("resolve registered provider: %v", err)
	}
	if resolved != provider {
		t.Fatalf("resolved provider = %v, want registered provider", resolved)
	}
}

func TestSetCloudAccountRequiresEncryptionKey(t *testing.T) {
	db := newCloudTestDB(t)
	factory := &fakeAccountProviderFactory{Provider: &fakeProvider{region: "RegionOne"}}
	service := NewService(db, factory)
	err := service.SetCloudAccount(context.Background(), 1, OpenStackAccountCredentials{
		ProjectID: "project", CredentialID: "credential", CredentialSecret: "secret",
	})
	if !errors.Is(err, ErrAccountEncryptionUnavailable) {
		t.Fatalf("SetCloudAccount() error = %v, want ErrAccountEncryptionUnavailable", err)
	}
}

func TestAccountEncryptionKeyFromEnvironment(t *testing.T) {
	key, err := AccountEncryptionKeyFromEnvironment("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	if err != nil || len(key) != 32 {
		t.Fatalf("AccountEncryptionKeyFromEnvironment() = %d bytes, %v; want 32 bytes", len(key), err)
	}
	if _, err := AccountEncryptionKeyFromEnvironment("not-a-valid-key"); err == nil {
		t.Fatal("AccountEncryptionKeyFromEnvironment() accepted invalid key material")
	}
}

type tokenOnlyCredentials struct {
	Token string `json:"token"`
}

func (tokenOnlyCredentials) ProviderName() string { return "test-only" }
func (tokenOnlyCredentials) ScopeID() string      { return "test-account" }
func (tokenOnlyCredentials) RegionName() string   { return "" }
func (credential tokenOnlyCredentials) Validate() error {
	if credential.Token == "" {
		return ErrInvalid
	}
	return nil
}
func (credential tokenOnlyCredentials) Encode() ([]byte, error) { return json.Marshal(credential) }

type tokenOnlyFactory struct{ Provider }

func (*tokenOnlyFactory) Name() string { return "test-only" }
func (factory *tokenOnlyFactory) ForAccount(_ context.Context, credential CloudAccountCredentials) (ComputeProvider, error) {
	if _, ok := credential.(tokenOnlyCredentials); !ok {
		return nil, ErrInvalid
	}
	return factory, nil
}
func (*tokenOnlyFactory) DecodeStoredAccount(record models.CloudAccount, payload []byte) (CloudAccountCredentials, error) {
	var credential tokenOnlyCredentials
	if record.Provider != "test-only" || record.CredentialVersion != 1 {
		return nil, ErrInvalid
	}
	if err := json.Unmarshal(payload, &credential); err != nil {
		return nil, err
	}
	return credential, credential.Validate()
}

func TestAccountServiceDoesNotRequireOpenStackCredentialFields(t *testing.T) {
	db := newCloudTestDB(t)
	factory := &tokenOnlyFactory{Provider: &fakeProvider{}}
	registry := NewComputeProviderRegistry()
	if err := registry.Register(factory); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWithRegistry(db, factory, registry, []byte("0123456789abcdef0123456789abcdef"))
	if err := service.SetCloudAccount(context.Background(), 1, tokenOnlyCredentials{Token: "test-secret"}); err != nil {
		t.Fatal(err)
	}
	credential, err := service.StorageAccountCredentials(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if credential.(tokenOnlyCredentials).Token != "test-secret" {
		t.Fatal("generic credentials did not round-trip")
	}
}

func TestOpenStackCredentialDecoderPreservesLegacyRecords(t *testing.T) {
	adapter := &OpenStackProvider{}
	credential, err := adapter.DecodeStoredAccount(models.CloudAccount{
		Provider: "openstack", ScopeID: "project", CredentialID: "credential", Region: "RegionOne",
	}, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if credential.(OpenStackAccountCredentials).CredentialSecret != "secret" {
		t.Fatal("legacy secret lost")
	}
	if _, err := adapter.ForAccount(context.Background(), tokenOnlyCredentials{Token: "secret"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("OpenStack accepted foreign credentials")
	}
}
