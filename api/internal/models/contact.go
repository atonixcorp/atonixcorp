package models

import "time"

type ContactMessage struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:255"`
	Email     string `gorm:"size:255"`
	Subject   string `gorm:"size:255"`
	Message   string `gorm:"type:text"`
	CreatedAt time.Time
}
