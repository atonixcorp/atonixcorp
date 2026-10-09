package services

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Signup(user *models.User) error
	Login(email, password string) (*models.User, error)
}

type authService struct {
	repo repositories.UserRepository
}

func NewAuthService(repo repositories.UserRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Signup(user *models.User) error {
	// Check if email already exists
	existing, _ := s.repo.FindByEmail(user.Email)
	if existing != nil {
		return errors.New("email already registered")
	}

	// Hash password
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
