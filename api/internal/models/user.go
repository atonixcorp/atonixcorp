package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	FullName  string `gorm:"size:255" json:"fullName"`
	Email     string `gorm:"uniqueIndex" json:"email"`
	Password  string `gorm:"size:255" json:"-"`
	CreatedAt time.Time
}
