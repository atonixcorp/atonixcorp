package services

import (
	"errors"
	"testing"

	"atonixcorp/api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPlatformAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open platform admin test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Tenant{}, &models.TenantMembership{},
		&models.Project{}, &models.ProjectMembership{}, &models.PlatformAdminAuditEvent{},
	); err != nil {
		t.Fatalf("migrate platform admin test database: %v", err)
	}
	return db
}

func TestBootstrapSuperAdminIsOneTimeAndCreatesHashedAccount(t *testing.T) {
	db := newPlatformAdminTestDB(t)
	if err := BootstrapSuperAdmin(db, "  Platform Admin ", "ADMIN@example.invalid", "sufficiently-long-password"); err != nil {
		t.Fatalf("bootstrap super admin: %v", err)
	}

	var admin models.User
	if err := db.Where("email = ?", "admin@example.invalid").First(&admin).Error; err != nil {
		t.Fatalf("load bootstrapped admin: %v", err)
	}
	if admin.PlatformRole != models.PlatformRoleSuperAdmin {
		t.Fatalf("role = %q, want super_admin", admin.PlatformRole)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("sufficiently-long-password")); err != nil {
		t.Fatal("bootstrapped password was not stored as a valid bcrypt hash")
	}
	if err := BootstrapSuperAdmin(db, "Second Admin", "second@example.invalid", "sufficiently-long-password"); !errors.Is(err, ErrSuperAdminAlreadyExists) {
		t.Fatalf("second bootstrap error = %v, want ErrSuperAdminAlreadyExists", err)
	}
	var auditCount int64
	if err := db.Model(&models.PlatformAdminAuditEvent{}).Count(&auditCount).Error; err != nil || auditCount != 1 {
		t.Fatalf("bootstrap audit count = %d, error = %v", auditCount, err)
	}
}

func TestPlatformRoleChangesAreAuthorizedAuditedAndPreserveLastAdmin(t *testing.T) {
	db := newPlatformAdminTestDB(t)
	admin := models.User{FullName: "Admin", Email: "admin@example.invalid", PlatformRole: models.PlatformRoleSuperAdmin}
	user := models.User{FullName: "User", Email: "user@example.invalid", PlatformRole: models.PlatformRoleUser}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	if err := UpdatePlatformRole(db, user.ID, admin.ID, models.PlatformRoleUser); !errors.Is(err, ErrPlatformAdminRequired) {
		t.Fatalf("non-admin role change error = %v, want ErrPlatformAdminRequired", err)
	}
	if err := UpdatePlatformRole(db, admin.ID, user.ID, models.PlatformRoleSuperAdmin); err != nil {
		t.Fatalf("promote user: %v", err)
	}
	if err := UpdatePlatformRole(db, admin.ID, admin.ID, models.PlatformRoleUser); err != nil {
		t.Fatalf("demote one admin while another remains: %v", err)
	}
	if err := UpdatePlatformRole(db, user.ID, user.ID, models.PlatformRoleUser); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("last admin demotion error = %v, want ErrLastSuperAdmin", err)
	}
	if err := UpdatePlatformRole(db, admin.ID, user.ID, "tenant_admin"); !errors.Is(err, ErrPlatformRoleInvalid) {
		t.Fatalf("unknown platform role error = %v, want ErrPlatformRoleInvalid", err)
	}
	var auditCount int64
	if err := db.Model(&models.PlatformAdminAuditEvent{}).Count(&auditCount).Error; err != nil || auditCount != 2 {
		t.Fatalf("role-change audit count = %d, error = %v, want 2", auditCount, err)
	}
}
