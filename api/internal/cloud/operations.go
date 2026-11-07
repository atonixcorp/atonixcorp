package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"atonixcorp/api/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type requestIDContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func (s *Service) createOperation(
	ctx context.Context,
	instance models.CloudInstance,
	actorID uint,
	action string,
) (models.CloudOperation, error) {
	id, err := newInstanceID()
	if err != nil {
		return models.CloudOperation{}, err
	}
	operation := models.CloudOperation{
		ID: id, TenantID: instance.TenantID, ProjectID: instance.ProjectID,
		ResourceID: instance.ID, ResourceType: "instance", Operation: action,
		Provider: instance.Provider, ProviderOperationID: instance.ProviderInstanceID,
		ProviderStatusAtStart: instance.ProviderStatus,
		Status: "RUNNING", ActorID: actorID, RequestID: operationRequestID(ctx),
		StartedAt: time.Now().UTC(),
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedInstance models.CloudInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&lockedInstance, "id = ?", instance.ID).Error; err != nil {
			return err
		}
		var activeCount int64
		if err := tx.Model(&models.CloudOperation{}).
			Where("resource_id = ? AND status IN ?", instance.ID, []string{"QUEUED", "RUNNING"}).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return fmt.Errorf("%w: an operation is already in progress", ErrConflict)
		}
		if err := tx.Create(&operation).Error; err != nil {
			return err
		}
		return tx.Create(&models.CloudAuditEvent{
			TenantID: instance.TenantID, ProjectID: instance.ProjectID, ActorID: actorID,
			ResourceID: instance.ID, ResourceType: "instance", OperationID: operation.ID,
			Action: "instance." + action, Result: "pending", RequestID: operation.RequestID,
			CreatedAt: operation.StartedAt,
		}).Error
	})
	return operation, err
}

func (s *Service) completeOperation(
	ctx context.Context,
	operation *models.CloudOperation,
	status, errorCode, result string,
) error {
	now := time.Now().UTC()
	operation.Status = status
	operation.ErrorCode = errorCode
	operation.CompletedAt = &now
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&models.CloudOperation{}).
			Where("id = ? AND status IN ?", operation.ID, []string{"QUEUED", "RUNNING"}).
			Updates(map[string]any{
				"status": status, "error_code": errorCode, "completed_at": now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return nil
		}
		return tx.Create(&models.CloudAuditEvent{
			TenantID: operation.TenantID, ProjectID: operation.ProjectID,
			ActorID: operation.ActorID, ResourceID: operation.ResourceID,
			ResourceType: operation.ResourceType, OperationID: operation.ID,
			Action: "instance." + operation.Operation, Result: result,
			RequestID: operation.RequestID, CreatedAt: now,
		}).Error
	})
}

func operationRequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	if len(requestID) > 128 {
		return requestID[:128]
	}
	return requestID
}

func (s *Service) ListOperations(
	ctx context.Context, userID, tenantID, projectID uint,
) ([]models.CloudOperation, error) {
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return nil, ErrForbidden
	}
	if err := s.authorizeProject(ctx, userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	var operations []models.CloudOperation
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		Order("created_at DESC").Limit(100).Find(&operations).Error
	return operations, err
}

func (s *Service) GetOperation(
	ctx context.Context, userID uint, id string,
) (models.CloudOperation, error) {
	if userID == 0 {
		return models.CloudOperation{}, ErrForbidden
	}
	var operation models.CloudOperation
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&operation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CloudOperation{}, ErrNotFound
		}
		return models.CloudOperation{}, err
	}
	if err := s.authorizeProject(
		ctx, userID, operation.TenantID, operation.ProjectID, false,
	); err != nil {
		if errors.Is(err, ErrForbidden) {
			return models.CloudOperation{}, ErrNotFound
		}
		return models.CloudOperation{}, err
	}
	return operation, nil
}

