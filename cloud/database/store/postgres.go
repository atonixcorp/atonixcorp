package store

import (
	"context"
	"errors"
	"time"

	"atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/internal"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, internal.Wrap("parse postgres config", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, internal.Wrap("connect postgres", err)
	}

	store := &PostgresStore{pool: pool}
	if err := store.Migrate(); err != nil {
		pool.Close()
		return nil, internal.Wrap("migrate database metadata schema", err)
	}
	return store, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) CanAccessProject(userID uint, tenantID, projectID string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			JOIN project_memberships pm ON pm.project_id = p.id
			WHERE p.id::text = $1 AND p.tenant_id::text = $2 AND pm.user_id = $3
		)
	`, projectID, tenantID, userID).Scan(&allowed)
	return allowed, internal.Wrap("authorize database project access", err)
}

func (s *PostgresStore) Migrate() error {
	_, err := s.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS db_instances (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			engine TEXT NOT NULL,
			engine_version TEXT NOT NULL DEFAULT '',
			plan TEXT NOT NULL,
			workload_profile TEXT NOT NULL DEFAULT 'general-purpose',
			cpu INTEGER NOT NULL,
			memory_mb INTEGER NOT NULL,
			storage_gb INTEGER NOT NULL,
			region TEXT NOT NULL,
			status TEXT NOT NULL,
			endpoint TEXT,
			port INTEGER NOT NULL DEFAULT 0,
			network_id TEXT NOT NULL DEFAULT '',
			openstack_server_id TEXT,
			openstack_volume_id TEXT,
			openstack_volume_attached BOOLEAN NOT NULL DEFAULT FALSE,
			openstack_network_id TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT '';
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS engine_version TEXT NOT NULL DEFAULT '';
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS workload_profile TEXT NOT NULL DEFAULT 'general-purpose';
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS port INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS network_id TEXT NOT NULL DEFAULT '';
		ALTER TABLE db_instances ADD COLUMN IF NOT EXISTS openstack_volume_attached BOOLEAN NOT NULL DEFAULT FALSE;
		CREATE INDEX IF NOT EXISTS idx_db_instances_tenant_project
			ON db_instances (tenant_id, project_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_db_instances_status_created
			ON db_instances (status, created_at ASC);
		CREATE TABLE IF NOT EXISTS db_operations (
			id TEXT PRIMARY KEY,
			database_id TEXT NOT NULL REFERENCES db_instances(id),
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			status TEXT NOT NULL,
			stage TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			attempt INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_db_operations_scope_created
			ON db_operations (tenant_id, project_id, created_at DESC);
		CREATE TABLE IF NOT EXISTS db_operation_events (
			id BIGSERIAL PRIMARY KEY,
			operation_id TEXT NOT NULL REFERENCES db_operations(id),
			previous_status TEXT NOT NULL,
			status TEXT NOT NULL,
			stage TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_db_operation_events_operation
			ON db_operation_events (operation_id, id);
		CREATE TABLE IF NOT EXISTS db_idempotency_keys (
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			database_id TEXT NOT NULL,
			operation_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (tenant_id, project_id, idempotency_key)
		);
		CREATE TABLE IF NOT EXISTS db_outbox (
			id BIGSERIAL PRIMARY KEY,
			operation_id TEXT NOT NULL REFERENCES db_operations(id),
			event_type TEXT NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL,
			UNIQUE (operation_id, event_type)
		);
		INSERT INTO db_operations (
			id, database_id, tenant_id, project_id, status, stage, reason, attempt, created_at, updated_at
		)
		SELECT 'op-' || db.id, db.id, db.tenant_id, db.project_id,
		       CASE WHEN db.status = 'provisioning' THEN 'queued' ELSE 'blocked' END,
		       CASE WHEN db.status = 'provisioning' THEN 'QUEUED' ELSE 'BOOTSTRAP_ENGINE' END,
		       CASE WHEN db.status = 'provisioning' THEN '' ELSE 'Legacy resource; managed engine bootstrap must be verified.' END,
		       0, db.created_at, db.updated_at
		FROM db_instances db
		WHERE db.status IN ('provisioning', 'starting')
		  AND NOT EXISTS (SELECT 1 FROM db_operations op WHERE op.database_id = db.id);
		INSERT INTO db_operation_events (
			operation_id, previous_status, status, stage, reason, created_at
		)
		SELECT op.id, '', op.status, op.stage, op.reason, op.created_at
		FROM db_operations op
		WHERE op.id LIKE 'op-db-%'
		  AND NOT EXISTS (
			SELECT 1 FROM db_operation_events event WHERE event.operation_id = op.id
		  );
	`)
	return internal.Wrap("create database metadata schema", err)
}

