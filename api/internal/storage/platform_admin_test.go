package storage

import (
	"context"
	"errors"
	"testing"

	"atonixcorp/api/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPlatformSuperAdminCanAccessValidCrossTenantProject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Tenant{}, &models.TenantMembership{},
		&models.Project{}, &models.ProjectMembership{},
	); err != nil {
		t.Fatal(err)
	}
	for _, record := range []any{
		&models.User{ID: 99, FullName: "Admin", Email: "admin@example.invalid", PlatformRole: models.PlatformRoleSuperAdmin},
		&models.Tenant{ID: 2, Name: "Other", Slug: "other"},
		&models.Project{ID: 2, TenantID: 2, Name: "Other project", Slug: "other-project"},
	} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(db, nil)
	if err := service.authorizeProject(context.Background(), 99, 2, 2, true); err != nil {
		t.Fatalf("super admin cross-tenant storage authorization: %v", err)
	}
	if err := service.authorizeProject(context.Background(), 7, 2, 2, false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ordinary user cross-tenant storage authorization error = %v, want ErrForbidden", err)
	}
}
