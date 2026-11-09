package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/models"

	"github.com/gin-gonic/gin"
)

var cloudInstanceIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type CloudInstanceController struct {
	service *cloud.Service
}

func NewCloudInstanceController(service *cloud.Service) *CloudInstanceController {
	return &CloudInstanceController{service: service}
}

func (c *CloudInstanceController) List(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, projectID, ok := cloudProjectScope(ctx)
	if !ok {
		return
	}
	instances, err := c.service.ListInstances(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	if instances == nil {
		instances = []models.CloudInstance{}
	}
	ctx.JSON(http.StatusOK, instances)
}

func (c *CloudInstanceController) ListOperations(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, projectID, ok := cloudProjectScope(ctx)
	if !ok {
		return
	}
	operations, err := c.service.ListOperations(
		ctx.Request.Context(), userID, tenantID, projectID,
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	if operations == nil {
		operations = []models.CloudOperation{}
	}
	ctx.JSON(http.StatusOK, operations)
}

func (c *CloudInstanceController) GetOperation(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("operationID")
	if !cloudInstanceIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid operation id"})
		return
	}
	operation, err := c.service.GetOperation(ctx.Request.Context(), userID, id)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, operation)
}

func (c *CloudInstanceController) Capabilities(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, err := strconv.ParseUint(ctx.Query("tenant_id"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}
	provisioningEnabled, _, services, err := c.service.TenantCapabilities(
		ctx.Request.Context(), userID, uint(tenantID),
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"organization_provisioning": provisioningEnabled,
		"services":                  services,
	})
}

func (c *CloudInstanceController) Catalog(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, projectID, ok := cloudProjectScope(ctx)
	if !ok {
		return
	}
	catalog, err := c.service.Catalog(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, catalog)
}

func (c *CloudInstanceController) ListSecurityGroups(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, projectID, ok := cloudProjectScope(ctx)
	if !ok {
		return
	}
	groups, err := c.service.ListSecurityGroups(
		ctx.Request.Context(), userID, tenantID, projectID,
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	if groups == nil {
		groups = []cloud.SecurityGroup{}
	}
	ctx.JSON(http.StatusOK, groups)
}

func (c *CloudInstanceController) CreateSecurityGroup(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tenantID, projectID, ok := cloudProjectScope(ctx)
	if !ok {
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var request cloud.CreateSecurityGroupRequest
	if err := decoder.Decode(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid security group request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid security group request"})
		return
	}
	group, err := c.service.CreateSecurityGroup(
		ctx.Request.Context(), userID, tenantID, projectID, request,
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, group)
}

func (c *CloudInstanceController) Create(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1<<20)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var request cloud.CreateInstanceRequest
	if err := decoder.Decode(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance request"})
		return
	}
	if request.TenantID == 0 || request.ProjectID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and project_id are required"})
		return
	}
	instance, err := c.service.CreateInstance(
		cloud.WithRequestID(ctx.Request.Context(), ctx.GetString("security_request_id")),
		userID, request.TenantID, request.ProjectID,
		ctx.GetHeader("Idempotency-Key"), request,
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, instance)
}

func (c *CloudInstanceController) Get(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("instanceID")
	if !cloudInstanceIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance id"})
		return
	}
	instance, err := c.service.GetInstance(ctx.Request.Context(), userID, id)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, instance)
}

func (c *CloudInstanceController) Operate(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("instanceID")
	if !cloudInstanceIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance id"})
		return
	}
	instance, err := c.service.OperateInstance(
		cloud.WithRequestID(ctx.Request.Context(), ctx.GetString("security_request_id")),
		userID, id, ctx.Param("action"),
	)
	if err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, instance)
}

func (c *CloudInstanceController) Delete(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("instanceID")
	if !cloudInstanceIDPattern.MatchString(id) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance id"})
		return
	}
	if err := c.service.DeleteInstance(
		cloud.WithRequestID(ctx.Request.Context(), ctx.GetString("security_request_id")),
		userID, id,
	); err != nil {
		writeCloudInstanceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"status": "deleting"})
}

func cloudProjectScope(ctx *gin.Context) (uint, uint, bool) {
	tenantID, err := strconv.ParseUint(ctx.Query("tenant_id"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return 0, 0, false
	}
	projectID, err := strconv.ParseUint(ctx.Query("project_id"), 10, 64)
	if err != nil || projectID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return 0, 0, false
	}
	return uint(tenantID), uint(projectID), true
}

func writeCloudInstanceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, cloud.ErrSecurityGroupUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Security groups are not available.",
			"code":  "SECURITY_GROUPS_UNAVAILABLE",
		})
	case errors.Is(err, cloud.ErrProviderUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Compute services are not available.", "code": "COMPUTE_UNAVAILABLE",
		})
	case errors.Is(err, cloud.ErrCloudAccountNotConfigured):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "This organization is not configured for cloud resources.",
			"code":  "CLOUD_ACCOUNT_NOT_CONFIGURED",
		})
	case errors.Is(err, cloud.ErrAccountEncryptionUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Compute services are not available.",
			"code":  "ACCOUNT_ENCRYPTION_UNAVAILABLE",
		})
	case errors.Is(err, cloud.ErrProjectProvisioningDisabled):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Compute provisioning is not enabled for this organization.",
			"code":  "PROJECT_PROVISIONING_DISABLED",
		})
	case errors.Is(err, cloud.ErrProviderRequest):
		log.Printf(
			"compute provider request failed request_id=%s: %v",
			ctx.GetString("security_request_id"), err,
		)
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": "The compute service could not complete the request.",
			"code":  "COMPUTE_REQUEST_FAILED",
		})
	case errors.Is(err, cloud.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": "project access denied", "code": "FORBIDDEN"})
	case errors.Is(err, cloud.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "instance not found", "code": "NOT_FOUND"})
	case errors.Is(err, cloud.ErrInvalid):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "The compute request is invalid.", "code": "INVALID_REQUEST",
		})
	case errors.Is(err, cloud.ErrConflict):
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "The requested operation conflicts with the current resource state.",
			"code":  "STATE_CONFLICT",
		})
	default:
		log.Printf("Compute instance request failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "the instance request could not be completed", "code": "INSTANCE_ERROR",
		})
	}
}
