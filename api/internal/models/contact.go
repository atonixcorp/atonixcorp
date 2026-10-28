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

type SponsorshipInquiry struct {
	Name            string  `json:"name"`
	Company         string  `json:"company"`
	Email           string  `json:"email"`
	Phone           string  `json:"phone"`
	Country         string  `json:"country"`
	Amount          float64 `json:"amount"`
	Package         string  `json:"package"`
	PaymentProvider string  `json:"paymentProvider"`
	Message         string  `json:"message"`
}