func (s *PostgresStore) CreateInstanceOperation(
	instance *core.DBInstance, operation *core.DatabaseOperation, key, requestHash string,
) (bool, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, internal.Wrap("begin database create transaction", err)
	}
	defer tx.Rollback(ctx)

	if err := insertInstance(ctx, tx, instance); err != nil {
		return false, internal.Wrap("insert database instance", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO db_operations (
			id, database_id, tenant_id, project_id, status, stage, reason, attempt, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, operation.ID, operation.DatabaseID, operation.TenantID, operation.ProjectID,
		operation.Status, operation.Stage, operation.Reason, operation.Attempt,
		operation.CreatedAt, operation.UpdatedAt); err != nil {
		return false, internal.Wrap("insert database operation", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO db_operation_events (
			operation_id, previous_status, status, stage, reason, created_at
		) VALUES ($1, '', $2, $3, '', $4)
	`, operation.ID, operation.Status, operation.Stage, operation.CreatedAt); err != nil {
		return false, internal.Wrap("record initial database operation event", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO db_outbox (operation_id, event_type, payload, created_at)
		VALUES ($1, 'database.create', jsonb_build_object('database_id', $2), $3)
	`, operation.ID, instance.ID, operation.CreatedAt); err != nil {
		return false, internal.Wrap("enqueue database create operation", err)
	}

	if key != "" {
		tag, err := tx.Exec(ctx, `
			INSERT INTO db_idempotency_keys (
				tenant_id, project_id, idempotency_key, request_hash, database_id, operation_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tenant_id, project_id, idempotency_key) DO NOTHING
		`, instance.TenantID, instance.ProjectID, key, requestHash, instance.ID, operation.ID, instance.CreatedAt)
		if err != nil {
			return false, internal.Wrap("save database idempotency key", err)
		}
		if tag.RowsAffected() == 0 {
			_ = tx.Rollback(ctx)
			var storedHash string
			err := s.pool.QueryRow(ctx, `
				SELECT request_hash, database_id, operation_id
				FROM db_idempotency_keys
				WHERE tenant_id = $1 AND project_id = $2 AND idempotency_key = $3
			`, instance.TenantID, instance.ProjectID, key).Scan(
				&storedHash, &instance.ID, &operation.ID,
			)
			if err != nil {
				return false, internal.Wrap("load existing database idempotency key", err)
			}
			if storedHash != requestHash {
				return false, core.ErrIdempotencyConflict
			}
			operation.DatabaseID = instance.ID
			return true, nil
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, internal.Wrap("commit database create transaction", err)
	}
	return false, nil
}

func insertInstance(ctx context.Context, tx pgx.Tx, inst *core.DBInstance) error {
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = time.Now().UTC()
	}
	inst.UpdatedAt = inst.CreatedAt
	_, err := tx.Exec(ctx, `
		INSERT INTO db_instances (
			id, tenant_id, project_id, name, engine, engine_version, plan, workload_profile,
			cpu, memory_mb, storage_gb, region, status, endpoint, port, network_id,
			openstack_server_id, openstack_volume_id, openstack_volume_attached,
			openstack_network_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
	`, inst.ID, inst.TenantID, inst.ProjectID, inst.Name, inst.Engine,
		inst.EngineVersion, inst.Plan, inst.WorkloadProfile, inst.CPU, inst.MemoryMB,
		inst.StorageGB, inst.Region, inst.Status, inst.Endpoint, inst.Port, inst.NetworkID,
		inst.OpenStackServerID, inst.OpenStackVolumeID, inst.OpenStackVolumeAttached,
		inst.OpenStackNetworkID, inst.CreatedAt, inst.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateInstance(inst *core.DBInstance) error {
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = time.Now().UTC()
	}
	inst.UpdatedAt = inst.CreatedAt

	_, err := s.pool.Exec(context.Background(), `
        INSERT INTO db_instances (
            id, tenant_id, project_id, name, engine, engine_version, plan, workload_profile,
            cpu, memory_mb, storage_gb, region,
            status, endpoint, port, network_id,
            openstack_server_id, openstack_volume_id, openstack_volume_attached, openstack_network_id,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8,
            $9, $10, $11, $12,
            $13, $14, $15, $16,
            $17, $18, $19, $20,
            $21, $22
        )
    `,
		inst.ID,
		inst.TenantID,
		inst.ProjectID,
		inst.Name,
		inst.Engine,
		inst.EngineVersion,
		inst.Plan,
		inst.WorkloadProfile,
		inst.CPU,
		inst.MemoryMB,
		inst.StorageGB,
		inst.Region,
		inst.Status,
		inst.Endpoint,
		inst.Port,
		inst.NetworkID,
		inst.OpenStackServerID,
		inst.OpenStackVolumeID,
		inst.OpenStackVolumeAttached,
		inst.OpenStackNetworkID,
		inst.CreatedAt,
		inst.UpdatedAt,
	)

	return internal.Wrap("insert db_instance", err)
}

func (s *PostgresStore) GetOperation(id string) (*core.DatabaseOperation, error) {
	row := s.pool.QueryRow(context.Background(), `
		SELECT id, database_id, tenant_id, project_id, status, stage, reason, attempt, created_at, updated_at
		FROM db_operations WHERE id = $1
	`, id)
	var operation core.DatabaseOperation
	if err := row.Scan(
		&operation.ID, &operation.DatabaseID, &operation.TenantID, &operation.ProjectID,
		&operation.Status, &operation.Stage, &operation.Reason, &operation.Attempt,
		&operation.CreatedAt, &operation.UpdatedAt,
	); err != nil {
		return nil, internal.Wrap("select database operation", err)
	}
	return &operation, nil
}

func (s *PostgresStore) GetOperationForDatabase(databaseID string) (*core.DatabaseOperation, error) {
	row := s.pool.QueryRow(context.Background(), `
		SELECT id, database_id, tenant_id, project_id, status, stage, reason, attempt, created_at, updated_at
		FROM db_operations WHERE database_id = $1 ORDER BY created_at DESC LIMIT 1
	`, databaseID)
	var operation core.DatabaseOperation
	if err := row.Scan(
		&operation.ID, &operation.DatabaseID, &operation.TenantID, &operation.ProjectID,
		&operation.Status, &operation.Stage, &operation.Reason, &operation.Attempt,
		&operation.CreatedAt, &operation.UpdatedAt,
	); err != nil {
		return nil, internal.Wrap("select database operation by database", err)
	}
	return &operation, nil
}

func (s *PostgresStore) UpdateOperation(operation *core.DatabaseOperation) error {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internal.Wrap("begin database operation update", err)
	}
	defer tx.Rollback(ctx)
	var previousStatus, previousStage, previousReason string
	err = tx.QueryRow(ctx, `
		SELECT status, stage, reason FROM db_operations WHERE id = $1 FOR UPDATE
	`, operation.ID).Scan(&previousStatus, &previousStage, &previousReason)
	if err != nil {
		return internal.Wrap("lock database operation", err)
	}
	if previousStatus != operation.Status && !validOperationTransition(previousStatus, operation.Status) {
		return errors.New("invalid database operation status transition")
	}
	operation.UpdatedAt = time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE db_operations
		SET status = $2, stage = $3, reason = $4, attempt = $5, updated_at = $6
		WHERE id = $1
	`, operation.ID, operation.Status, operation.Stage, operation.Reason, operation.Attempt, operation.UpdatedAt)
	if err != nil {
		return internal.Wrap("update database operation", err)
	}
	if previousStatus != operation.Status || previousStage != operation.Stage || previousReason != operation.Reason {
		if _, err := tx.Exec(ctx, `
			INSERT INTO db_operation_events (operation_id, previous_status, status, stage, reason, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, operation.ID, previousStatus, operation.Status, operation.Stage, operation.Reason, operation.UpdatedAt); err != nil {
			return internal.Wrap("append database operation event", err)
		}
	}
	return internal.Wrap("commit database operation update", tx.Commit(ctx))
}

func validOperationTransition(from, to string) bool {
	switch from {
	case "queued":
		return to == "running" || to == "failed" || to == "cancelled"
	case "running":
		return to == "blocked" || to == "completed" || to == "failed" || to == "cancelled"
	case "blocked":
		return to == "running" || to == "failed" || to == "cancelled"
	default:
		return false
	}
}

func (s *PostgresStore) UpdateInstance(inst *core.DBInstance) error {
	inst.UpdatedAt = time.Now()

	_, err := s.pool.Exec(context.Background(), `
        UPDATE db_instances SET
            tenant_id = $2,
            project_id = $3,
            name = $4,
            engine = $5,
            engine_version = $6,
            plan = $7,
            workload_profile = $8,
            cpu = $9,
            memory_mb = $10,
            storage_gb = $11,
            region = $12,
            status = $13,
            endpoint = $14,
            port = $15,
            network_id = $16,
            openstack_server_id = $17,
            openstack_volume_id = $18,
            openstack_volume_attached = $19,
            openstack_network_id = $20,
            updated_at = $21
        WHERE id = $1
    `,
		inst.ID,
		inst.TenantID,
		inst.ProjectID,
		inst.Name,
		inst.Engine,
		inst.EngineVersion,
		inst.Plan,
		inst.WorkloadProfile,
		inst.CPU,
		inst.MemoryMB,
		inst.StorageGB,
		inst.Region,
		inst.Status,
		inst.Endpoint,
		inst.Port,
		inst.NetworkID,
		inst.OpenStackServerID,
		inst.OpenStackVolumeID,
		inst.OpenStackVolumeAttached,
		inst.OpenStackNetworkID,
		inst.UpdatedAt,
	)

	return internal.Wrap("update db_instance", err)
}

func (s *PostgresStore) ListInstances(tenantID, projectID string) ([]core.DBInstance, error) {
	rows, err := s.pool.Query(context.Background(), `
		SELECT id, tenant_id, project_id, name, engine, engine_version, plan, workload_profile,
		       cpu, memory_mb, storage_gb, region, status, COALESCE(endpoint, ''),
		       port, network_id, COALESCE(openstack_server_id, ''),
		       COALESCE(openstack_volume_id, ''), openstack_volume_attached,
		       COALESCE(openstack_network_id, ''),
		       created_at, updated_at
		FROM db_instances
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY created_at DESC LIMIT 500
	`, tenantID, projectID)
	if err != nil {
		return nil, internal.Wrap("list db_instances", err)
	}
	defer rows.Close()
	var instances []core.DBInstance
	for rows.Next() {
		var instance core.DBInstance
		if err := scanInstance(rows, &instance); err != nil {
			return nil, internal.Wrap("scan db_instance", err)
		}
		instances = append(instances, instance)
	}
	return instances, internal.Wrap("iterate db_instances", rows.Err())
}

func (s *PostgresStore) ListInstancesByStatus(status string, limit int) ([]core.DBInstance, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(context.Background(), `
		SELECT id, tenant_id, project_id, name, engine, engine_version, plan, workload_profile,
		       cpu, memory_mb, storage_gb, region, status, COALESCE(endpoint, ''),
		       port, network_id, COALESCE(openstack_server_id, ''),
		       COALESCE(openstack_volume_id, ''), openstack_volume_attached,
		       COALESCE(openstack_network_id, ''),
		       created_at, updated_at
		FROM db_instances
		WHERE status = $1 ORDER BY created_at ASC LIMIT $2
	`, status, limit)
	if err != nil {
		return nil, internal.Wrap("list db_instances by status", err)
	}
	defer rows.Close()
	var instances []core.DBInstance
	for rows.Next() {
		var instance core.DBInstance
		if err := scanInstance(rows, &instance); err != nil {
			return nil, internal.Wrap("scan db_instance", err)
		}
		instances = append(instances, instance)
	}
	return instances, internal.Wrap("iterate db_instances", rows.Err())
}

func (s *PostgresStore) GetInstance(id string) (*core.DBInstance, error) {
	row := s.pool.QueryRow(context.Background(), `
        SELECT
            id, tenant_id, project_id, name, engine, engine_version, plan, workload_profile,
            cpu, memory_mb, storage_gb, region,
            status, COALESCE(endpoint, ''), port, network_id,
            openstack_server_id, openstack_volume_id, openstack_volume_attached, openstack_network_id,
            created_at, updated_at
        FROM db_instances
        WHERE id = $1
    `, id)

	var inst core.DBInstance
	err := row.Scan(
		&inst.ID,
		&inst.TenantID,
		&inst.ProjectID,
		&inst.Name,
		&inst.Engine,
		&inst.EngineVersion,
		&inst.Plan,
		&inst.WorkloadProfile,
		&inst.CPU,
		&inst.MemoryMB,
		&inst.StorageGB,
		&inst.Region,
		&inst.Status,
		&inst.Endpoint,
		&inst.Port,
		&inst.NetworkID,
		&inst.OpenStackServerID,
		&inst.OpenStackVolumeID,
		&inst.OpenStackVolumeAttached,
		&inst.OpenStackNetworkID,
		&inst.CreatedAt,
		&inst.UpdatedAt,
	)
	if err != nil {
		return nil, internal.Wrap("select db_instance", err)
	}

	return &inst, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInstance(row rowScanner, instance *core.DBInstance) error {
	return row.Scan(
		&instance.ID, &instance.TenantID, &instance.ProjectID, &instance.Name,
		&instance.Engine, &instance.EngineVersion, &instance.Plan, &instance.WorkloadProfile,
		&instance.CPU, &instance.MemoryMB, &instance.StorageGB, &instance.Region,
		&instance.Status, &instance.Endpoint, &instance.Port, &instance.NetworkID,
		&instance.OpenStackServerID, &instance.OpenStackVolumeID, &instance.OpenStackVolumeAttached,
		&instance.OpenStackNetworkID,
		&instance.CreatedAt, &instance.UpdatedAt,
	)
}
