package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrForbidden = errors.New("storage project access denied")
	ErrNotFound  = errors.New("storage resource not found")
	ErrInvalid   = errors.New("invalid storage request")
	ErrConflict  = errors.New("storage request conflicts with resource state")
	ErrQuota     = errors.New("storage quota exceeded")
)

var volumeNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])$`)

type Service struct {
	db     *gorm.DB
	driver BlockDriver
}

type CreateVolumeRequest struct {
	TenantID         uint              `json:"tenant_id"`
	ProjectID        uint              `json:"project_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Region           string            `json:"region"`
	AvailabilityZone string            `json:"availability_zone"`
	StorageClass     string            `json:"storage_class"`
	SizeGB           int64             `json:"size_gb"`
	Encrypted        bool              `json:"encrypted"`
	DeleteProtection *bool             `json:"delete_protection"`
	Tags             map[string]string `json:"tags"`
}

type VolumeResponse struct {
	Volume    models.StorageVolume    `json:"volume"`
	Operation models.StorageOperation `json:"operation"`
}

type Overview struct {
	Driver      string `json:"driver"`
	Volumes     int64  `json:"volumes"`
	UsedGB      int64  `json:"used_gb"`
	QuotaGB     int64  `json:"quota_gb"`
	VolumeQuota int    `json:"volume_quota"`
}

func NewService(db *gorm.DB, driver BlockDriver) *Service {
	return &Service{db: db, driver: driver}
}

func (s *Service) VolumeTypes(ctx context.Context) ([]VolumeType, error) {
	if s.driver == nil {
		return []VolumeType{}, nil
	}
	return s.driver.VolumeTypes(ctx)
}

