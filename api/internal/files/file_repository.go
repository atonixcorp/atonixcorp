package files

import (
	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db}
}

func (r *FileRepository) Save(file *models.File) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) GetByID(id uint) (*models.File, error) {
	var file models.File
	err := r.db.First(&file, id).Error
	return &file, err
}
