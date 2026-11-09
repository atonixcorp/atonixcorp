package authorization

import (
	"context"

	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

func IsPlatformSuperAdmin(ctx context.Context, db *gorm.DB, userID uint) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var count int64
	err := db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND platform_role = ?", userID, models.PlatformRoleSuperAdmin).
		Count(&count).Error
	return count == 1, err
}