func (s *Service) CreateVolume(
	ctx context.Context,
	userID uint,
	idempotencyKey string,
	request CreateVolumeRequest,
) (VolumeResponse, error) {
	if s.driver == nil {
		return VolumeResponse{}, ErrDriverUnavailable
	}
	if userID == 0 || request.TenantID == 0 || request.ProjectID == 0 {
		return VolumeResponse{}, ErrForbidden
	}
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return VolumeResponse{}, fmt.Errorf("%w: a valid Idempotency-Key header is required", ErrInvalid)
	}
	if err := s.authorizeProject(ctx, userID, request.TenantID, request.ProjectID, true); err != nil {
		return VolumeResponse{}, err
	}
	if err := normalizeVolumeRequest(&request); err != nil {
		return VolumeResponse{}, err
	}
	if err := s.driver.ValidateVolume(
		ctx, request.Region, request.AvailabilityZone, request.StorageClass, request.Encrypted,
	); err != nil {
		return VolumeResponse{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return VolumeResponse{}, err
	}
	requestHash := sha256.Sum256(requestJSON)
	hash := hex.EncodeToString(requestHash[:])

	var result VolumeResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.StorageVolume
		lookupErr := tx.Where(
			"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
			request.TenantID, request.ProjectID, idempotencyKey,
		).First(&existing).Error
		if lookupErr == nil {
			if existing.RequestHash != hash {
				return ErrConflict
			}
			var operation models.StorageOperation
			if err := tx.Where("resource_id = ? AND action = ?", existing.ID, "create").
				Order("created_at DESC").First(&operation).Error; err != nil {
				return err
			}
			if err := decodeVolumeTags(&existing); err != nil {
				return err
			}
			result = VolumeResponse{Volume: existing, Operation: operation}
			return nil
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}
		var duplicateName models.StorageVolume
		if err := tx.Where(
			"tenant_id = ? AND project_id = ? AND name = ? AND state <> ?",
			request.TenantID, request.ProjectID, request.Name, "DELETED",
		).First(&duplicateName).Error; err == nil {
			return fmt.Errorf("%w: a volume with this name already exists in the project", ErrConflict)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		quota := models.StorageQuota{TenantID: request.TenantID, ProjectID: request.ProjectID}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND project_id = ?", request.TenantID, request.ProjectID).
			FirstOrCreate(&quota).Error; err != nil {
			return err
		}
		var count int64
		var usedGB int64
		query := tx.Model(&models.StorageVolume{}).
			Where("tenant_id = ? AND project_id = ? AND state <> ?", request.TenantID, request.ProjectID, "DELETED")
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if err := query.Select("COALESCE(SUM(size_gb), 0)").Scan(&usedGB).Error; err != nil {
			return err
		}
		if count >= int64(quota.MaxVolumes) || usedGB+request.SizeGB > quota.MaxCapacityGB {
			return ErrQuota
		}

		id, err := newID()
		if err != nil {
			return err
		}
		operationID, err := newID()
		if err != nil {
			return err
		}
		tagsJSON, err := json.Marshal(request.Tags)
		if err != nil {
			return err
		}
		deleteProtection := true
		if request.DeleteProtection != nil {
			deleteProtection = *request.DeleteProtection
		}
		volume := models.StorageVolume{
			ID: id, TenantID: request.TenantID, ProjectID: request.ProjectID, OwnerID: userID,
			Name: request.Name, Description: request.Description, Region: request.Region,
			AvailabilityZone: request.AvailabilityZone, StorageClass: request.StorageClass,
			SizeGB: request.SizeGB, Encrypted: request.Encrypted,
			DeleteProtection: deleteProtection, TagsJSON: tagsJSON, Tags: request.Tags,
			State: "CREATING", Driver: s.driver.Name(), IdempotencyKey: idempotencyKey,
			RequestHash: hash,
		}
		operation := models.StorageOperation{
			ID: operationID, TenantID: request.TenantID, ProjectID: request.ProjectID,
			ResourceID: id, ResourceType: "volume", Action: "create",
			IdempotencyKey: idempotencyKey, ActorID: userID, Status: "PENDING",
		}
		if err := tx.Create(&volume).Error; err != nil {
			return err
		}
		if err := tx.Create(&operation).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.StorageAuditEvent{
			TenantID: request.TenantID, ProjectID: request.ProjectID, ResourceID: id,
			ResourceType: "volume", OperationID: operationID, ActorID: userID,
			Source: "api", Action: "volume.create.requested", Result: "pending",
		}).Error; err != nil {
			return err
		}
		result = VolumeResponse{Volume: volume, Operation: operation}
		return nil
	})
	if err != nil {
		var existing models.StorageVolume
		if lookupErr := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
			request.TenantID, request.ProjectID, idempotencyKey,
		).First(&existing).Error; lookupErr == nil {
			if existing.RequestHash != hash {
				return VolumeResponse{}, ErrConflict
			}
			if decodeErr := decodeVolumeTags(&existing); decodeErr != nil {
				return VolumeResponse{}, decodeErr
			}
			var operation models.StorageOperation
			if opErr := s.db.WithContext(ctx).Where(
				"resource_id = ? AND action = ?", existing.ID, "create",
			).Order("created_at DESC").First(&operation).Error; opErr == nil {
				return VolumeResponse{Volume: existing, Operation: operation}, nil
			}
		}
		return VolumeResponse{}, err
	}
	return result, nil
}

func (s *Service) ListVolumes(ctx context.Context, userID, tenantID, projectID uint) ([]models.StorageVolume, error) {
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	var volumes []models.StorageVolume
	err := s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND state <> ?", tenantID, projectID, "DELETED",
	).Order("created_at DESC").Limit(500).Find(&volumes).Error
	if err != nil {
		return nil, err
	}
	for index := range volumes {
		if err := decodeVolumeTags(&volumes[index]); err != nil {
			return nil, err
		}
	}
	return volumes, nil
}

func (s *Service) GetVolume(ctx context.Context, userID uint, id string) (models.StorageVolume, error) {
	var volume models.StorageVolume
	if err := s.db.WithContext(ctx).First(&volume, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return models.StorageVolume{}, ErrNotFound
	} else if err != nil {
		return models.StorageVolume{}, err
	}
	if err := s.authorizeProject(ctx, userID, volume.TenantID, volume.ProjectID, false); err != nil {
		return models.StorageVolume{}, ErrNotFound
	}
	if err := decodeVolumeTags(&volume); err != nil {
		return models.StorageVolume{}, err
	}
	return volume, nil
}

