package models

import "time"

type StorageObject struct {
	ID               string     `gorm:"primaryKey;size:36" json:"id"`
	Provider         string     `gorm:"size:32;not null" json:"provider"`
	Bucket           string     `gorm:"size:255;not null" json:"bucket"`
	ObjectKey        string     `gorm:"size:1024;not null;uniqueIndex" json:"object_key"`
	OwnerID          uint       `gorm:"index;not null" json:"owner_id"`
	TenantID         uint       `gorm:"index;not null" json:"tenant_id"`
	OriginalFilename string     `gorm:"size:255;not null" json:"original_filename"`
	ContentType      string     `gorm:"size:255;not null" json:"content_type"`
	SizeBytes        int64      `gorm:"not null" json:"size_bytes"`
	Checksum         string     `gorm:"size:64;not null" json:"checksum"`
	Category         string     `gorm:"size:32;not null" json:"category"`
	Visibility       string     `gorm:"size:32;not null" json:"visibility"`
	Status           string     `gorm:"size:32;not null;index" json:"status"`
	StagingKey       string     `gorm:"size:1024" json:"-"`
	UploadExpiresAt  *time.Time `json:"upload_expires_at,omitempty"`
	LegacyFileID     *uint      `gorm:"uniqueIndex" json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type ObjectAuditEvent struct {
	ID        uint   `gorm:"primaryKey"`
	ObjectID  string `gorm:"size:36;index;not null"`
	ActorID   uint   `gorm:"not null"`
	TenantID  uint   `gorm:"not null"`
	Action    string `gorm:"size:32;not null"`
	CreatedAt time.Time
}
