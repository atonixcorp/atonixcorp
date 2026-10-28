package models

import "time"

type ContainerDeploymentRevision struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	DeploymentID     string    `gorm:"size:32;index:idx_container_revision;not null" json:"deployment_id"`
	TenantID         uint      `gorm:"index;not null" json:"tenant_id"`
	ProjectID        uint      `gorm:"index;not null" json:"project_id"`
	Revision         uint      `gorm:"uniqueIndex:idx_container_revision;not null" json:"revision"`
	SpecJSON         []byte    `gorm:"type:jsonb;not null" json:"-"`
	EncryptedSecrets []byte    `gorm:"type:bytea" json:"-"`
	SpecHash         string    `gorm:"size:64;not null" json:"spec_hash"`
	ActorID          uint      `gorm:"not null" json:"actor_id"`
	CreatedAt        time.Time `json:"created_at"`
}
