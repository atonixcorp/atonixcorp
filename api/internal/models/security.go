package models

import "time"

type SecurityPolicy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TenantID       uint      `gorm:"uniqueIndex:idx_security_policy_idempotency;index;not null" json:"tenant_id"`
	ProjectID      uint      `gorm:"uniqueIndex:idx_security_policy_idempotency;index;not null" json:"project_id"`
	OwnerID        uint      `gorm:"not null" json:"-"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	Type           string    `gorm:"size:32;not null" json:"type"`
	Description    string    `gorm:"size:1000;not null" json:"description"`
	RulesJSON      []byte    `gorm:"type:jsonb;not null" json:"-"`
	State          string    `gorm:"size:32;not null;index" json:"state"`
	Version        uint      `gorm:"not null;default:1" json:"version"`
	IdempotencyKey string    `gorm:"uniqueIndex:idx_security_policy_idempotency;size:128;not null" json:"-"`
	RequestHash    string    `gorm:"size:64;not null" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SecurityAuditEvent struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TenantID      uint      `gorm:"index;not null" json:"tenant_id"`
	ProjectID     uint      `gorm:"index;not null" json:"project_id"`
	ActorID       uint      `gorm:"index;not null" json:"actor_id"`
	Action        string    `gorm:"size:64;not null" json:"action"`
	ResourceType  string    `gorm:"size:32;not null" json:"resource_type"`
	ResourceID    string    `gorm:"size:64;not null" json:"resource_id"`
	Result        string    `gorm:"size:32;not null" json:"result"`
	Reason        string    `gorm:"size:128;not null" json:"reason"`
	RequestID     string    `gorm:"size:128;index;not null" json:"request_id"`
	CorrelationID string    `gorm:"size:128;index;not null" json:"correlation_id"`
	CreatedAt     time.Time `gorm:"index;not null" json:"created_at"`
}