func (s *Service) DeleteVolume(
	ctx context.Context, userID uint, id, idempotencyKey string,
) (models.StorageOperation, error) {
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return models.StorageOperation{}, fmt.Errorf("%w: a valid Idempotency-Key header is required", ErrInvalid)
	}
	volume, err := s.GetVolume(ctx, userID, id)
	if err != nil {
		return models.StorageOperation{}, err
	}
	if err := s.authorizeProject(ctx, userID, volume.TenantID, volume.ProjectID, true); err != nil {
		return models.StorageOperation{}, ErrNotFound
	}
	var prior models.StorageOperation
	if err := s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
		volume.TenantID, volume.ProjectID, idempotencyKey,
	).First(&prior).Error; err == nil {
		if prior.ResourceID != volume.ID || prior.Action != "delete" {
			return models.StorageOperation{}, ErrConflict
		}
		return prior, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.StorageOperation{}, err
	}
	if volume.DeleteProtection {
		return models.StorageOperation{}, fmt.Errorf("%w: disable delete protection before deleting this volume", ErrConflict)
	}
	if volume.State != "AVAILABLE" && volume.State != "ERROR" {
		return models.StorageOperation{}, ErrConflict
	}

	var operation models.StorageOperation
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var prior models.StorageOperation
		if err := tx.Where(
			"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
			volume.TenantID, volume.ProjectID, idempotencyKey,
		).First(&prior).Error; err == nil {
			operation = prior
			return nil
		}
		id, err := newID()
		if err != nil {
			return err
		}
		operation = models.StorageOperation{
			ID: id, TenantID: volume.TenantID, ProjectID: volume.ProjectID,
			ResourceID: volume.ID, ResourceType: "volume", Action: "delete",
			IdempotencyKey: idempotencyKey, ActorID: userID, Status: "PENDING",
		}
		update := tx.Model(&volume).Where("state IN ?", []string{"AVAILABLE", "ERROR"}).
			Update("state", "DELETING")
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrConflict
		}
		if err := tx.Create(&operation).Error; err != nil {
			return err
		}
		return tx.Create(&models.StorageAuditEvent{
			TenantID: volume.TenantID, ProjectID: volume.ProjectID, ResourceID: volume.ID,
			ResourceType: "volume", OperationID: operation.ID, ActorID: userID, Source: "api",
			Action: "volume.delete.requested", Result: "pending",
		}).Error
	})
	return operation, err
}

func (s *Service) GetOperation(ctx context.Context, userID uint, id string) (models.StorageOperation, error) {
	var operation models.StorageOperation
	err := s.db.WithContext(ctx).First(&operation, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.StorageOperation{}, ErrNotFound
	}
	if err != nil {
		return models.StorageOperation{}, err
	}
	if err := s.authorizeProject(ctx, userID, operation.TenantID, operation.ProjectID, false); err != nil {
		return models.StorageOperation{}, ErrNotFound
	}
	return operation, nil
}

func (s *Service) ListOperations(
	ctx context.Context, userID, tenantID, projectID uint,
) ([]models.StorageOperation, error) {
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	var operations []models.StorageOperation
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		Order("created_at DESC").Limit(100).Find(&operations).Error
	return operations, err
}

func (s *Service) GetOverview(ctx context.Context, userID, tenantID, projectID uint) (Overview, error) {
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return Overview{}, err
	}
	var summary struct {
		Volumes int64
		UsedGB  int64
	}
	err := s.db.WithContext(ctx).Model(&models.StorageVolume{}).
		Select("COUNT(*) AS volumes, COALESCE(SUM(size_gb), 0) AS used_gb").
		Where("tenant_id = ? AND project_id = ? AND state <> ?", tenantID, projectID, "DELETED").
		Scan(&summary).Error
	if err != nil {
		return Overview{}, err
	}
	quota := models.StorageQuota{TenantID: tenantID, ProjectID: projectID}
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		FirstOrCreate(&quota).Error; err != nil {
		return Overview{}, err
	}
	driverName := "unavailable"
	if s.driver != nil {
		driverName = s.driver.Name()
	}
	return Overview{
		Driver: driverName, Volumes: summary.Volumes, UsedGB: summary.UsedGB,
		QuotaGB: quota.MaxCapacityGB, VolumeQuota: quota.MaxVolumes,
	}, nil
}

