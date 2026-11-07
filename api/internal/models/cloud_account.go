package models

import "time"

type CloudAccount struct {
	ID                 uint      `gorm:"primaryKey" json:"-"`
	TenantID           uint      `gorm:"uniqueIndex;not null" json:"-"`
	OpenStackProjectID string    `gorm:"size:255;uniqueIndex;not null" json:"-"`
	CredentialID       string    `gorm:"size:255;not null" json:"-"`
	EncryptedSecret    []byte    `gorm:"type:bytea;not null" json:"-"`
	Region             string    `gorm:"size:64;not null" json:"-"`
	CreatedAt          time.Time `json:"-"`
	UpdatedAt          time.Time `json:"-"`
}
