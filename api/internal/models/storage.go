package models

import "time"

type StorageVolume struct {
	ID               string            `gorm:"primaryKey;size:32" json:"id"`
	TenantID         uint              `gorm:"uniqueIndex:idx_storage_volume_idempotency;uniqueIndex:idx_storage_active_volume_name,where:deleted_at IS NULL;index;not null" json:"tenant_id"`
	ProjectID        uint              `gorm:"uniqueIndex:idx_storage_volume_idempotency;uniqueIndex:idx_storage_active_volume_name,where:deleted_at IS NULL;index;not null" json:"project_id"`
	OwnerID          uint              `gorm:"not null" json:"owner_id"`
	Name             string            `gorm:"size:63;uniqueIndex:idx_storage_active_volume_name,where:deleted_at IS NULL;not null" json:"name"`
	Description      string            `gorm:"size:512" json:"description,omitempty"`
	Region           string            `gorm:"size:64;not null" json:"region"`
	AvailabilityZone string            `gorm:"size:64" json:"availability_zone,omitempty"`
	StorageClass     string            `gorm:"size:32;not null" json:"storage_class"`
	SizeGB           int64             `gorm:"not null" json:"size_gb"`
	Encrypted        bool              `gorm:"not null" json:"encrypted"`
	DeleteProtection bool              `gorm:"not null" json:"delete_protection"`
	TagsJSON         []byte            `gorm:"type:jsonb" json:"-"`
	Tags             map[string]string `gorm:"-" json:"tags,omitempty"`
	State            string            `gorm:"size:32;index;not null" json:"state"`
	Driver           string            `gorm:"size:32;not null" json:"driver"`
	IdempotencyKey   string            `gorm:"uniqueIndex:idx_storage_volume_idempotency;size:128;not null" json:"-"`
	RequestHash      string            `gorm:"size:64;not null" json:"-"`
	ProviderID       string            `gorm:"size:255" json:"-"`
	ErrorCode        string            `gorm:"size:64" json:"error_code,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        *time.Time        `gorm:"index" json:"deleted_at,omitempty"`
}

type StorageQuota struct {
	ID            uint  `gorm:"primaryKey"`
	TenantID      uint  `gorm:"uniqueIndex:idx_storage_project_quota;not null"`
	ProjectID     uint  `gorm:"uniqueIndex:idx_storage_project_quota;not null"`
	MaxVolumes    int   `gorm:"not null;default:20"`
	MaxCapacityGB int64 `gorm:"not null;default:1024"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StorageOperation struct {
	ID             string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID       uint       `gorm:"uniqueIndex:idx_storage_operation_idempotency;index;not null" json:"tenant_id"`
	ProjectID      uint       `gorm:"uniqueIndex:idx_storage_operation_idempotency;index;not null" json:"project_id"`
	ResourceID     string     `gorm:"size:32;index;not null" json:"resource_id"`
	ResourceType   string     `gorm:"size:32;not null" json:"resource_type"`
	Action         string     `gorm:"size:32;not null" json:"action"`
	IdempotencyKey string     `gorm:"uniqueIndex:idx_storage_operation_idempotency;size:128;not null" json:"-"`
	ActorID        uint       `gorm:"not null" json:"-"`
	Status         string     `gorm:"size:32;index;not null" json:"status"`
	Attempts       int        `gorm:"not null;default:0" json:"attempts"`
	ErrorCode      string     `gorm:"size:64" json:"error_code,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type StorageAuditEvent struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TenantID     uint      `gorm:"index;not null" json:"tenant_id"`
	ProjectID    uint      `gorm:"index;not null" json:"project_id"`
	ResourceID   string    `gorm:"size:32;index;not null" json:"resource_id"`
	ResourceType string    `gorm:"size:32;not null" json:"resource_type"`
	OperationID  string    `gorm:"size:32;index;not null" json:"operation_id"`
	ActorID      uint      `gorm:"not null" json:"actor_id"`
	Source       string    `gorm:"size:32;not null" json:"source"`
	Action       string    `gorm:"size:64;not null" json:"action"`
	Result       string    `gorm:"size:32;not null" json:"result"`
	CreatedAt    time.Time `json:"created_at"`
}
