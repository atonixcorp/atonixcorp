package middleware

import (
	"errors"
	"net/http"
	"strings"

	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/security/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(jwtSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeAuthenticationFailure(c, "missing token")
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")

		claims, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			writeAuthenticationFailure(c, "invalid token")
			return
		}

		var user models.User
		err = db.WithContext(c.Request.Context()).
			Select("id", "platform_role").First(&user, claims.UserID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeAuthenticationFailure(c, "invalid token")
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication service unavailable"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("platform_role", user.PlatformRole)

		c.Next()
	}
}

func RequirePlatformSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("platform_role") != models.PlatformRoleSuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "platform super admin permission is required"})
			return
		}
		c.Next()
	}
}

func writeAuthenticationFailure(c *gin.Context, message string) {
	response := gin.H{"error": message}
	if correlationID := c.GetString("security_correlation_id"); correlationID != "" {
		response["correlation_id"] = correlationID
		c.Header("Cache-Control", "no-store")
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, response)
}
