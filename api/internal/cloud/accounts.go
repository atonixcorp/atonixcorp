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
	ErrCloudAccountNotConfigured    = errors.New("cloud account is not configured for this organization")
	ErrAccountEncryptionUnavailable = errors.New("cloud account encryption is not configured")
)

type CloudAccountProviderFactory = ComputeProviderFactory

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
	if tenantID == 0 || account == nil || account.ProviderName() != s.providerName() {
		return ErrInvalid
	}
	if err := account.Validate(); err != nil {
		return ErrInvalid
	}
	if len(s.secretKey) != 32 {
		return ErrAccountEncryptionUnavailable
	}
	payload, err := account.Encode()
	if err != nil {
		return ErrInvalid
	}
	provider, err := s.providerForAccount(ctx, account)
	if err != nil {
		if errors.Is(err, ErrProviderUnavailable) || errors.Is(err, ErrProviderNotRegistered) {
			return ErrProviderUnavailable
		}
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
		s.secretKey, payload,
	)
	if err != nil {
		return fmt.Errorf("encrypt organization cloud credential: %w", err)
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
		Where("open_stack_project_id = ? AND tenant_id <> ?", account.ScopeID(), tenantID).
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
			TenantID: tenantID, Provider: account.ProviderName(), ScopeID: account.ScopeID(),
			CredentialVersion: 1, EncryptedSecret: encryptedSecret,
			Region: account.RegionName(),
		}
		return s.db.WithContext(ctx).Create(&record).Error
	}
	if err != nil {
		return err
	}
	if record.ScopeID != account.ScopeID() || record.Provider != account.ProviderName() {
		var resourceCount int64
		if err := s.db.WithContext(ctx).Model(&models.CloudInstance{}).
			Where("tenant_id = ?", tenantID).Count(&resourceCount).Error; err != nil {
			return err
		}
		if resourceCount > 0 {
			return ErrConflict
		}
	}
	record.Provider = account.ProviderName()
	record.ScopeID = account.ScopeID()
	record.CredentialVersion = 1
	record.CredentialID = ""
	record.EncryptedSecret = encryptedSecret
	record.Region = account.RegionName()
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
			return nil, ErrCloudAccountNotConfigured
		}
		return nil, err
	}
	if len(s.secretKey) != 32 {
		return nil, ErrAccountEncryptionUnavailable
	}
	secret, err := platformcrypto.DecryptAESGCM(s.secretKey, account.EncryptedSecret)
	if err != nil {
		return nil, errors.New("unable to decrypt organization cloud credential")
	}
	if s.registry != nil {
		return s.registry.DecodeStoredAccount(account, secret)
	}
	decoder, ok := s.provider.(AccountCredentialDecoder)
	if !ok {
		return nil, ErrProviderUnavailable
	}
	return decoder.DecodeStoredAccount(account, secret)
}

func (s *Service) TenantCapabilities(
	ctx context.Context, userID, tenantID uint,
) (bool, string, ComputeCapabilities, error) {
	if userID == 0 || tenantID == 0 {
		return false, "", ComputeCapabilities{}, ErrForbidden
	}
	var membership models.TenantMembership
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", ComputeCapabilities{}, ErrForbidden
	}
	if err != nil {
		return false, "", ComputeCapabilities{}, err
	}
	configured, err := s.CloudAccountConfigured(ctx, tenantID)
	if err != nil {
		return false, "", ComputeCapabilities{}, err
	}
	if !configured {
		return false, s.Region(), ComputeCapabilities{}, nil
	}
	provider, err := s.providerForTenant(ctx, tenantID)
	if err != nil {
		return false, "", ComputeCapabilities{}, err
	}
	capabilities := ComputeCapabilities{}
	if capabilityProvider, ok := provider.(interface {
		Capabilities() ComputeCapabilities
	}); ok {
		capabilities = capabilityProvider.Capabilities()
	}
	enabled := capabilities.Compute && capabilities.Image && capabilities.Network &&
		len(s.secretKey) == 32
	return enabled, provider.Region(), capabilities, nil
}

func (s *Service) providerForTenant(ctx context.Context, tenantID uint) (Provider, error) {
	providerName := s.providerName()
	if providerName == "" {
		return nil, ErrProviderUnavailable
	}
	if s.registry == nil {
		_, ok := s.provider.(CloudAccountProviderFactory)
		if !ok {
			if s.provider == nil {
				return nil, ErrProviderUnavailable
			}
			return s.provider, nil
		}
	}
	account, err := s.StorageAccountCredentials(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerForAccount(ctx, account)
	if err != nil {
		if errors.Is(err, ErrProviderUnavailable) || errors.Is(err, ErrProviderNotRegistered) {
			return nil, ErrProviderUnavailable
		}
		return nil, fmt.Errorf("%w: connect organization cloud account", ErrProviderRequest)
	}
	return provider, nil
}

func (s *Service) providerForAccount(
	ctx context.Context,
	account CloudAccountCredentials,
) (ComputeProvider, error) {
	if s.registry != nil {
		name := s.providerName()
		if name == "" {
			return nil, ErrProviderUnavailable
		}
		return s.registry.ForAccount(ctx, name, account)
	}
	factory, ok := s.provider.(CloudAccountProviderFactory)
	if !ok {
		return nil, ErrProviderUnavailable
	}
	provider, err := factory.ForAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("%w: provider factory returned no client", ErrProviderRequest)
	}
	return provider, nil
}

func (s *Service) providerName() string {
	if s.provider != nil {
		return s.provider.Name()
	}
	if s.registry != nil {
		return s.registry.DefaultName()
	}
	return ""
}
