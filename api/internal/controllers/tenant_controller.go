package controllers

import (
	"atonixcorp/api/internal/models"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var tenantSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,78}[a-z0-9])?$`)

type TenantController struct {
	db *gorm.DB
}

func NewTenantController(db *gorm.DB) *TenantController {
	return &TenantController{db: db}
}

func (c *TenantController) List(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var tenants []models.Tenant
	err := c.db.Model(&models.Tenant{}).
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = tenants.id").
		Where("tenant_memberships.user_id = ?", userID).
		Order("tenants.id").
		Find(&tenants).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not load tenants"})
		return
	}
	ctx.JSON(http.StatusOK, tenants)
}

func (c *TenantController) Create(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var request struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant request"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Slug = strings.ToLower(strings.TrimSpace(request.Slug))
	if request.Name == "" || len(request.Name) > 255 || !tenantSlugPattern.MatchString(request.Slug) {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "tenant name and a valid slug are required"})
		return
	}

	tenant := models.Tenant{Name: request.Name, Slug: request.Slug}
	err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.TenantMembership{
			TenantID: tenant.ID, UserID: userID, Role: "owner",
		}).Error; err != nil {
			return err
		}
		project := models.Project{TenantID: tenant.ID, Name: "Default", Slug: "default"}
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		return tx.Create(&models.ProjectMembership{
			ProjectID: project.ID, UserID: userID, Role: "owner",
		}).Error
	})
	if err != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "tenant could not be created"})
		return
	}
	ctx.JSON(http.StatusCreated, tenant)
}

func (c *TenantController) ListProjects(ctx *gin.Context) {
	userID, tenantID, ok := c.authorizeTenant(ctx)
	if !ok {
		return
	}

	var projects []models.Project
	err := c.db.Model(&models.Project{}).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id").
		Where("projects.tenant_id = ? AND project_memberships.user_id = ?", tenantID, userID).
		Order("projects.id").
		Find(&projects).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not load projects"})
		return
	}
	ctx.JSON(http.StatusOK, projects)
}

func (c *TenantController) CreateProject(ctx *gin.Context) {
	userID, tenantID, ok := c.authorizeTenant(ctx)
	if !ok {
		return
	}
	var request struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project request"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Slug = strings.ToLower(strings.TrimSpace(request.Slug))
	if request.Name == "" || len(request.Name) > 255 || !tenantSlugPattern.MatchString(request.Slug) {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "project name and a valid slug are required"})
		return
	}
	if !c.hasTenantRole(tenantID, userID, "owner", "admin") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "tenant owner or admin role is required"})
		return
	}

	project := models.Project{TenantID: tenantID, Name: request.Name, Slug: request.Slug}
	err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		return tx.Create(&models.ProjectMembership{
			ProjectID: project.ID, UserID: userID, Role: "owner",
		}).Error
	})
	if err != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "project could not be created"})
		return
	}
	ctx.JSON(http.StatusCreated, project)
}

func (c *TenantController) authorizeTenant(ctx *gin.Context) (uint, uint, bool) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return 0, 0, false
	}
	tenantID, err := strconv.ParseUint(ctx.Param("tenantID"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return 0, 0, false
	}
	var membership models.TenantMembership
	err = c.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return 0, 0, false
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not authorize tenant access"})
		return 0, 0, false
	}
	return userID, uint(tenantID), true
}

func (c *TenantController) hasTenantRole(tenantID, userID uint, roles ...string) bool {
	var membership models.TenantMembership
	if err := c.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&membership).Error; err != nil {
		return false
	}
	for _, role := range roles {
		if membership.Role == role {
			return true
		}
	}
	return false
}

func authenticatedUserID(ctx *gin.Context) (uint, bool) {
	value, exists := ctx.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok && userID > 0
}
