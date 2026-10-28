package controllers

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"atonixcorp/api/internal/storage"
	"github.com/gin-gonic/gin"
)

var storageIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type StorageController struct {
	service *storage.Service
}

func NewStorageController(service *storage.Service) *StorageController {
	return &StorageController{service: service}
}

func (c *StorageController) VolumeTypes(ctx *gin.Context) {
	types, err := c.service.VolumeTypes(ctx.Request.Context())
	if err != nil {
		log.Printf("load storage volume types: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage volume types are unavailable"})
		return
	}
	ctx.JSON(http.StatusOK, types)
}

func (c *StorageController) Overview(ctx *gin.Context) {
	userID, tenantID, projectID, ok := c.projectScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.GetOverview(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *StorageController) ListVolumes(ctx *gin.Context) {
	userID, tenantID, projectID, ok := c.projectScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.ListVolumes(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *StorageController) CreateVolume(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var request storage.CreateVolumeRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid volume request"})
		return
	}
	result, err := c.service.CreateVolume(
		ctx.Request.Context(), userID, ctx.GetHeader("Idempotency-Key"), request,
	)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func (c *StorageController) GetVolume(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("id")
	if !storageIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid volume id"})
		return
	}
	result, err := c.service.GetVolume(ctx.Request.Context(), userID, id)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *StorageController) DeleteVolume(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("id")
	if !storageIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid volume id"})
		return
	}
	result, err := c.service.DeleteVolume(
		ctx.Request.Context(), userID, id, ctx.GetHeader("Idempotency-Key"),
	)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func (c *StorageController) GetOperation(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("id")
	if !storageIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid operation id"})
		return
	}
	result, err := c.service.GetOperation(ctx.Request.Context(), userID, id)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *StorageController) ListOperations(ctx *gin.Context) {
	userID, tenantID, projectID, ok := c.projectScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.ListOperations(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeStorageError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *StorageController) projectScope(ctx *gin.Context) (uint, uint, uint, bool) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return 0, 0, 0, false
	}
	tenantID, err := strconv.ParseUint(ctx.Query("tenant_id"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return 0, 0, 0, false
	}
	projectID, err := strconv.ParseUint(ctx.Query("project_id"), 10, 64)
	if err != nil || projectID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return 0, 0, 0, false
	}
	return userID, uint(tenantID), uint(projectID), true
}

func writeStorageError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, storage.ErrDriverUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Storage provisioning is not configured.", "code": "STORAGE_DRIVER_UNAVAILABLE"})
	case errors.Is(err, storage.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": "project access denied", "code": "FORBIDDEN"})
	case errors.Is(err, storage.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "storage resource not found", "code": "NOT_FOUND"})
	case errors.Is(err, storage.ErrQuota):
		ctx.JSON(http.StatusConflict, gin.H{"error": "project storage quota would be exceeded", "code": "QUOTA_EXCEEDED"})
	case errors.Is(err, storage.ErrConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "STATE_CONFLICT"})
	case errors.Is(err, storage.ErrInvalid):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "code": "INVALID_REQUEST"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "storage request failed", "code": "STORAGE_ERROR"})
	}
}
