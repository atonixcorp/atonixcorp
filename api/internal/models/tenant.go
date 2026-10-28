package models

import "time"

type Tenant struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Slug      string    `gorm:"size:80;uniqueIndex;not null" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantMembership struct {
	ID        uint   `gorm:"primaryKey"`
	TenantID  uint   `gorm:"uniqueIndex:idx_tenant_user;not null"`
	UserID    uint   `gorm:"uniqueIndex:idx_tenant_user;not null"`
	Role      string `gorm:"size:32;not null"`
	CreatedAt time.Time
}

type Project struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"uniqueIndex:idx_tenant_project_slug;not null" json:"tenant_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Slug      string    `gorm:"size:80;uniqueIndex:idx_tenant_project_slug;not null" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectMembership struct {
	ID        uint   `gorm:"primaryKey"`
	ProjectID uint   `gorm:"uniqueIndex:idx_project_user;not null"`
	UserID    uint   `gorm:"uniqueIndex:idx_project_user;not null"`
	Role      string `gorm:"size:32;not null"`
	CreatedAt time.Time
}
