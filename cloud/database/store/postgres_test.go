package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"atonixcorp/cloud/database/core"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresProviderMetadataMigrationAndRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_TEST_DSN")
	if dsn == "" {
		t.Skip("set DATABASE_TEST_DSN to an isolated PostgreSQL database")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("provider_boundary_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &PostgresStore{pool: pool}
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO db_instances (
		id, tenant_id, project_id, name, engine, plan, cpu, memory_mb, storage_gb,
		region, status, openstack_server_id, openstack_volume_id, openstack_network_id,
		created_at, updated_at
	) VALUES ('legacy', '1', '2', 'legacy', 'postgres', 'basic', 1, 2048, 20,
		'RegionOne', 'provisioning', 'legacy-server', 'legacy-volume', 'legacy-network', NOW(), NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	legacy, err := store.GetInstance("legacy")
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(legacy.ProviderMetadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if legacy.Provider != "openstack" || metadata["server_id"] != "legacy-server" || metadata["volume_id"] != "legacy-volume" || metadata["network_id"] != "legacy-network" {
		t.Fatalf("legacy provider state lost: %+v", legacy)
	}
	instance := &core.DBInstance{
		ID: "generic", TenantID: "1", ProjectID: "2", Name: "generic", Engine: "postgres", Plan: "basic",
		CPU: 1, MemoryMB: 2048, StorageGB: 20, Region: "test-region", Status: "provisioning", Provider: "test-only",
		ProviderMetadata: json.RawMessage(`{"remote_database":"test-resource"}`),
	}
	operation := &core.DatabaseOperation{ID: "operation", DatabaseID: instance.ID, TenantID: "1", ProjectID: "2", Status: "queued", Stage: "QUEUED", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if _, err := store.CreateInstanceOperation(instance, operation, "test-key", "test-hash"); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetInstance(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Provider != "test-only" {
		t.Fatal("provider identity did not persist")
	}
	loaded.Status = "starting"
	loaded.ProviderMetadata = json.RawMessage(`{"remote_database":"test-resource","ready":true}`)
	if err := store.UpdateInstance(loaded); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.GetInstance(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(loaded.ProviderMetadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if loaded.Provider != "test-only" || metadata["ready"] != true {
		t.Fatal("migration overwrote non-OpenStack metadata")
	}
	instances, err := store.ListInstances("1", "2")
	if err != nil || len(instances) != 2 {
		t.Fatalf("list failed: %v", err)
	}
	pending, err := store.ListInstancesByStatus("provisioning", 20)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending list failed: %v", err)
	}
	replay := *instance
	replay.ID = "duplicate"
	replayedOperation := *operation
	replayedOperation.ID = "duplicate-operation"
	replayedOperation.DatabaseID = replay.ID
	replayed, err := store.CreateInstanceOperation(&replay, &replayedOperation, "test-key", "test-hash")
	if err != nil || !replayed || replay.ID != instance.ID {
		t.Fatalf("idempotency lost: %v", err)
	}
}
