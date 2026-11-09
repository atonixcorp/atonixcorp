package files

import (
	"context"
	"log"
	"time"

	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
)

func (s *Service) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		s.cleanupExpired(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) cleanupExpired(ctx context.Context) {
	if s.provider == nil {
		return
	}
	var objects []models.StorageObject
	err := s.db.WithContext(ctx).Where(
		"staging_key <> ? AND upload_expires_at < ? AND status IN ? AND provider = ? AND bucket = ?",
		"", time.Now().UTC(), []string{"pending", "expired", "active", "deleting", "deleted"}, s.provider.Name(), s.provider.Bucket(),
	).Limit(100).Find(&objects).Error
	if err != nil {
		log.Printf("find expired object uploads: %v", err)
		return
	}
	for _, obj := range objects {
		if obj.Status == "pending" {
			err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				result := tx.Model(&models.StorageObject{}).Where("id = ? AND status = ?", obj.ID, "pending").Update("status", "expired")
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return gorm.ErrRecordNotFound
				}
				return tx.Create(&models.ObjectAuditEvent{ObjectID: obj.ID, ActorID: obj.OwnerID, TenantID: obj.TenantID, Action: "upload_expire"}).Error
			})
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					log.Printf("expire upload %s: %v", obj.ID, err)
				}
				continue
			}
		}
		if err := s.provider.Delete(ctx, obj.StagingKey); err != nil {
			log.Printf("clean staging object %s: %v", obj.ID, err)
			continue
		}
		if err := s.db.WithContext(ctx).Model(&models.StorageObject{}).Where("id = ? AND staging_key = ?", obj.ID, obj.StagingKey).Update("staging_key", "").Error; err != nil {
			log.Printf("clear staging metadata %s: %v", obj.ID, err)
		}
	}
}