func (s *Service) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := s.recoverStaleOperations(ctx); err != nil && ctx.Err() == nil {
			log.Printf("recover storage operations: %v", err)
		}
		if err := s.processNext(ctx); err != nil && ctx.Err() == nil {
			log.Printf("process storage operation: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) recoverStaleOperations(ctx context.Context) error {
	var operations []models.StorageOperation
	if err := s.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", "RUNNING", time.Now().UTC().Add(-2*time.Minute)).
		Order("updated_at ASC").Limit(20).Find(&operations).Error; err != nil {
		return err
	}
	for _, operation := range operations {
		if operation.Attempts >= 3 {
			var volume models.StorageVolume
			if err := s.db.WithContext(ctx).First(&volume, "id = ?", operation.ResourceID).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				log.Printf("storage operation %s references a missing volume", operation.ID)
			}
			if err := s.failOperation(ctx, operation, volume); err != nil {
				return err
			}
			continue
		}
		if err := s.db.WithContext(ctx).Model(&models.StorageOperation{}).
			Where("id = ? AND status = ?", operation.ID, "RUNNING").
			Update("status", "PENDING").Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) processNext(ctx context.Context) error {
	if s.driver == nil {
		return ErrDriverUnavailable
	}
	var operation models.StorageOperation
	result := s.db.WithContext(ctx).Where("status = ?", "PENDING").
		Order("created_at ASC").Limit(1).Find(&operation)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	claim := s.db.WithContext(ctx).Model(&models.StorageOperation{}).
		Where("id = ? AND status = ?", operation.ID, "PENDING").
		Update("status", "RUNNING")
	if claim.Error != nil || claim.RowsAffected == 0 {
		return claim.Error
	}
	var volume models.StorageVolume
	if err := s.db.WithContext(ctx).First(&volume, "id = ?", operation.ResourceID).Error; err != nil {
		return s.failOperation(ctx, operation, models.StorageVolume{})
	}

	var driverErr error
	complete := false
	switch operation.Action {
	case "create":
		var providerVolume ProviderVolume
		if volume.ProviderID == "" {
			providerVolume, driverErr = s.driver.EnsureVolume(ctx, volume)
			if driverErr == nil {
				volume.ProviderID = providerVolume.ID
				if err := s.db.WithContext(ctx).Model(&volume).Update("provider_id", providerVolume.ID).Error; err != nil {
					return err
				}
			}
		}
		if driverErr == nil {
			providerVolume, driverErr = s.driver.InspectVolume(ctx, volume.ProviderID)
		}
		if driverErr == nil {
			switch providerVolume.Status {
			case "available":
				complete = true
			case "creating", "downloading", "retyping":
				return s.releaseOperation(ctx, operation)
			default:
				driverErr = fmt.Errorf("provider volume entered unexpected state %q", providerVolume.Status)
			}
		}
	case "delete":
		complete, driverErr = s.driver.DeleteVolume(ctx, volume)
	default:
		driverErr = errors.New("unsupported storage operation")
	}
	if driverErr != nil {
		log.Printf("storage driver operation %s failed: %v", operation.ID, driverErr)
		return s.failOperation(ctx, operation, volume)
	}
	if !complete {
		return s.releaseOperation(ctx, operation)
	}
	return s.completeOperation(ctx, operation, volume)
}

func (s *Service) releaseOperation(ctx context.Context, operation models.StorageOperation) error {
	return s.db.WithContext(ctx).Model(&models.StorageOperation{}).
		Where("id = ? AND status = ?", operation.ID, "RUNNING").
		Update("status", "PENDING").Error
}

func (s *Service) failOperation(
	ctx context.Context, operation models.StorageOperation, volume models.StorageVolume,
) error {
	var current models.StorageOperation
	if err := s.db.WithContext(ctx).First(&current, "id = ?", operation.ID).Error; err != nil {
		return err
	}
	current.Attempts++
	retry := current.Attempts < 3
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status := "FAILED"
		updates := map[string]any{
			"status": status, "error_code": "DRIVER_FAILURE", "attempts": current.Attempts,
		}
		if retry {
			updates = map[string]any{"status": "PENDING", "error_code": ""}
		} else {
			now := time.Now().UTC()
			updates["completed_at"] = now
		}
		if err := tx.Model(&current).Updates(updates).Error; err != nil {
			return err
		}
		if !retry && volume.ID != "" {
			state := "ERROR"
			if operation.Action == "delete" {
				state = "AVAILABLE"
			}
			if err := tx.Model(&volume).Updates(map[string]any{
				"state": state, "error_code": "DRIVER_FAILURE",
			}).Error; err != nil {
				return err
			}
			return tx.Create(&models.StorageAuditEvent{
				TenantID: operation.TenantID, ProjectID: operation.ProjectID,
				ResourceID: operation.ResourceID, ResourceType: operation.ResourceType,
				OperationID: operation.ID, ActorID: operation.ActorID, Source: "worker",
				Action: "volume." + operation.Action + ".failed", Result: "failed",
			}).Error
		}
		return nil
	})
}

