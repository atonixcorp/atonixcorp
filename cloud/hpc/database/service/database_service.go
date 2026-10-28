package service

import (
	"fmt"
	"sync"

	"atonixcorp/cloud/hpc/database/io"
	"atonixcorp/cloud/hpc/database/model"
	"atonixcorp/cloud/hpc/database/monitoring"
	"atonixcorp/cloud/hpc/database/registry"
)

type DatabaseService struct {
	Registry  *registry.DatasetRegistry
	Versions  *registry.VersionManager
	Access    *registry.AccessControl
	IO        *io.ParallelIO
	MPIIO     *io.MPIIO
	Cache     *io.CacheEngine
	Mounts    *io.Mounts
	Metrics   map[string]*monitoring.DatasetMetrics
	IOMetrics map[string]*monitoring.IOMetrics
	Health    map[string]*monitoring.HealthStatus
	managedDatabaseMu sync.RWMutex
	ManagedDatabases map[string]model.ManagedDatabase
}

func NewDatabaseService() *DatabaseService {
	return &DatabaseService{
		Registry:  registry.NewDatasetRegistry(),
		Versions:  registry.NewVersionManager(),
		Access:    registry.NewAccessControl(),
		IO:        io.NewParallelIO(),
		MPIIO:     io.NewMPIIO(),
		Cache:     io.NewCacheEngine(),
		Mounts:    io.NewMounts(),
		Metrics:   make(map[string]*monitoring.DatasetMetrics),
		IOMetrics: make(map[string]*monitoring.IOMetrics),
		Health:    make(map[string]*monitoring.HealthStatus),
		ManagedDatabases: make(map[string]model.ManagedDatabase),
	}
}

func (db *DatabaseService) RegisterManagedDatabase(database model.ManagedDatabase) error {
	if database.ID == "" || database.Engine == "" || database.Endpoint == "" ||
		database.Port == 0 || database.DatabaseName == "" || database.CredentialRef == "" ||
		database.NetworkID == "" || !database.TLSRequired {
		return fmt.Errorf("database connection requires identity, private network, credential reference, and TLS")
	}
	db.managedDatabaseMu.Lock()
	defer db.managedDatabaseMu.Unlock()
	db.ManagedDatabases[database.ID] = database
	return nil
}

func (db *DatabaseService) ManagedDatabase(id string) (model.ManagedDatabase, bool) {
	db.managedDatabaseMu.RLock()
	defer db.managedDatabaseMu.RUnlock()
	database, ok := db.ManagedDatabases[id]
	return database, ok
}

func (db *DatabaseService) RegisterDataset(meta *model.Dataset) {
	db.Registry.Register(&registry.DatasetMetadata{
		ID:          meta.ID,
		Name:        meta.Name,
		Format:      meta.Format,
		Owner:       meta.Owner,
		Tags:        meta.Tags,
		Description: meta.Description,
		CreatedAt:   meta.CreatedAt,
		UpdatedAt:   meta.UpdatedAt,
	})

	db.Metrics[meta.ID] = monitoring.NewDatasetMetrics(meta.ID)
	db.IOMetrics[meta.ID] = monitoring.NewIOMetrics(meta.ID)
	db.Health[meta.ID] = monitoring.NewHealthStatus(meta.ID)

	fmt.Printf("Dataset registered in DatabaseService: %s\n", meta.ID)
}
