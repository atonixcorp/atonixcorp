package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	FullName  string `gorm:"size:255"`
	Email     string `gorm:"uniqueIndex"`
	Password  string `gorm:"size:255"`
	CreatedAt time.Time
}
