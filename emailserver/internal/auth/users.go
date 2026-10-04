package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string
	Email    string
	Password string
	Domain   string
}

type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	CreateUser(email, password, domain string) (*User, error)
	Authenticate(email, password string) (string, error)
}

type userStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) UserStore {
	return &userStore{db: db}
}

func (s *userStore) GetUserByEmail(email string) (*User, error) {
	row := s.db.QueryRow(`
        SELECT id, email, password, domain
        FROM users
        WHERE email = $1
    `, email)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Password, &u.Domain); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *userStore) CreateUser(email, password, domain string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var id string
	err = s.db.QueryRow(`
        INSERT INTO users (email, password, domain)
        VALUES ($1, $2, $3)
        RETURNING id
    `, email, string(hash), domain).Scan(&id)

	if err != nil {
		return nil, err
	}

	return &User{
		ID:       id,
		Email:    email,
		Password: string(hash),
		Domain:   domain,
	}, nil
}

// Authenticate verifies email + password and returns the user ID.
func (s *userStore) Authenticate(email, password string) (string, error) {
	u, err := s.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return "", errors.New("invalid credentials")
	}

	return u.ID, nil
}
