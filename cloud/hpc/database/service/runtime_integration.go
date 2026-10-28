package service

import (
	"fmt"

	"atonixcorp/cloud/hpc/database/model"
)

type RuntimeIntegration struct {
	DB *DatabaseService
}

func NewRuntimeIntegration(db *DatabaseService) *RuntimeIntegration {
	return &RuntimeIntegration{DB: db}
}

func (ri *RuntimeIntegration) InjectForSlurm(datasetID, jobID string) string {
	mount := ri.DB.Mounts.MountForSlurm(datasetID, jobID)
	fmt.Printf("Runtime injected for Slurm: dataset=%s job=%s mount=%s\n",
		datasetID, jobID, mount)
	return mount
}

func (ri *RuntimeIntegration) InjectForK8s(datasetID, pod string) string {
	mount := ri.DB.Mounts.MountForK8s(datasetID, pod)
	fmt.Printf("Runtime injected for K8s: dataset=%s pod=%s mount=%s\n",
		datasetID, pod, mount)
	return mount
}

func (ri *RuntimeIntegration) DatabaseForSlurm(databaseID string) (model.ManagedDatabase, error) {
	database, ok := ri.DB.ManagedDatabase(databaseID)
	if !ok {
		return model.ManagedDatabase{}, fmt.Errorf("managed database %q is not registered", databaseID)
	}
	return database, nil
}
