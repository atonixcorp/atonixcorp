package controllers

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"atonixcorp/api/internal/observability"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var observabilityIDPattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]{1,128}$`)

type ObservabilityController struct {
	service *observability.Service
}

func NewObservabilityController(db *gorm.DB) *ObservabilityController {
	return &ObservabilityController{service: observability.NewService(db)}
}

func (c *ObservabilityController) Overview(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Overview(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Resources(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Resources(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Resource(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	resourceType, resourceID, ok := observabilityResourceID(ctx)
	if !ok {
		return
	}
	result, err := c.service.GetResource(ctx.Request.Context(), scope, resourceType, resourceID)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Metrics(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	resourceType, resourceID, ok := observabilityResourceID(ctx)
	if !ok {
		return
	}
	result, err := c.service.ResourceMetrics(ctx.Request.Context(), scope, resourceType, resourceID)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) MetricsSummary(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.MetricsSummary(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Integrations(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Integrations(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Events(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Events(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Logs(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Logs(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Alerts(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Alerts(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func (c *ObservabilityController) Dashboards(ctx *gin.Context) {
	scope, ok := observabilityScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Dashboards(ctx.Request.Context(), scope)
	if err != nil {
		writeObservabilityError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, result)
}

func observabilityScope(ctx *gin.Context) (observability.Scope, bool) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required.", "code": "AUTHENTICATION_REQUIRED"})
		return observability.Scope{}, false
	}
	tenantID, err := strconv.ParseUint(ctx.Param("tenantID"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant scope.", "code": "INVALID_SCOPE"})
		return observability.Scope{}, false
	}
	projectID, err := strconv.ParseUint(ctx.Param("projectID"), 10, 64)
	if err != nil || projectID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project scope.", "code": "INVALID_SCOPE"})
		return observability.Scope{}, false
	}
	return observability.Scope{UserID: userID, TenantID: uint(tenantID), ProjectID: uint(projectID)}, true
}

func observabilityResourceID(ctx *gin.Context) (string, string, bool) {
	resourceType := ctx.Param("resourceType")
	resourceID := ctx.Param("resourceID")
	if !observabilityIDPattern.MatchString(resourceType) || !observabilityIDPattern.MatchString(resourceID) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource identifier.", "code": "INVALID_RESOURCE"})
		return "", "", false
	}
	return resourceType, resourceID, true
}

func writeObservabilityError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, observability.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Access to this observability scope is denied.", "code": "FORBIDDEN"})
	case errors.Is(err, observability.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "The resource was not found in this project.", "code": "NOT_FOUND"})
	default:
		log.Printf("observability request failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "The observability request could not be completed.", "code": "OBSERVABILITY_REQUEST_FAILED"})
	}
}
