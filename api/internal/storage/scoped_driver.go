package storage

import (
	"context"
	"errors"
	"fmt"

	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/models"
)

type AccountCredentialsSource interface {
	StorageAccountCredentials(context.Context, uint) (cloud.CloudAccountCredentials, error)
}

type ScopedDrivers struct {
	accounts        AccountCredentialsSource
	registry        *BlockDriverRegistry
	localDriverName string
	newCinder       func(CinderConfig) (BlockDriver, error)
}

func NewScopedDrivers(
	accounts AccountCredentialsSource,
	local BlockDriver,
	config CinderConfig,
) (*ScopedDrivers, error) {
	if accounts == nil {
		return nil, errors.New("storage account credential source is required")
	}
	registry := NewBlockDriverRegistry()
	scoped := &ScopedDrivers{
		accounts:  accounts,
		newCinder: func(cfg CinderConfig) (BlockDriver, error) { return NewCinderDriver(cfg) },
	}
	if local != nil {
		if err := registry.Register(local); err != nil {
			return nil, err
		}
		scoped.localDriverName = local.Name()
	}
	if err := registry.RegisterFactory(&cinderDriverFactory{
		config: config,
		newDriver: func(cfg CinderConfig) (BlockDriver, error) {
			return scoped.newCinder(cfg)
		},
	}); err != nil {
		return nil, err
	}
	scoped.registry = registry
	return scoped, nil
}

func NewScopedDriversWithRegistry(
	accounts AccountCredentialsSource,
	registry *BlockDriverRegistry,
	localDriverName string,
) (*ScopedDrivers, error) {
	if accounts == nil || registry == nil {
		return nil, errors.New("storage account source and driver registry are required")
	}
	if registry.DefaultFactoryName() == "" {
		return nil, fmt.Errorf("%w: no account-scoped storage adapter is registered", ErrDriverUnavailable)
	}
	if localDriverName != "" {
		driver, err := registry.Driver(localDriverName)
		if err != nil || driver.Name() != localDriverName {
			return nil, fmt.Errorf("%w: local storage driver is not registered", ErrDriverUnavailable)
		}
	}
	return &ScopedDrivers{
		accounts: accounts, registry: registry, localDriverName: localDriverName,
	}, nil
}

func (r *ScopedDrivers) forTenant(ctx context.Context, tenantID uint) (BlockDriver, string, error) {
	return r.forTenantProject(ctx, tenantID, "")
}

func (r *ScopedDrivers) forTenantProject(ctx context.Context, tenantID uint, expectedProject string) (BlockDriver, string, error) {
	if r == nil || r.accounts == nil || r.registry == nil {
		return nil, "", ErrDriverUnavailable
	}
	account, err := r.accounts.StorageAccountCredentials(ctx, tenantID)
	if errors.Is(err, cloud.ErrCloudAccountNotConfigured) {
		if r.localDriverName == "" || expectedProject != "" {
			return nil, "", ErrDriverUnavailable
		}
		driver, err := r.registry.Driver(r.localDriverName)
		if err != nil {
			return nil, "", err
		}
		return driver, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("%w: read mapped storage account: %v", ErrDriverUnavailable, err)
	}
	if account == nil || expectedProject != "" && account.ScopeID() != expectedProject {
		return nil, "", ErrDriverUnavailable
	}
	name := r.registry.DefaultFactoryName()
	if name == "" {
		return nil, "", ErrDriverUnavailable
	}
	return r.driverForAccount(ctx, name, account, expectedProject)
}

func (r *ScopedDrivers) forVolume(ctx context.Context, volume models.StorageVolume) (BlockDriver, error) {
	if r == nil || r.accounts == nil || r.registry == nil {
		return nil, ErrDriverUnavailable
	}
	if driver, err := r.registry.Driver(volume.Driver); err == nil {
		if volume.ProviderProjectID != "" {
			return nil, ErrDriverUnavailable
		}
		return driver, nil
	}
	if volume.ProviderProjectID == "" {
		return nil, ErrDriverUnavailable
	}
	account, err := r.accounts.StorageAccountCredentials(ctx, volume.TenantID)
	if err != nil {
		return nil, ErrDriverUnavailable
	}
	driver, projectID, err := r.driverForAccount(
		ctx, volume.Driver, account, volume.ProviderProjectID,
	)
	if err != nil || projectID == "" || projectID != volume.ProviderProjectID {
		return nil, ErrDriverUnavailable
	}
	return driver, nil
}

func (r *ScopedDrivers) driverForAccount(
	ctx context.Context,
	name string,
	account cloud.CloudAccountCredentials,
	expectedProject string,
) (BlockDriver, string, error) {
	if account == nil || expectedProject != "" && account.ScopeID() != expectedProject {
		return nil, "", ErrDriverUnavailable
	}
	driver, err := r.registry.ForAccount(ctx, name, account)
	if err != nil {
		return nil, "", fmt.Errorf("%w: initialize scoped storage driver: %v", ErrDriverUnavailable, err)
	}
	if expectedProject != "" && driver.Name() != name {
		return nil, "", ErrDriverUnavailable
	}
	return driver, account.ScopeID(), nil
}

type cinderDriverFactory struct {
	config    CinderConfig
	newDriver func(CinderConfig) (BlockDriver, error)
}

func NewCinderDriverFactory(configs ...CinderConfig) ScopedBlockDriverFactory {
	var config CinderConfig
	if len(configs) > 0 {
		config = configs[0]
	}
	return &cinderDriverFactory{
		config: config,
		newDriver: func(config CinderConfig) (BlockDriver, error) {
			return NewCinderDriver(config)
		},
	}
}

func (f *cinderDriverFactory) Name() string { return "openstack-cinder" }

func (f *cinderDriverFactory) ForAccount(
	ctx context.Context,
	account cloud.CloudAccountCredentials,
) (BlockDriver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config := f.config
	credential, ok := account.(cloud.OpenStackAccountCredentials)
	if !ok {
		return nil, ErrDriverUnavailable
	}
	if err := credential.Validate(); err != nil {
		return nil, err
	}
	config.ProjectID = credential.ScopeID()
	config.ApplicationCredentialID = credential.CredentialID
	config.ApplicationCredentialSecret = credential.CredentialSecret
	config.Username, config.Password = "", ""
	if credential.RegionName() != "" {
		config.Region = credential.RegionName()
	}
	return f.newDriver(config)
}
