package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PlatformAdminController struct {
	db *gorm.DB
}

func NewPlatformAdminController(db *gorm.DB) *PlatformAdminController {
	return &PlatformAdminController{db: db}
}

func (c *PlatformAdminController) ListUsers(ctx *gin.Context) {
	var users []models.User
	if err := c.db.WithContext(ctx.Request.Context()).
		Select("id", "full_name", "email", "platform_role", "created_at").
		Order("id ASC").Limit(500).Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not load platform users"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

func (c *PlatformAdminController) UpdateUserRole(ctx *gin.Context) {
	targetID, err := strconv.ParseUint(ctx.Param("userID"), 10, 64)
	if err != nil || targetID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	actorID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var request struct {
		PlatformRole string `json:"platformRole"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid platform role request"})
		return
	}
	err = services.UpdatePlatformRole(c.db, actorID, uint(targetID), request.PlatformRole)
	switch {
	case err == nil:
		ctx.JSON(http.StatusOK, gin.H{"message": "platform role updated"})
	case errors.Is(err, services.ErrPlatformRoleInvalid):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrLastSuperAdmin):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPlatformAdminRequired):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "platform role could not be updated"})
	}
}
