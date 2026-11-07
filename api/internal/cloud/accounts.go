package cloud

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"atonixcorp/api/internal/models"
	platformcrypto "atonixcorp/api/internal/security/crypto"

	"gorm.io/gorm"
)

var (
	ErrCloudAccountNotConfigured    = errors.New("OpenStack account is not configured for this organization")
	ErrAccountEncryptionUnavailable = errors.New("OpenStack account encryption is not configured")
)

type CloudAccountProviderFactory interface {
	ForAccount(context.Context, CloudAccountCredentials) (Provider, error)
}

func AccountEncryptionKeyFromEnvironment(encoded string) ([]byte, error) {
	if strings.TrimSpace(encoded) == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(key) != 32 {
		return nil, errors.New("OPENSTACK_ACCOUNT_ENCRYPTION_KEY must be base64-encoded 32-byte key material")
	}
	return key, nil
}

func (s *Service) SetCloudAccount(
	ctx context.Context,
	tenantID uint,
	account CloudAccountCredentials,
) error {
	if tenantID == 0 || strings.TrimSpace(account.ProjectID) == "" ||
		strings.TrimSpace(account.CredentialID) == "" ||
		strings.TrimSpace(account.CredentialSecret) == "" {
		return ErrInvalid
	}
	if len(s.secretKey) != 32 {
		return ErrAccountEncryptionUnavailable
	}
	account.ProjectID = strings.TrimSpace(account.ProjectID)
	account.CredentialID = strings.TrimSpace(account.CredentialID)
	account.Region = strings.TrimSpace(account.Region)
	if account.Region == "" {
		account.Region = s.Region()
	}
	if len(account.ProjectID) > 255 || len(account.CredentialID) > 255 || len(account.Region) > 64 {
		return ErrInvalid
	}
	factory, ok := s.provider.(CloudAccountProviderFactory)
	if !ok {
		return ErrProviderUnavailable
	}
	provider, err := factory.ForAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("%w: validate organization cloud account", ErrProviderRequest)
	}
	if provider == nil {
		return fmt.Errorf("%w: provider factory returned no client", ErrProviderRequest)
	}
	if computeProvider, ok := provider.(interface{ ComputeAvailable() bool }); ok &&
		!computeProvider.ComputeAvailable() {
		return ErrProviderUnavailable
	}
	encryptedSecret, err := platformcrypto.EncryptAESGCM(
		s.secretKey, []byte(account.CredentialSecret),
	)
	if err != nil {
		return fmt.Errorf("encrypt OpenStack organization credential: %w", err)
	}

	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	var projectMapping models.CloudAccount
	err = s.db.WithContext(ctx).
		Where("open_stack_project_id = ? AND tenant_id <> ?", account.ProjectID, tenantID).
		First(&projectMapping).Error
	if err == nil {
		return ErrConflict
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var record models.CloudAccount
	err = s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = models.CloudAccount{
			TenantID: tenantID, OpenStackProjectID: account.ProjectID,
			CredentialID: account.CredentialID, EncryptedSecret: encryptedSecret,
			Region: account.Region,
		}
		return s.db.WithContext(ctx).Create(&record).Error
	}
	if err != nil {
		return err
	}
	if record.OpenStackProjectID != account.ProjectID {
		var resourceCount int64
		if err := s.db.WithContext(ctx).Model(&models.CloudInstance{}).
			Where("tenant_id = ?", tenantID).Count(&resourceCount).Error; err != nil {
			return err
		}
		if resourceCount > 0 {
			return ErrConflict
		}
	}
	record.OpenStackProjectID = account.ProjectID
	record.CredentialID = account.CredentialID
	record.EncryptedSecret = encryptedSecret
	record.Region = account.Region
	return s.db.WithContext(ctx).Save(&record).Error
}

func (s *Service) CloudAccountConfigured(ctx context.Context, tenantID uint) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.CloudAccount{}).
		Where("tenant_id = ?", tenantID).Count(&count).Error
	return count == 1, err
}

// StorageAccountCredentials is for internal provider construction after project authorization.
func (s *Service) StorageAccountCredentials(ctx context.Context, tenantID uint) (CloudAccountCredentials, error) {
	var account models.CloudAccount
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CloudAccountCredentials{}, ErrCloudAccountNotConfigured
		}
		return CloudAccountCredentials{}, err
	}
	if len(s.secretKey) != 32 {
		return CloudAccountCredentials{}, ErrAccountEncryptionUnavailable
	}
	secret, err := platformcrypto.DecryptAESGCM(s.secretKey, account.EncryptedSecret)
	if err != nil {
		return CloudAccountCredentials{}, errors.New("unable to decrypt organization cloud credential")
	}
	return CloudAccountCredentials{
		ProjectID: account.OpenStackProjectID, CredentialID: account.CredentialID,
		CredentialSecret: string(secret), Region: account.Region,
	}, nil
}

func (s *Service) TenantCapabilities(
	ctx context.Context, userID, tenantID uint,
) (bool, string, OpenStackCapabilities, error) {
	if userID == 0 || tenantID == 0 {
		return false, "", OpenStackCapabilities{}, ErrForbidden
	}
	var membership models.TenantMembership
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", OpenStackCapabilities{}, ErrForbidden
	}
	if err != nil {
		return false, "", OpenStackCapabilities{}, err
	}
	configured, err := s.CloudAccountConfigured(ctx, tenantID)
	if err != nil {
		return false, "", OpenStackCapabilities{}, err
	}
	if !configured {
		return false, s.Region(), OpenStackCapabilities{}, nil
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return false, "", OpenStackCapabilities{}, err
	}
	capabilities := OpenStackCapabilities{}
	if capabilityProvider, ok := provider.(interface {
		Capabilities() OpenStackCapabilities
	}); ok {
		capabilities = capabilityProvider.Capabilities()
	}
	enabled := capabilities.Compute && capabilities.Image && capabilities.Network &&
		len(s.secretKey) == 32
	return enabled, provider.Region(), capabilities, nil
}

func (s *Service) providerForTenant(ctx context.Context, tenantID uint) (Provider, error) {
	if s.provider == nil {
		return nil, ErrProviderUnavailable
	}
	factory, ok := s.provider.(CloudAccountProviderFactory)
	if !ok {
		return s.provider, nil
	}
	if len(s.secretKey) != 32 {
		return nil, ErrAccountEncryptionUnavailable
	}
	var account models.CloudAccount
	if err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).
		First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCloudAccountNotConfigured
		}
		return nil, err
	}
	secret, err := platformcrypto.DecryptAESGCM(s.secretKey, account.EncryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt OpenStack organization credential: %w", err)
	}
	provider, err := factory.ForAccount(ctx, CloudAccountCredentials{
		ProjectID: account.OpenStackProjectID, CredentialID: account.CredentialID,
		CredentialSecret: string(secret), Region: account.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: connect organization cloud account", ErrProviderRequest)
	}
	if provider == nil {
		return nil, fmt.Errorf("%w: provider factory returned no client", ErrProviderRequest)
	}
	return provider, nil
}
