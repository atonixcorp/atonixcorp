package models

import "time"

type ContainerDeployment struct {
	ID                string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID          uint       `gorm:"index;uniqueIndex:idx_container_deployment_idempotency;uniqueIndex:idx_active_container_name,where:deleted_at IS NULL;not null" json:"tenant_id"`
	ProjectID         uint       `gorm:"index;uniqueIndex:idx_active_container_name,where:deleted_at IS NULL;not null" json:"project_id"`
	OwnerID           uint       `gorm:"index;not null" json:"owner_id"`
	Name              string     `gorm:"size:63;uniqueIndex:idx_active_container_name,where:deleted_at IS NULL;not null" json:"name"`
	State             string     `gorm:"size:32;index;not null" json:"state"`
	CurrentRevision   uint       `gorm:"not null;default:1" json:"current_revision"`
	DesiredReplicas   int32      `gorm:"not null" json:"desired_replicas"`
	SpecJSON          []byte     `gorm:"type:jsonb;not null" json:"-"`
	EncryptedSecrets  []byte     `gorm:"type:bytea" json:"-"`
	IdempotencyKey    string     `gorm:"size:128;uniqueIndex:idx_container_deployment_idempotency;not null" json:"-"`
	RequestHash       string     `gorm:"size:64;not null" json:"-"`
	CorrelationID     string     `gorm:"size:64;index;not null" json:"correlation_id"`
	ProvisioningJobID string     `gorm:"size:32;index;not null" json:"provisioning_job_id"`
	Attempts          int        `gorm:"not null;default:0" json:"-"`
	Endpoint          string     `gorm:"size:512" json:"endpoint,omitempty"`
	ErrorCode         string     `gorm:"size:64" json:"error_code,omitempty"`
	ErrorDetails      string     `gorm:"type:text" json:"error_details,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type ContainerDeploymentEvent struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	DeploymentID      string    `gorm:"size:32;index;not null" json:"deployment_id"`
	TenantID          uint      `gorm:"index;not null" json:"tenant_id"`
	ProjectID         uint      `gorm:"index;not null" json:"project_id"`
	PreviousState     string    `gorm:"size:32" json:"previous_state,omitempty"`
	NewState          string    `gorm:"size:32;not null" json:"new_state"`
	Reason            string    `gorm:"size:255;not null" json:"reason"`
	ActorID           uint      `gorm:"not null" json:"actor_id"`
	CorrelationID     string    `gorm:"size:64;index;not null" json:"correlation_id"`
	ProvisioningJobID string    `gorm:"size:32;not null" json:"provisioning_job_id"`
	ErrorCode         string    `gorm:"size:64" json:"error_code,omitempty"`
	ErrorDetails      string    `gorm:"type:text" json:"error_details,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
