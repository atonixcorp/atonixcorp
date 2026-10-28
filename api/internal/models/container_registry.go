package models

import "time"

type ContainerRegistry struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TenantID          uint      `gorm:"uniqueIndex:idx_container_registry_scope;index;not null" json:"tenant_id"`
	ProjectID         uint      `gorm:"uniqueIndex:idx_container_registry_scope;index;not null" json:"project_id"`
	Name              string    `gorm:"size:100;uniqueIndex:idx_container_registry_scope;not null" json:"name"`
	URL               string    `gorm:"size:512;not null" json:"url"`
	Username          string    `gorm:"size:255" json:"-"`
	EncryptedPassword []byte    `gorm:"type:bytea" json:"-"`
	OwnerID           uint      `gorm:"index;not null" json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
