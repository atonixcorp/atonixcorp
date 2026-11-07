package controllers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"atonixcorp/api/internal/cloud"

	"github.com/gin-gonic/gin"
)

type CloudAccountController struct {
	service *cloud.Service
}

func NewCloudAccountController(service *cloud.Service) *CloudAccountController {
	return &CloudAccountController{service: service}
}

func (c *CloudAccountController) SetOrganizationAccount(ctx *gin.Context) {
	expectedToken := strings.TrimSpace(os.Getenv("OPENSTACK_ACCOUNT_MANAGEMENT_TOKEN"))
	if len(expectedToken) < 32 {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OpenStack account management is not configured.",
			"code":  "ACCOUNT_MANAGEMENT_UNAVAILABLE",
		})
		return
	}
	secure, err := accountManagementRequestIsTLS(ctx.Request)
	if err != nil {
		log.Printf("OpenStack account-management proxy configuration is invalid: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OpenStack account management is not configured.",
			"code":  "ACCOUNT_MANAGEMENT_UNAVAILABLE",
		})
		return
	}
	if !secure {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "OpenStack account credentials must be submitted over TLS.",
			"code":  "TRANSPORT_SECURITY_REQUIRED",
		})
		return
	}
	authorization := ctx.GetHeader("Authorization")
	providedToken := ""
	if scheme, token, ok := strings.Cut(authorization, " "); ok &&
		strings.EqualFold(scheme, "Bearer") && token != "" && !strings.ContainsAny(token, " \t\r\n") {
		providedToken = token
	}
	if len(providedToken) != len(expectedToken) ||
		subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Operator authentication required.", "code": "AUTHENTICATION_REQUIRED",
		})
		return
	}

	tenantID, err := strconv.ParseUint(ctx.Param("tenantID"), 10, 64)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID", "code": "INVALID_ID"})
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16<<10)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var account cloud.CloudAccountCredentials
	if err := decoder.Decode(&account); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cloud account request", "code": "INVALID_REQUEST"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cloud account request", "code": "INVALID_REQUEST"})
		return
	}

	if err := c.service.SetCloudAccount(ctx.Request.Context(), uint(tenantID), account); err != nil {
		switch {
		case errors.Is(err, cloud.ErrAccountEncryptionUnavailable):
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "OpenStack account encryption is not configured.",
				"code":  "ACCOUNT_ENCRYPTION_UNAVAILABLE",
			})
		case errors.Is(err, cloud.ErrProviderUnavailable):
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "The OpenStack Compute service is unavailable.",
				"code":  "COMPUTE_UNAVAILABLE",
			})
		case errors.Is(err, cloud.ErrInvalid):
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "The OpenStack account details are invalid.", "code": "INVALID_ACCOUNT",
			})
		case errors.Is(err, cloud.ErrNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"error": "organization not found", "code": "NOT_FOUND"})
		case errors.Is(err, cloud.ErrProviderRequest):
			ctx.JSON(http.StatusBadGateway, gin.H{
				"error": "The OpenStack account could not be validated.", "code": "ACCOUNT_VALIDATION_FAILED",
			})
		case errors.Is(err, cloud.ErrConflict):
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "The OpenStack project is already mapped, or this organization has existing compute resources.",
				"code":  "PROJECT_MAPPING_CONFLICT",
			})
		default:
			log.Printf("OpenStack account mapping update failed for organization %d: %v", tenantID, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "The OpenStack account mapping could not be saved.", "code": "ACCOUNT_MAPPING_ERROR",
			})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "configured"})
}

func accountManagementRequestIsTLS(request *http.Request) (bool, error) {
	if request.TLS != nil {
		return true, nil
	}
	if strings.EqualFold(
		strings.TrimSpace(os.Getenv("OPENSTACK_ACCOUNT_MANAGEMENT_ALLOW_LOCAL_HTTP")), "true",
	) {
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		if err == nil {
			remoteIP := net.ParseIP(host)
			if remoteIP != nil && remoteIP.IsLoopback() {
				return true, nil
			}
		}
	}
	forwardedProto := request.Header.Values("X-Forwarded-Proto")
	if len(forwardedProto) != 1 || forwardedProto[0] != "https" {
		return false, nil
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return false, nil
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return false, nil
	}
	configuredProxies := strings.TrimSpace(os.Getenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS"))
	if configuredProxies == "" {
		return false, nil
	}
	for _, entry := range strings.Split(configuredProxies, ",") {
		cidr := strings.TrimSpace(entry)
		if cidr == "" {
			return false, errors.New("trusted proxy CIDR list contains an empty entry")
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return false, errors.New("trusted proxy CIDR list contains an invalid CIDR")
		}
		if network.Contains(remoteIP) {
			return true, nil
		}
	}
	return false, nil
}
