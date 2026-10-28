package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"atonixcorp/api/internal/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SecurityController struct {
	service *security.Service
}

func NewSecurityController(db *gorm.DB) *SecurityController {
	return &SecurityController{service: security.NewService(db)}
}

func (c *SecurityController) Overview(ctx *gin.Context) {
	scope, requestContext, ok := c.requestScope(ctx)
	if !ok {
		return
	}
	result, err := c.service.Overview(ctx.Request.Context(), scope, requestContext)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *SecurityController) ListPolicies(ctx *gin.Context) {
	scope, requestContext, ok := c.requestScope(ctx)
	if !ok {
		return
	}
	limit, err := parseLimit(ctx.Query("limit"), 20)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	after, err := security.ParseCursor(ctx.Query("cursor"))
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	result, err := c.service.ListPolicies(
		ctx.Request.Context(), scope, requestContext, limit, after, ctx.Query("type"),
	)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *SecurityController) CreatePolicy(ctx *gin.Context) {
	scope, requestContext, ok := c.requestScope(ctx)
	if !ok {
		return
	}
	var request security.CreatePolicyRequest
	if err := decodeStrictJSON(ctx, &request, 32*1024); err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	if err := security.ValidatePolicyRequest(&request); err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	result, err := c.service.CreateDraft(
		ctx.Request.Context(), scope, requestContext, ctx.GetHeader("Idempotency-Key"), request,
	)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.Header("Location", "/api/v1/security/tenants/"+strconv.FormatUint(uint64(scope.TenantID), 10)+
		"/projects/"+strconv.FormatUint(uint64(scope.ProjectID), 10)+"/policies/"+strconv.FormatUint(uint64(result.ID), 10))
	ctx.JSON(http.StatusCreated, result)
}

func (c *SecurityController) GetPolicy(ctx *gin.Context) {
	scope, requestContext, policyID, ok := c.policyRequest(ctx)
	if !ok {
		return
	}
	result, err := c.service.GetPolicy(ctx.Request.Context(), scope, requestContext, policyID)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *SecurityController) SimulatePolicy(ctx *gin.Context) {
	scope, requestContext, policyID, ok := c.policyRequest(ctx)
	if !ok {
		return
	}
	if err := c.service.SimulatePolicy(ctx.Request.Context(), scope, requestContext, policyID); err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"status": "simulating"})
}

func (c *SecurityController) ListAudit(ctx *gin.Context) {
	scope, requestContext, ok := c.requestScope(ctx)
	if !ok {
		return
	}
	limit, err := parseLimit(ctx.Query("limit"), 50)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	after, err := security.ParseCursor(ctx.Query("cursor"))
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	sinceHours := 24
	if value := ctx.Query("since_hours"); value != "" {
		sinceHours, err = strconv.Atoi(value)
		if err != nil {
			writeSecurityError(ctx, security.ErrInvalid, requestContext.CorrelationID)
			return
		}
	}
	result, err := c.service.ListAudit(ctx.Request.Context(), scope, requestContext, limit, after, sinceHours)
	if err != nil {
		writeSecurityError(ctx, err, requestContext.CorrelationID)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *SecurityController) policyRequest(ctx *gin.Context) (security.Scope, security.RequestContext, uint, bool) {
	scope, requestContext, ok := c.requestScope(ctx)
	if !ok {
		return security.Scope{}, requestContext, 0, false
	}
	policyID, err := strconv.ParseUint(ctx.Param("policyID"), 10, 64)
	if err != nil || policyID == 0 {
		writeSecurityError(ctx, security.ErrInvalid, requestContext.CorrelationID)
		return security.Scope{}, requestContext, 0, false
	}
	return scope, requestContext, uint(policyID), true
}

func (c *SecurityController) requestScope(ctx *gin.Context) (security.Scope, security.RequestContext, bool) {
	requestContext := makeRequestContext(ctx)
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		writeSecurityError(ctx, security.ErrForbidden, requestContext.CorrelationID)
		return security.Scope{}, requestContext, false
	}
	tenantID, err := strconv.ParseUint(ctx.Param("tenantID"), 10, 64)
	if err != nil || tenantID == 0 {
		writeSecurityError(ctx, security.ErrInvalid, requestContext.CorrelationID)
		return security.Scope{}, requestContext, false
	}
	projectID, err := strconv.ParseUint(ctx.Param("projectID"), 10, 64)
	if err != nil || projectID == 0 {
		writeSecurityError(ctx, security.ErrInvalid, requestContext.CorrelationID)
		return security.Scope{}, requestContext, false
	}
	return security.Scope{UserID: userID, TenantID: uint(tenantID), ProjectID: uint(projectID)}, requestContext, true
}

func makeRequestContext(ctx *gin.Context) security.RequestContext {
	requestID := ctx.GetString("security_request_id")
	correlationID := ctx.GetString("security_correlation_id")
	return security.RequestContext{RequestID: requestID, CorrelationID: correlationID}
}

func decodeStrictJSON(ctx *gin.Context, destination any, maxBytes int64) error {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return security.ErrTooLarge
		}
		return security.ErrInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return security.ErrTooLarge
		}
		return security.ErrInvalid
	}
	return nil
}

func parseLimit(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > 100 {
		return 0, security.ErrInvalid
	}
	return limit, nil
}

func writeSecurityError(ctx *gin.Context, err error, correlationID string) {
	status := http.StatusInternalServerError
	code := "SECURITY_REQUEST_FAILED"
	message := "The security request could not be completed."
	switch {
	case errors.Is(err, security.ErrInvalid):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "The security request is invalid."
	case errors.Is(err, security.ErrTooLarge):
		status, code, message = http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The security request exceeds the allowed size."
	case errors.Is(err, security.ErrForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "Access to this security scope is denied."
	case errors.Is(err, security.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "The security resource was not found."
	case errors.Is(err, security.ErrConflict):
		status, code, message = http.StatusConflict, "STATE_CONFLICT", "The request conflicts with an existing security record."
	case errors.Is(err, security.ErrSimulationUnavailable):
		status, code, message = http.StatusServiceUnavailable, "SIMULATION_UNAVAILABLE", "No authorized policy simulation engine is configured."
	}
	ctx.JSON(status, gin.H{
		"error": message, "code": code, "correlation_id": correlationID,
	})
}
