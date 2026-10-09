package repositories

import (
	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

type ContactRepository interface {
	Save(message *models.ContactMessage) error
	GetAll() ([]models.ContactMessage, error)
}

type contactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) ContactRepository {
	return &contactRepository{db: db}
}

func (r *contactRepository) Save(message *models.ContactMessage) error {
	return r.db.Create(message).Error
}

func (r *contactRepository) GetAll() ([]models.ContactMessage, error) {
	var messages []models.ContactMessage
	err := r.db.Order("created_at DESC").Find(&messages).Error
	return messages, err
}
