package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"atonixcorp/api/internal/containers"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
)

type ContainerDeploymentController struct {
	service *containers.Service
}

func NewContainerDeploymentController(service *containers.Service) *ContainerDeploymentController {
	return &ContainerDeploymentController{service: service}
}

func (c *ContainerDeploymentController) Create(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var request struct {
		TenantID  uint                      `json:"tenant_id"`
		ProjectID uint                      `json:"project_id"`
		Spec      containers.DeploymentSpec `json:"spec"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deployment request"})
		return
	}
	result, err := c.service.Create(
		ctx.Request.Context(), userID, request.TenantID, request.ProjectID,
		ctx.GetHeader("Idempotency-Key"), ctx.GetHeader("X-Request-ID"), request.Spec,
	)
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func (c *ContainerDeploymentController) List(ctx *gin.Context) {
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
	result, err := c.service.List(ctx.Request.Context(), userID, tenantID, projectID)
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Get(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := c.service.Get(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Operate(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	operation := ctx.Param("operation")
	if ctx.Request.Method == http.MethodDelete {
		operation = "delete"
	}
	var request struct {
		Replicas int32 `json:"replicas"`
		Revision uint  `json:"revision"`
	}
	if operation == "scale" {
		if err := ctx.ShouldBindJSON(&request); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "scale request must include replicas"})
			return
		}
	} else if operation == "rollback" {
		if err := ctx.ShouldBindJSON(&request); err != nil || request.Revision == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "rollback request must include a revision"})
			return
		}
		result, err := c.service.Rollback(ctx.Request.Context(), userID, ctx.Param("deploymentID"), request.Revision)
		if err != nil {
			writeContainerError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, result)
		return
	}
	result, err := c.service.Operate(ctx.Request.Context(), userID, ctx.Param("deploymentID"), operation, request.Replicas)
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Events(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id := ctx.Param("deploymentID")
	audit, err := c.service.Events(ctx.Request.Context(), userID, id)
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	runtimeEvents, err := c.service.RuntimeEvents(ctx.Request.Context(), userID, id)
	if err != nil && !errors.Is(err, containers.ErrRuntimeUnavailable) {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"audit": audit, "runtime": runtimeEvents})
}

func (c *ContainerDeploymentController) Logs(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	logs, err := c.service.Logs(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(logs))
}

func (c *ContainerDeploymentController) Instances(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	pods, err := c.service.Instances(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	result := make([]gin.H, 0, len(pods))
	for _, pod := range pods {
		result = append(result, gin.H{
			"name": pod.Name, "phase": pod.Status.Phase,
			"node": pod.Spec.NodeName, "created_at": pod.CreationTimestamp,
			"restarts": restartCount(pod),
			"ready":    isPodReady(pod),
		})
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Configuration(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := c.service.Get(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"spec": result.Spec})
}

func (c *ContainerDeploymentController) Metrics(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := c.service.Metrics(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Patch(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	expectedRevision, err := strconv.ParseUint(strings.Trim(ctx.GetHeader("If-Match"), `"`), 10, 64)
	if err != nil || expectedRevision == 0 {
		ctx.JSON(http.StatusPreconditionRequired, gin.H{"error": "If-Match must contain the current deployment revision"})
		return
	}
	var request struct {
		Spec containers.DeploymentSpec `json:"spec"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid deployment specification"})
		return
	}
	result, err := c.service.Update(ctx.Request.Context(), userID, ctx.Param("deploymentID"), uint(expectedRevision), request.Spec)
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *ContainerDeploymentController) Revisions(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	result, err := c.service.Revisions(ctx.Request.Context(), userID, ctx.Param("deploymentID"))
	if err != nil {
		writeContainerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func parseID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id), nil
}

func writeContainerError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, containers.ErrMetricsUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Kubernetes metrics are unavailable on the configured cluster.", "code": "METRICS_UNAVAILABLE"})
	case errors.Is(err, containers.ErrRuntimeUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Kubernetes runtime is not configured.", "code": "RUNTIME_UNAVAILABLE"})
	case errors.Is(err, containers.ErrSecretKeyUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "SECRET_ENCRYPTION_UNAVAILABLE"})
	case errors.Is(err, containers.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "container deployment not found", "code": "NOT_FOUND"})
	case errors.Is(err, containers.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": "project access denied", "code": "FORBIDDEN"})
	case errors.Is(err, containers.ErrConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "IDEMPOTENCY_CONFLICT"})
	case errors.Is(err, containers.ErrNameConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "NAME_IN_USE"})
	case errors.Is(err, containers.ErrOperationConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "OPERATION_CONFLICT"})
	case errors.Is(err, containers.ErrInvalidRequest):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "code": "INVALID_DEPLOYMENT"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "container control plane request failed", "code": "CONTROL_PLANE_ERROR"})
	}
}

func restartCount(pod corev1.Pod) int32 {
	var count int32
	for _, status := range pod.Status.ContainerStatuses {
		count += status.RestartCount
	}
	return count
}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
