package cloud

import (
	"context"
	"errors"
	"testing"

	"atonixcorp/api/internal/models"
	platformcrypto "atonixcorp/api/internal/security/crypto"
)

type fakeAccountProviderFactory struct {
	Provider
	account CloudAccountCredentials
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
	key := []byte("0123456789abcdef0123456789abcdef")
	service := NewService(db, factory, key)
	account := CloudAccountCredentials{
		ProjectID: "openstack-project", CredentialID: "application-credential",
		CredentialSecret: "private-credential-secret", Region: "RegionOne",
	}

	if err := service.SetCloudAccount(context.Background(), 1, account); err != nil {
		t.Fatalf("SetCloudAccount(): %v", err)
	}
	if factory.account.ProjectID != account.ProjectID ||
		factory.account.CredentialSecret != account.CredentialSecret {
		t.Fatalf("provider did not validate the requested account: %+v", factory.account)
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
	if string(secret) != account.CredentialSecret {
		t.Fatal("stored cloud account credential did not decrypt to the supplied secret")
	}
	configured, err := service.CloudAccountConfigured(context.Background(), 1)
	if err != nil || !configured {
		t.Fatalf("CloudAccountConfigured() = %v, %v; want true", configured, err)
	}
}

func TestSetCloudAccountRequiresEncryptionKey(t *testing.T) {
	db := newCloudTestDB(t)
	factory := &fakeAccountProviderFactory{Provider: &fakeProvider{region: "RegionOne"}}
	service := NewService(db, factory)
	err := service.SetCloudAccount(context.Background(), 1, CloudAccountCredentials{
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
