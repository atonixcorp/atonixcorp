package models

import "time"

type PlatformAdminAuditEvent struct {
	ID           uint      `gorm:"primaryKey"`
	ActorID      uint      `gorm:"not null;index"`
	TargetUserID uint      `gorm:"not null;index"`
	Action       string    `gorm:"size:64;not null"`
	PreviousRole string    `gorm:"size:32;not null"`
	NewRole      string    `gorm:"size:32;not null"`
	CreatedAt    time.Time `gorm:"not null;index"`
}
