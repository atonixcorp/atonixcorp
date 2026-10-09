package services

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"
)

type BlogService interface {
	Create(post *models.BlogPost) error
	Update(post *models.BlogPost) error
	Delete(id uint) error
	GetByID(id uint) (*models.BlogPost, error)
	GetAll() ([]models.BlogPost, error)
}

type blogService struct {
	repo repositories.BlogRepository
}

func NewBlogService(repo repositories.BlogRepository) BlogService {
	return &blogService{repo: repo}
}

func (s *blogService) Create(post *models.BlogPost) error {
	return s.repo.Create(post)
}

func (s *blogService) Update(post *models.BlogPost) error {
	return s.repo.Update(post)
}

func (s *blogService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *blogService) GetByID(id uint) (*models.BlogPost, error) {
	return s.repo.GetByID(id)
}

func (s *blogService) GetAll() ([]models.BlogPost, error) {
	return s.repo.GetAll()
}
