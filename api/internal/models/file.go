package models

import "time"

type File struct {
	ID        uint   `gorm:"primaryKey"`
	FileName  string `gorm:"not null"`
	FilePath  string `gorm:"not null"`
	Size      int64  `gorm:"not null"`
	MimeType  string `gorm:"not null"`
	CreatedAt time.Time
}
