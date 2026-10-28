package controllers

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/services"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type signupCaptureService struct {
	user *models.User
}

func (s *signupCaptureService) Signup(user *models.User) error {
	copy := *user
	s.user = &copy
	return nil
}

func (*signupCaptureService) Login(string, string) (*models.User, error) {
	return nil, nil
}

func (*signupCaptureService) GetByID(uint) (*models.User, error) {
	return nil, nil
}

var _ services.AuthService = (*signupCaptureService)(nil)

func TestSignupBindsPasswordWithoutExposingIt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &signupCaptureService{}
	router := gin.New()
	router.POST("/signup", NewAuthController(service).Signup)
	password := "TestingPassword123456"
	body, err := json.Marshal(map[string]string{
		"fullName": "Samuel Orca",
		"email":    "orcaprojectstack@gmail.com",
		"password": password,
	})
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader(body)))

	if response.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if service.user == nil || service.user.Password != password {
		t.Fatalf("signup service did not receive password from request")
	}
	if bytes.Contains(response.Body.Bytes(), []byte(password)) {
		t.Fatal("signup response exposed the password")
	}
}
