package controllers

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/security/auth"
	"atonixcorp/api/internal/services"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	service services.AuthService
}

func NewAuthController(service services.AuthService) *AuthController {
	return &AuthController{service: service}
}

func (c *AuthController) Signup(ctx *gin.Context) {
	var payload struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid signup request"})
		return
	}
	user := models.User{
		FullName: strings.TrimSpace(payload.FullName),
		Email:    strings.ToLower(strings.TrimSpace(payload.Email)),
		Password: payload.Password,
	}
	email, emailErr := mail.ParseAddress(user.Email)
	if user.FullName == "" || len(user.FullName) > 255 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "enter a name up to 255 characters"})
		return
	}
	if emailErr != nil || email.Address != user.Email {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "enter a valid email address"})
		return
	}
	if len(user.Password) < 12 || len(user.Password) > 72 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "use a password between 12 and 72 bytes"})
		return
	}

	if err := c.service.Signup(&user); err != nil {
		if err.Error() == "email already registered" {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "account could not be created"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "signup successful"})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid login request"})
		return
	}

	user, err := c.service.Login(strings.ToLower(strings.TrimSpace(payload.Email)), payload.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := auth.GenerateToken(os.Getenv("JWT_SECRET"), user.ID, "user")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"fullName": user.FullName,
			"email":    user.Email,
		},
	})
}

func (c *AuthController) Session(ctx *gin.Context) {
	value, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID, ok := value.(uint)
	if !ok || userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	user, err := c.service.GetByID(userID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"fullName": user.FullName,
			"email":    user.Email,
		},
	})
}
