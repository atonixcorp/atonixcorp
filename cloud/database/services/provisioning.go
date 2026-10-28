package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/internal"
)

type ProvisioningService struct {
	controller *core.Controller
}

func NewProvisioningService(controller *core.Controller) *ProvisioningService {
	return &ProvisioningService{controller}
}

func (p *ProvisioningService) GetInstance(instanceID string) (*core.DBInstance, error) {
	return p.controller.GetInstance(instanceID)
}

// ProvisionDBInstance is the high-level workflow used by API/UI.
func (p *ProvisioningService) ProvisionDBInstance(req core.CreateInstanceRequest) (*core.DBInstance, error) {
	internal.Info("Provisioning DB instance for tenant %s", req.TenantID)

	inst, err := p.controller.CreateInstance(req)
	if err != nil {
		internal.Error("Provisioning failed: %v", err)
		return nil, fmt.Errorf("failed to provision instance: %w", err)
	}

	internal.Info("Database provisioning request accepted: %s", inst.ID)
	return inst, nil
}

func (p *ProvisioningService) CreateDatabase(
	req core.CreateInstanceRequest, idempotencyKey string,
) (*core.CreateDatabaseResult, error) {
	result, err := p.controller.CreateInstanceOperation(req, idempotencyKey)
	if err != nil {
		internal.Error("Database request rejected: %v", err)
		return nil, fmt.Errorf("database request failed: %w", err)
	}
	return result, nil
}

func (p *ProvisioningService) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := p.controller.ProcessPendingInstances(20); err != nil && ctx.Err() == nil {
			log.Printf("process pending database provisions: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
