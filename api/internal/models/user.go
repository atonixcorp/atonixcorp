package models

import "time"

const (
	PlatformRoleUser       = "user"
	PlatformRoleSuperAdmin = "super_admin"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	FullName     string    `gorm:"size:255" json:"fullName"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	Password     string    `gorm:"size:255" json:"-"`
	PlatformRole string    `gorm:"size:32;not null;default:user" json:"platformRole"`
	CreatedAt    time.Time `json:"createdAt"`
}
