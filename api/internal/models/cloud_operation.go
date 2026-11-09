package models

import "time"

type CloudOperation struct {
	ID                         string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID                   uint       `gorm:"index;not null" json:"tenant_id"`
	ProjectID                  uint       `gorm:"index;not null" json:"project_id"`
	ResourceID                 string     `gorm:"size:32;index;not null" json:"resource_id"`
	ResourceType               string     `gorm:"size:32;not null" json:"resource_type"`
	Operation                  string     `gorm:"size:32;not null" json:"operation"`
	Provider                   string     `gorm:"size:32;not null" json:"-"`
	ProviderOperationID        string     `gorm:"size:255" json:"-"`
	ProviderStatusAtStart      string     `gorm:"size:32" json:"-"`
	ProviderTransitionObserved bool       `gorm:"not null;default:false" json:"-"`
	Status                     string     `gorm:"size:32;index;not null" json:"status"`
	ErrorCode                  string     `gorm:"size:64" json:"error_code,omitempty"`
	RequestID                  string     `gorm:"size:128;index" json:"request_id,omitempty"`
	ActorID                    uint       `gorm:"not null" json:"-"`
	StartedAt                  time.Time  `gorm:"index;not null" json:"started_at"`
	CompletedAt                *time.Time `json:"completed_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type CloudAuditEvent struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TenantID     uint      `gorm:"index;not null" json:"tenant_id"`
	ProjectID    uint      `gorm:"index;not null" json:"project_id"`
	ActorID      uint      `gorm:"index;not null" json:"actor_id"`
	ResourceID   string    `gorm:"size:32;index;not null" json:"resource_id"`
	ResourceType string    `gorm:"size:32;not null" json:"resource_type"`
	OperationID  string    `gorm:"size:32;index;not null" json:"operation_id"`
	Action       string    `gorm:"size:64;not null" json:"action"`
	Result       string    `gorm:"size:32;not null" json:"result"`
	RequestID    string    `gorm:"size:128;index" json:"request_id,omitempty"`
	CreatedAt    time.Time `gorm:"index;not null" json:"created_at"`
}
