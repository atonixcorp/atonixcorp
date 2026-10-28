package models

import "time"

type SupportTicket struct {
	ID                uint      `gorm:"primaryKey" json:"-"`
	Reference         string    `gorm:"size:32;uniqueIndex" json:"reference"`
	UserID            uint      `gorm:"not null;index" json:"-"`
	Name              string    `gorm:"size:255;not null" json:"-"`
	Email             string    `gorm:"size:255;not null" json:"-"`
	Category          string    `gorm:"size:50;not null" json:"category"`
	Priority          string    `gorm:"size:20;not null" json:"priority"`
	Subject           string    `gorm:"size:255;not null" json:"subject"`
	Description       string    `gorm:"type:text;not null" json:"-"`
	ResourceReference string    `gorm:"size:255" json:"-"`
	Status            string    `gorm:"size:30;not null;default:open" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
