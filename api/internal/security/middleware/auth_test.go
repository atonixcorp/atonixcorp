package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/security/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPlatformRoleIsLoadedFromDatabaseAndCanBeRevokedImmediately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open auth test database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate auth test database: %v", err)
	}
	user := models.User{FullName: "Administrator", Email: "admin@example.invalid", PlatformRole: models.PlatformRoleSuperAdmin}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create auth test user: %v", err)
	}
	const secret = "test-jwt-secret-with-sufficient-length"
	token, err := auth.GenerateToken(secret, user.ID, models.PlatformRoleUser)
	if err != nil {
		t.Fatalf("generate auth token: %v", err)
	}

	router := gin.New()
	router.GET("/admin", AuthMiddleware(secret, db), RequirePlatformSuperAdmin(), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("DB super admin with stale ordinary token role got %d, want 204", response.Code)
	}

	if err := db.Model(&user).Update("platform_role", models.PlatformRoleUser).Error; err != nil {
		t.Fatalf("revoke platform role: %v", err)
	}
	request = httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("revoked admin token got %d, want 403", response.Code)
	}
}
