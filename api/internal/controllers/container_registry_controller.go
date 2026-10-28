package controllers

import (
	"errors"
	"net/http"

	"atonixcorp/api/internal/containers"
	"github.com/gin-gonic/gin"
)

type ContainerRegistryController struct {
	service *containers.Service
}

func NewContainerRegistryController(service *containers.Service) *ContainerRegistryController {
	return &ContainerRegistryController{service: service}
}

func (c *ContainerRegistryController) List(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, err := parseID(ctx.Query("tenant_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	projectID, err := parseID(ctx.Query("project_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}
	result, err := c.service.ListRegistries(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeRegistryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerRegistryController) Create(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var request containers.RegistryRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry request"})
		return
	}
	result, err := c.service.CreateRegistry(ctx.Request.Context(), userID, request)
	if err != nil {
		writeRegistryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, result)
}

func writeRegistryError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, containers.ErrSecretKeyUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Registry credential encryption is not configured.", "code": "SECRET_ENCRYPTION_UNAVAILABLE"})
	case errors.Is(err, containers.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": "project access denied", "code": "FORBIDDEN"})
	case errors.Is(err, containers.ErrInvalidRequest):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "code": "INVALID_REGISTRY"})
	case errors.Is(err, containers.ErrRegistryConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": "a registry with this name already exists in the project", "code": "REGISTRY_EXISTS"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "registry request failed", "code": "REGISTRY_ERROR"})
	}
}
