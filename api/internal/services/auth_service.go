package services

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Signup(user *models.User) error
	Login(email, password string) (*models.User, error)
	GetByID(id uint) (*models.User, error)
}

type authService struct {
	repo repositories.UserRepository
}

func NewAuthService(repo repositories.UserRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Signup(user *models.User) error {
	existing, err := s.repo.FindByEmail(user.Email)
	if err == nil && existing != nil {
		return errors.New("email already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("could not check account email")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user.Password = string(hashed)

	return s.repo.Create(user)
}

func (s *authService) Login(email, password string) (*models.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

func (s *authService) GetByID(id uint) (*models.User, error) {
	return s.repo.FindByID(id)
}
