package storage

import (
	"context"
	"errors"

	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/models"
)

type AccountCredentialsSource interface {
	StorageAccountCredentials(context.Context, uint) (cloud.CloudAccountCredentials, error)
}

type ScopedDrivers struct {
	accounts  AccountCredentialsSource
	local     BlockDriver
	config    CinderConfig
	newCinder func(CinderConfig) (BlockDriver, error)
}

func NewScopedDrivers(accounts AccountCredentialsSource, local BlockDriver, config CinderConfig) *ScopedDrivers {
	return &ScopedDrivers{accounts: accounts, local: local, config: config,
		newCinder: func(cfg CinderConfig) (BlockDriver, error) { return NewCinderDriver(cfg) }}
}

func (r *ScopedDrivers) forTenant(ctx context.Context, tenantID uint) (BlockDriver, string, error) {
	return r.forTenantProject(ctx, tenantID, "")
}

func (r *ScopedDrivers) forTenantProject(ctx context.Context, tenantID uint, expectedProject string) (BlockDriver, string, error) {
	account, err := r.accounts.StorageAccountCredentials(ctx, tenantID)
	if errors.Is(err, cloud.ErrCloudAccountNotConfigured) {
		if r.local == nil || expectedProject != "" {
			return nil, "", ErrDriverUnavailable
		}
		return r.local, "", nil
	}
	if err != nil {
		return nil, "", ErrDriverUnavailable
	}
	if expectedProject != "" && account.ProjectID != expectedProject {
		return nil, "", ErrDriverUnavailable
	}
	cfg := r.config
	cfg.ProjectID = account.ProjectID
	cfg.ApplicationCredentialID = account.CredentialID
	cfg.ApplicationCredentialSecret = account.CredentialSecret
	cfg.Username, cfg.Password = "", ""
	if account.Region != "" {
		cfg.Region = account.Region
	}
	driver, err := r.newCinder(cfg)
	if err != nil || driver == nil {
		return nil, "", ErrDriverUnavailable
	}
	return driver, account.ProjectID, nil
}

func (r *ScopedDrivers) forVolume(ctx context.Context, volume models.StorageVolume) (BlockDriver, error) {
	if volume.Driver != "openstack-cinder" {
		if r.local == nil || r.local.Name() != volume.Driver {
			return nil, ErrDriverUnavailable
		}
		return r.local, nil
	}
	if volume.ProviderProjectID == "" {
		return nil, ErrDriverUnavailable
	}
	driver, projectID, err := r.forTenantProject(ctx, volume.TenantID, volume.ProviderProjectID)
	if err != nil || projectID == "" || projectID != volume.ProviderProjectID || driver.Name() != volume.Driver {
		return nil, ErrDriverUnavailable
	}
	return driver, nil
}
