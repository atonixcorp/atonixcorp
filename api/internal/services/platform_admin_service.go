package services

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"atonixcorp/api/internal/authorization"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSuperAdminAlreadyExists = errors.New("a platform super admin already exists")
	ErrLastSuperAdmin          = errors.New("the last platform super admin cannot be demoted")
	ErrPlatformRoleInvalid     = errors.New("invalid platform role")
	ErrPlatformAdminRequired   = errors.New("platform super admin permission is required")
)

const platformAdminLockID int64 = 748219631

func BootstrapSuperAdmin(db *gorm.DB, fullName, email, password string) error {
	fullName = strings.TrimSpace(fullName)
	email = strings.ToLower(strings.TrimSpace(email))
	address, emailErr := mail.ParseAddress(email)
	if fullName == "" || len(fullName) > 255 || emailErr != nil || address.Address != email {
		return errors.New("a name and valid email address are required")
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("password must be between 12 and 72 bytes")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return fmt.Errorf("hash platform super admin password: %w", err)
	}

	user := &models.User{
		FullName: fullName, Email: email, Password: string(hashed),
		PlatformRole: models.PlatformRoleSuperAdmin,
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockPlatformAdminChanges(tx); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.User{}).
			Where("platform_role = ?", models.PlatformRoleSuperAdmin).
			Count(&count).Error; err != nil {
			return fmt.Errorf("check existing platform super admins: %w", err)
		}
		if count > 0 {
			return ErrSuperAdminAlreadyExists
		}
		if err := repositories.CreateWithWorkspace(tx, user); err != nil {
			return err
		}
		return tx.Create(&models.PlatformAdminAuditEvent{
			ActorID: user.ID, TargetUserID: user.ID, Action: "platform_admin.bootstrapped",
			PreviousRole: models.PlatformRoleUser, NewRole: models.PlatformRoleSuperAdmin,
			CreatedAt: time.Now().UTC(),
		}).Error
	})
}

func UpdatePlatformRole(db *gorm.DB, actorID, targetID uint, role string) error {
	if role != models.PlatformRoleUser && role != models.PlatformRoleSuperAdmin {
		return ErrPlatformRoleInvalid
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockPlatformAdminChanges(tx); err != nil {
			return err
		}
		isAdmin, err := authorization.IsPlatformSuperAdmin(tx.Statement.Context, tx, actorID)
		if err != nil {
			return fmt.Errorf("verify platform administrator: %w", err)
		}
		if !isAdmin {
			return ErrPlatformAdminRequired
		}

		var target models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&target, targetID).Error; err != nil {
			return err
		}
		if target.PlatformRole == role {
			return nil
		}
		if target.PlatformRole == models.PlatformRoleSuperAdmin && role == models.PlatformRoleUser {
			var count int64
			if err := tx.Model(&models.User{}).
				Where("platform_role = ?", models.PlatformRoleSuperAdmin).
				Count(&count).Error; err != nil {
				return fmt.Errorf("check remaining platform super admins: %w", err)
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}
		if err := tx.Model(&target).Update("platform_role", role).Error; err != nil {
			return err
		}
		return tx.Create(&models.PlatformAdminAuditEvent{
			ActorID: actorID, TargetUserID: target.ID, Action: "platform_role.updated",
			PreviousRole: target.PlatformRole, NewRole: role, CreatedAt: time.Now().UTC(),
		}).Error
	})
}

func lockPlatformAdminChanges(tx *gorm.DB) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", platformAdminLockID).Error
}
