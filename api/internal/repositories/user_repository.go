package repositories

import (
	"atonixcorp/api/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		tenant := models.Tenant{
			Name: user.FullName + "'s workspace",
			Slug: fmt.Sprintf("tenant-%d", user.ID),
		}
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}

		project := models.Project{TenantID: tenant.ID, Name: "Default", Slug: "default"}
		if err := tx.Create(&project).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.TenantMembership{
			TenantID: tenant.ID, UserID: user.ID, Role: "owner",
		}).Error; err != nil {
			return err
		}

		return tx.Create(&models.ProjectMembership{
			ProjectID: project.ID, UserID: user.ID, Role: "owner",
		}).Error
	})
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
