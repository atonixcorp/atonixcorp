package models

import "time"

type ObservabilityResource struct {
	ID               uint      `gorm:"primaryKey"`
	TenantID         uint      `gorm:"uniqueIndex:idx_observability_resource;index;not null"`
	ProjectID        uint      `gorm:"uniqueIndex:idx_observability_resource;index;not null"`
	ResourceType     string    `gorm:"uniqueIndex:idx_observability_resource;size:64;not null"`
	ResourceID       string    `gorm:"uniqueIndex:idx_observability_resource;size:128;not null"`
	Provider         string    `gorm:"size:64;not null"`
	Name             string    `gorm:"size:255;not null"`
	Region           string    `gorm:"size:128"`
	ResourceStatus   string    `gorm:"size:64;not null"`
	MonitoringStatus string    `gorm:"size:32;not null;default:not_configured"`
	FirstSeenAt      time.Time `gorm:"not null"`
	LastSeenAt       time.Time `gorm:"not null;index"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
