package middleware

import (
	"net/http"
	"strings"

	"atonixcorp/api/internal/security/auth"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
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

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

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