func (s *Service) ensureNoRunningOperation(ctx context.Context, resourceID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.CloudOperation{}).
		Where("resource_id = ? AND status IN ?", resourceID, []string{"QUEUED", "RUNNING"}).
		Count(&count).Error; err != nil {
		return fmt.Errorf("check active cloud operation: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: an operation is already in progress", ErrConflict)
	}
	return nil
}

func (s *Service) ReconcilePendingOperations(ctx context.Context) error {
	var reconciliationErrors []error
	lastID := ""
	for {
		var operations []models.CloudOperation
		if err := s.db.WithContext(ctx).
			Where("status IN ? AND id > ?", []string{"QUEUED", "RUNNING"}, lastID).
			Order("id ASC").Limit(200).Find(&operations).Error; err != nil {
			return errors.Join(append(reconciliationErrors, err)...)
		}
		if len(operations) == 0 {
			break
		}
		for index := range operations {
			if err := s.reconcileOperation(ctx, &operations[index]); err != nil {
				reconciliationErrors = append(reconciliationErrors,
					fmt.Errorf("reconcile operation %s: %w", operations[index].ID, err))
			}
		}
		lastID = operations[len(operations)-1].ID
	}
	return errors.Join(reconciliationErrors...)
}

func (s *Service) reconcileOperation(
	ctx context.Context, operation *models.CloudOperation,
) error {
	var instance models.CloudInstance
	if err := s.db.WithContext(ctx).Where("id = ?", operation.ResourceID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.completeOperation(ctx, operation, "FAILED", "RESOURCE_NOT_FOUND", "failed")
		}
		return err
	}
	if instance.ProviderInstanceID == "" && operation.ProviderOperationID != "" {
		instance.ProviderInstanceID = operation.ProviderOperationID
		if err := s.db.WithContext(ctx).Model(&instance).
			Update("provider_instance_id", instance.ProviderInstanceID).Error; err != nil {
			return err
		}
	}
	provider, err := s.providerForTenant(ctx, operation.TenantID)
	if err != nil {
		return err
	}
	if !providerComputeAvailable(provider) {
		return ErrProviderUnavailable
	}
	if operation.Operation == "delete" && instance.ProviderInstanceID == "" {
		markDeleted(&instance)
		if err := s.saveInstance(ctx, &instance); err != nil {
			return err
		}
		return s.completeOperation(ctx, operation, "SUCCEEDED", "", "succeeded")
	}
	if operation.Operation == "delete" && instance.Status != "DELETING" {
		instance.Status = "DELETING"
		if err := s.db.WithContext(ctx).Model(&instance).Update("status", "DELETING").Error; err != nil {
			return err
		}
	}
	if err := s.refreshInstance(ctx, &instance, provider); err != nil {
		return err
	}
	if operation.Operation == "reboot" && !operation.ProviderTransitionObserved &&
		instance.ProviderStatus != "" &&
		instance.ProviderStatus != operation.ProviderStatusAtStart {
		operation.ProviderTransitionObserved = true
		if err := s.db.WithContext(ctx).Model(operation).
			Update("provider_transition_observed", true).Error; err != nil {
			return err
		}
	}
	if instance.Status == "ERROR" {
		return s.completeOperation(ctx, operation, "FAILED", "PROVIDER_OPERATION_FAILED", "failed")
	}
	if operation.Operation == "delete" && instance.Status == "DELETED" {
		return s.completeOperation(ctx, operation, "SUCCEEDED", "", "succeeded")
	}
	targetStatus := map[string]string{
		"create": "RUNNING", "start": "RUNNING", "stop": "STOPPED", "reboot": "RUNNING",
	}[operation.Operation]
	if targetStatus != "" && instance.Status == targetStatus {
		if operation.Operation == "reboot" && !operation.ProviderTransitionObserved {
			return nil
		}
		return s.completeOperation(ctx, operation, "SUCCEEDED", "", "succeeded")
	}
	return nil
}

func (s *Service) RunReconciler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.ReconcilePendingOperations(ctx); err != nil {
			log.Printf("OpenStack operation reconciliation failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
