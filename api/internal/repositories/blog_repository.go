package repositories

import (
	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

type BlogRepository interface {
	Create(post *models.BlogPost) error
	Update(post *models.BlogPost) error
	Delete(id uint) error
	GetByID(id uint) (*models.BlogPost, error)
	GetAll() ([]models.BlogPost, error)
}

type blogRepository struct {
	db *gorm.DB
}

func NewBlogRepository(db *gorm.DB) BlogRepository {
	return &blogRepository{db: db}
}

func (r *blogRepository) Create(post *models.BlogPost) error {
	return r.db.Create(post).Error
}

func (r *blogRepository) Update(post *models.BlogPost) error {
	return r.db.Save(post).Error
}

func (r *blogRepository) Delete(id uint) error {
	return r.db.Delete(&models.BlogPost{}, id).Error
}

func (r *blogRepository) GetByID(id uint) (*models.BlogPost, error) {
	var post models.BlogPost
	err := r.db.First(&post, id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *blogRepository) GetAll() ([]models.BlogPost, error) {
	var posts []models.BlogPost
	err := r.db.Order("created_at DESC").Find(&posts).Error
	return posts, err
}