func (s *Service) completeOperation(
	ctx context.Context, operation models.StorageOperation, volume models.StorageVolume,
) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&operation).Updates(map[string]any{
			"status": "SUCCEEDED", "error_code": "", "completed_at": now,
		}).Error; err != nil {
			return err
		}
		volumeUpdates := map[string]any{"error_code": ""}
		if operation.Action == "create" {
			volumeUpdates["state"] = "AVAILABLE"
		} else {
			volumeUpdates["state"] = "DELETED"
			volumeUpdates["deleted_at"] = now
		}
		if err := tx.Model(&volume).Updates(volumeUpdates).Error; err != nil {
			return err
		}
		return tx.Create(&models.StorageAuditEvent{
			TenantID: operation.TenantID, ProjectID: operation.ProjectID,
			ResourceID: operation.ResourceID, ResourceType: operation.ResourceType,
			OperationID: operation.ID, ActorID: operation.ActorID, Source: "worker",
			Action: "volume." + operation.Action + ".completed", Result: "succeeded",
		}).Error
	})
}

func (s *Service) authorizeProject(
	ctx context.Context, userID, tenantID, projectID uint, write bool,
) error {
	var tenantMembership models.TenantMembership
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&tenantMembership).Error; err != nil {
		return ErrForbidden
	}
	var project models.Project
	if err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", projectID, tenantID).
		First(&project).Error; err != nil {
		return ErrForbidden
	}
	var membership models.ProjectMembership
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&membership).Error; err != nil {
		return ErrForbidden
	}
	if write && membership.Role != "owner" && membership.Role != "admin" &&
		membership.Role != "deployer" {
		return ErrForbidden
	}
	return nil
}

func normalizeVolumeRequest(request *CreateVolumeRequest) error {
	request.Name = strings.ToLower(strings.TrimSpace(request.Name))
	request.Description = strings.TrimSpace(request.Description)
	request.Region = strings.TrimSpace(request.Region)
	request.AvailabilityZone = strings.TrimSpace(request.AvailabilityZone)
	request.StorageClass = strings.TrimSpace(request.StorageClass)
	if !volumeNamePattern.MatchString(request.Name) {
		return fmt.Errorf("%w: name must be 2-63 lowercase letters, numbers, or hyphens", ErrInvalid)
	}
	if len(request.Description) > 512 {
		return fmt.Errorf("%w: description is too long", ErrInvalid)
	}
	if request.SizeGB < 1 || request.SizeGB > 1024 {
		return fmt.Errorf("%w: capacity must be between 1 and 1024 GB", ErrInvalid)
	}
	if len(request.Tags) > 20 {
		return fmt.Errorf("%w: a maximum of 20 tags is allowed", ErrInvalid)
	}
	for key, value := range request.Tags {
		if strings.TrimSpace(key) == "" || len(key) > 64 || len(value) > 256 {
			return fmt.Errorf("%w: tag keys and values exceed the allowed limits", ErrInvalid)
		}
	}
	return nil
}

func newID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func decodeVolumeTags(volume *models.StorageVolume) error {
	if len(volume.TagsJSON) == 0 {
		volume.Tags = map[string]string{}
		return nil
	}
	if err := json.Unmarshal(volume.TagsJSON, &volume.Tags); err != nil {
		return fmt.Errorf("decode storage volume tags: %w", err)
	}
	return nil
}
