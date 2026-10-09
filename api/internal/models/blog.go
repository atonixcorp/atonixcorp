package models

import "time"

type BlogPost struct {
	ID        uint   `gorm:"primaryKey"`
	Title     string `gorm:"size:255"`
	Content   string `gorm:"type:text"`
	AuthorID  uint
	CreatedAt time.Time
	UpdatedAt time.Time
}
