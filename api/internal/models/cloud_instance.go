package models

import "time"

type CloudInstance struct {
	ID                 string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID           uint       `gorm:"index;uniqueIndex:idx_cloud_instance_idempotency;uniqueIndex:idx_cloud_active_instance_name,where:deleted_at IS NULL;not null" json:"tenant_id"`
	ProjectID          uint       `gorm:"index;uniqueIndex:idx_cloud_instance_idempotency;uniqueIndex:idx_cloud_active_instance_name,where:deleted_at IS NULL;not null" json:"project_id"`
	OwnerID            uint       `gorm:"not null" json:"-"`
	Name               string     `gorm:"size:63;uniqueIndex:idx_cloud_active_instance_name,where:deleted_at IS NULL;not null" json:"name"`
	Provider           string     `gorm:"size:32;not null" json:"-"`
	ProviderInstanceID string     `gorm:"size:255;index" json:"-"`
	Region             string     `gorm:"size:64;not null" json:"-"`
	ImageID            string     `gorm:"size:255;not null" json:"-"`
	FlavorID           string     `gorm:"size:255;not null" json:"-"`
	NetworkID          string     `gorm:"size:255;not null" json:"-"`
	RegionResourceID   string     `gorm:"size:32" json:"region_id"`
	ImageResourceID    string     `gorm:"size:32" json:"image_id"`
	SizeResourceID     string     `gorm:"size:32" json:"size_id"`
	NetworkResourceID  string     `gorm:"size:32" json:"network_id"`
	KeyName            string     `gorm:"size:255" json:"key_name,omitempty"`
	Status             string     `gorm:"size:32;index;not null" json:"status"`
	ProviderStatus     string     `gorm:"size:32" json:"-"`
	PrivateIP          string     `gorm:"size:64" json:"private_ip,omitempty"`
	PublicIP           string     `gorm:"size:64" json:"public_ip,omitempty"`
	IdempotencyKey     string     `gorm:"uniqueIndex:idx_cloud_instance_idempotency;size:128;not null" json:"-"`
	RequestHash        string     `gorm:"size:64;not null" json:"-"`
	ErrorCode          string     `gorm:"size:64" json:"error_code,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}
