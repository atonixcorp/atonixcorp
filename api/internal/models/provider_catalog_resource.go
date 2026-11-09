package models

import "time"

type ProviderCatalogResource struct {
	ID                 string    `gorm:"primaryKey;size:32"`
	TenantID           uint      `gorm:"uniqueIndex:idx_provider_catalog_identity,priority:1;not null"`
	ProjectID          uint      `gorm:"uniqueIndex:idx_provider_catalog_identity,priority:2;not null"`
	Provider           string    `gorm:"uniqueIndex:idx_provider_catalog_identity,priority:3;size:32;not null"`
	ResourceType       string    `gorm:"uniqueIndex:idx_provider_catalog_identity,priority:4;size:32;not null"`
	ProviderResourceID string    `gorm:"uniqueIndex:idx_provider_catalog_identity,priority:5;size:255;not null"`
	Name               string    `gorm:"size:255"`
	Active             bool      `gorm:"index;not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
