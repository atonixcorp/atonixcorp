package repositories

import (
	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

type ContactRepository interface {
	Save(message *models.ContactMessage) error
	GetAll() ([]models.ContactMessage, error)
	SaveSupportTicket(ticket *models.SupportTicket) error
	ListSupportTickets(userID uint) ([]models.SupportTicket, error)
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

func (r *contactRepository) SaveSupportTicket(ticket *models.SupportTicket) error {
	return r.db.Create(ticket).Error
}

func (r *contactRepository) ListSupportTickets(userID uint) ([]models.SupportTicket, error) {
	var tickets []models.SupportTicket
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&tickets).Error
	return tickets, err
}
