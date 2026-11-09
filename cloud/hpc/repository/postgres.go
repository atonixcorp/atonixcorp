package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"atonixcorp/cloud/hpc/datasets"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/nodes"
	_ "github.com/lib/pq"
)

var ErrRecordNotFound = errors.New("HPC record not found")

type Postgres struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Postgres{db: db}, nil
}

func (p *Postgres) Close() error {
	return p.db.Close()
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *Postgres) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS hpc_jobs (
			id text PRIMARY KEY,
			tenant_id text NOT NULL,
			user_id text NOT NULL,
			name text NOT NULL,
			backend text NOT NULL,
			scheduler_id text NOT NULL DEFAULT '',
			execution_profile text NOT NULL,
			partition_name text NOT NULL DEFAULT '',
			account_name text NOT NULL DEFAULT '',
			script_path text NOT NULL,
			cpus integer NOT NULL,
			memory_mb integer NOT NULL,
			gpus integer NOT NULL,
			working_dir text NOT NULL DEFAULT '',
			environment jsonb NOT NULL DEFAULT '{}'::jsonb,
			state text NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			started_at timestamptz,
			finished_at timestamptz
		)`,
		`CREATE INDEX IF NOT EXISTS hpc_jobs_tenant_created_idx ON hpc_jobs (tenant_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS hpc_job_logs (
			id bigserial PRIMARY KEY,
			job_id text NOT NULL REFERENCES hpc_jobs(id) ON DELETE CASCADE,
			source text NOT NULL,
			content text NOT NULL,
			created_at timestamptz NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS hpc_job_logs_job_created_idx ON hpc_job_logs (job_id, created_at, id)`,
		`CREATE TABLE IF NOT EXISTS hpc_job_events (
			id bigserial PRIMARY KEY,
			event_type text NOT NULL,
			job_id text NOT NULL,
			tenant_id text NOT NULL,
			payload jsonb NOT NULL DEFAULT '{}'::jsonb,
			created_at timestamptz NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS hpc_job_events_tenant_created_idx ON hpc_job_events (tenant_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS hpc_datasets (
			id text PRIMARY KEY,
			tenant_id text NOT NULL,
			name text NOT NULL,
			size_bytes bigint NOT NULL,
			content_type text NOT NULL,
			storage_path text NOT NULL,
			labels jsonb NOT NULL DEFAULT '{}'::jsonb,
			created_at timestamptz NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS hpc_datasets_tenant_created_idx ON hpc_datasets (tenant_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS hpc_nodes (
			id text PRIMARY KEY,
			hostname text NOT NULL,
			address text NOT NULL DEFAULT '',
			role text NOT NULL,
			state text NOT NULL,
			cpus integer NOT NULL DEFAULT 0,
			memory_gb integer NOT NULL DEFAULT 0,
			gpus integer NOT NULL DEFAULT 0,
			labels jsonb NOT NULL DEFAULT '{}'::jsonb,
			partitions jsonb NOT NULL DEFAULT '[]'::jsonb,
			last_seen_at timestamptz,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			health_score double precision NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS hpc_partitions (
			name text PRIMARY KEY,
			description text NOT NULL DEFAULT '',
			nodes jsonb NOT NULL DEFAULT '[]'::jsonb,
			labels jsonb NOT NULL DEFAULT '{}'::jsonb,
			max_cpus integer NOT NULL DEFAULT 0,
			max_gpus integer NOT NULL DEFAULT 0,
			active boolean NOT NULL DEFAULT false,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL
		)`,
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin HPC schema migration: %w", err)
	}
	defer tx.Rollback()
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply HPC schema migration: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit HPC schema migration: %w", err)
	}
	return nil
}

func (p *Postgres) SaveJob(ctx context.Context, job *jobs.Job) error {
	if job == nil || job.ID == "" || job.TenantID == "" {
		return errors.New("job, job ID, and tenant ID are required")
	}
	environment, err := jsonValue(job.Env, map[string]string{})
	if err != nil {
		return fmt.Errorf("encode job environment: %w", err)
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO hpc_jobs (
		id, tenant_id, user_id, name, backend, scheduler_id, execution_profile,
		partition_name, account_name, script_path, cpus, memory_mb, gpus,
		working_dir, environment, state, created_at, updated_at, started_at, finished_at
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
	) ON CONFLICT (id) DO UPDATE SET
		tenant_id=EXCLUDED.tenant_id, user_id=EXCLUDED.user_id, name=EXCLUDED.name,
		backend=EXCLUDED.backend, scheduler_id=EXCLUDED.scheduler_id,
		execution_profile=EXCLUDED.execution_profile, partition_name=EXCLUDED.partition_name,
		account_name=EXCLUDED.account_name, script_path=EXCLUDED.script_path,
		cpus=EXCLUDED.cpus, memory_mb=EXCLUDED.memory_mb, gpus=EXCLUDED.gpus,
		working_dir=EXCLUDED.working_dir, environment=EXCLUDED.environment,
		state=EXCLUDED.state, updated_at=EXCLUDED.updated_at,
		started_at=EXCLUDED.started_at, finished_at=EXCLUDED.finished_at`,
		job.ID, job.TenantID, job.UserID, job.Name, job.Backend, job.SchedulerID,
		job.ExecutionProfile, job.Partition, job.Account, job.ScriptPath,
		job.CPUs, job.MemoryMB, job.GPUs, job.WorkingDir, environment, job.State,
		job.CreatedAt, job.UpdatedAt, job.StartedAt, job.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("save HPC job %q: %w", job.ID, err)
	}
	return nil
}

func (p *Postgres) GetJob(ctx context.Context, id string) (*jobs.Job, error) {
	job, err := scanJob(p.db.QueryRowContext(ctx, `SELECT
		id, tenant_id, user_id, name, backend, scheduler_id, execution_profile,
		partition_name, account_name, script_path, cpus, memory_mb, gpus,
		working_dir, environment, state, created_at, updated_at, started_at, finished_at
		FROM hpc_jobs WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("HPC job %q: %w", id, ErrRecordNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get HPC job %q: %w", id, err)
	}
	return job, nil
}

func (p *Postgres) ListJobsByTenant(ctx context.Context, tenantID string) ([]*jobs.Job, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT
		id, tenant_id, user_id, name, backend, scheduler_id, execution_profile,
		partition_name, account_name, script_path, cpus, memory_mb, gpus,
		working_dir, environment, state, created_at, updated_at, started_at, finished_at
		FROM hpc_jobs WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list HPC jobs for tenant %q: %w", tenantID, err)
	}
	defer rows.Close()
	result := make([]*jobs.Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan HPC job: %w", err)
		}
		result = append(result, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate HPC jobs: %w", err)
	}
	return result, nil
}

func (p *Postgres) SaveLog(ctx context.Context, entry *jobs.JobLog) error {
	if entry == nil || entry.JobID == "" {
		return errors.New("job log and job ID are required")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO hpc_job_logs (job_id, source, content, created_at)
		VALUES ($1,$2,$3,$4)`, entry.JobID, entry.Source, entry.Content, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("save HPC job log for %q: %w", entry.JobID, err)
	}
	return nil
}

func (p *Postgres) ListLogs(ctx context.Context, jobID string) ([]*jobs.JobLog, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT job_id, source, content, created_at
		FROM hpc_job_logs WHERE job_id=$1 ORDER BY created_at, id`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list HPC job logs for %q: %w", jobID, err)
	}
	defer rows.Close()
	result := make([]*jobs.JobLog, 0)
	for rows.Next() {
		entry := new(jobs.JobLog)
		if err := rows.Scan(&entry.JobID, &entry.Source, &entry.Content, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan HPC job log: %w", err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate HPC job logs: %w", err)
	}
	return result, nil
}

func (p *Postgres) Publish(ctx context.Context, event jobs.Event) error {
	payload, err := jsonValue(event.Payload, map[string]any{})
	if err != nil {
		return fmt.Errorf("encode HPC job event: %w", err)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO hpc_job_events
		(event_type, job_id, tenant_id, payload, created_at) VALUES ($1,$2,$3,$4,$5)`,
		event.Type, event.JobID, event.TenantID, payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("publish HPC job event %q: %w", event.Type, err)
	}
	return nil
}

func (p *Postgres) Save(ctx context.Context, dataset *datasets.Dataset) error {
	if dataset == nil || dataset.ID == "" || dataset.TenantID == "" {
		return errors.New("dataset, dataset ID, and tenant ID are required")
	}
	labels, err := jsonValue(dataset.Labels, map[string]string{})
	if err != nil {
		return fmt.Errorf("encode HPC dataset labels: %w", err)
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO hpc_datasets
		(id, tenant_id, name, size_bytes, content_type, storage_path, labels, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id, name=EXCLUDED.name,
		size_bytes=EXCLUDED.size_bytes, content_type=EXCLUDED.content_type,
		storage_path=EXCLUDED.storage_path, labels=EXCLUDED.labels`,
		dataset.ID, dataset.TenantID, dataset.Name, dataset.SizeBytes, dataset.ContentType,
		dataset.StoragePath, labels, dataset.CreatedAt)
	if err != nil {
		return fmt.Errorf("save HPC dataset %q: %w", dataset.ID, err)
	}
	return nil
}

func (p *Postgres) Get(ctx context.Context, id string) (*datasets.Dataset, error) {
	result := new(datasets.Dataset)
	var labels []byte
	err := p.db.QueryRowContext(ctx, `SELECT id, tenant_id, name, size_bytes, content_type,
		storage_path, labels, created_at FROM hpc_datasets WHERE id=$1`, id).Scan(
		&result.ID, &result.TenantID, &result.Name, &result.SizeBytes, &result.ContentType,
		&result.StoragePath, &labels, &result.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("HPC dataset %q: %w", id, ErrRecordNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get HPC dataset %q: %w", id, err)
	}
	if err := json.Unmarshal(labels, &result.Labels); err != nil {
		return nil, fmt.Errorf("decode HPC dataset labels: %w", err)
	}
	return result, nil
}

func (p *Postgres) ListByTenant(ctx context.Context, tenantID string) ([]*datasets.Dataset, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, tenant_id, name, size_bytes, content_type,
		storage_path, labels, created_at FROM hpc_datasets WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list HPC datasets for tenant %q: %w", tenantID, err)
	}
	defer rows.Close()
	result := make([]*datasets.Dataset, 0)
	for rows.Next() {
		dataset := new(datasets.Dataset)
		var labels []byte
		if err := rows.Scan(&dataset.ID, &dataset.TenantID, &dataset.Name, &dataset.SizeBytes,
			&dataset.ContentType, &dataset.StoragePath, &labels, &dataset.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan HPC dataset: %w", err)
		}
		if err := json.Unmarshal(labels, &dataset.Labels); err != nil {
			return nil, fmt.Errorf("decode HPC dataset labels: %w", err)
		}
		result = append(result, dataset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate HPC datasets: %w", err)
	}
	return result, nil
}

func (p *Postgres) SaveNode(ctx context.Context, node nodes.Node) error {
	if node.ID == "" || node.Hostname == "" {
		return errors.New("node ID and hostname are required")
	}
	labels, err := jsonValue(node.Labels, map[string]string{})
	if err != nil {
		return fmt.Errorf("encode HPC node labels: %w", err)
	}
	partitions, err := jsonValue(node.Partitions, []string{})
	if err != nil {
		return fmt.Errorf("encode HPC node partitions: %w", err)
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO hpc_nodes
		(id, hostname, address, role, state, cpus, memory_gb, gpus, labels, partitions,
		last_seen_at, created_at, updated_at, health_score)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (id) DO UPDATE SET hostname=EXCLUDED.hostname, address=EXCLUDED.address,
		role=EXCLUDED.role, state=EXCLUDED.state, cpus=EXCLUDED.cpus, memory_gb=EXCLUDED.memory_gb,
		gpus=EXCLUDED.gpus, labels=EXCLUDED.labels, partitions=EXCLUDED.partitions,
		last_seen_at=EXCLUDED.last_seen_at, updated_at=EXCLUDED.updated_at,
		health_score=EXCLUDED.health_score`,
		node.ID, node.Hostname, node.Address, node.Role, node.State, node.CPUs,
		node.MemoryGB, node.GPUs, labels, partitions, node.LastSeenAt, node.CreatedAt,
		node.UpdatedAt, node.HealthScore)
	if err != nil {
		return fmt.Errorf("save HPC node %q: %w", node.ID, err)
	}
	return nil
}

func (p *Postgres) GetNode(ctx context.Context, id string) (nodes.Node, error) {
	var node nodes.Node
	var labels, partitions []byte
	err := p.db.QueryRowContext(ctx, `SELECT id, hostname, address, role, state, cpus,
		memory_gb, gpus, labels, partitions, last_seen_at, created_at, updated_at, health_score
		FROM hpc_nodes WHERE id=$1`, id).Scan(&node.ID, &node.Hostname, &node.Address,
		&node.Role, &node.State, &node.CPUs, &node.MemoryGB, &node.GPUs, &labels,
		&partitions, &node.LastSeenAt, &node.CreatedAt, &node.UpdatedAt, &node.HealthScore)
	if errors.Is(err, sql.ErrNoRows) {
		return nodes.Node{}, fmt.Errorf("HPC node %q: %w", id, nodes.ErrNodeNotFound)
	}
	if err != nil {
		return nodes.Node{}, fmt.Errorf("get HPC node %q: %w", id, err)
	}
	if err := json.Unmarshal(labels, &node.Labels); err != nil {
		return nodes.Node{}, fmt.Errorf("decode HPC node labels: %w", err)
	}
	if err := json.Unmarshal(partitions, &node.Partitions); err != nil {
		return nodes.Node{}, fmt.Errorf("decode HPC node partitions: %w", err)
	}
	return node, nil
}

func (p *Postgres) ListNodes(ctx context.Context) ([]nodes.Node, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, hostname, address, role, state, cpus,
		memory_gb, gpus, labels, partitions, last_seen_at, created_at, updated_at, health_score
		FROM hpc_nodes ORDER BY hostname`)
	if err != nil {
		return nil, fmt.Errorf("list HPC nodes: %w", err)
	}
	defer rows.Close()
	result := make([]nodes.Node, 0)
	for rows.Next() {
		var node nodes.Node
		var labels, partitions []byte
		if err := rows.Scan(&node.ID, &node.Hostname, &node.Address, &node.Role, &node.State,
			&node.CPUs, &node.MemoryGB, &node.GPUs, &labels, &partitions, &node.LastSeenAt,
			&node.CreatedAt, &node.UpdatedAt, &node.HealthScore); err != nil {
			return nil, fmt.Errorf("scan HPC node: %w", err)
		}
		if err := json.Unmarshal(labels, &node.Labels); err != nil {
			return nil, fmt.Errorf("decode HPC node labels: %w", err)
		}
		if err := json.Unmarshal(partitions, &node.Partitions); err != nil {
			return nil, fmt.Errorf("decode HPC node partitions: %w", err)
		}
		result = append(result, node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate HPC nodes: %w", err)
	}
	return result, nil
}

func (p *Postgres) SavePartition(ctx context.Context, partition nodes.Partition) error {
	if partition.Name == "" {
		return errors.New("partition name is required")
	}
	nodeIDs, err := jsonValue(partition.Nodes, []string{})
	if err != nil {
		return fmt.Errorf("encode HPC partition nodes: %w", err)
	}
	labels, err := jsonValue(partition.Labels, map[string]string{})
	if err != nil {
		return fmt.Errorf("encode HPC partition labels: %w", err)
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO hpc_partitions
		(name, description, nodes, labels, max_cpus, max_gpus, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (name) DO UPDATE SET description=EXCLUDED.description,
		nodes=EXCLUDED.nodes, labels=EXCLUDED.labels, max_cpus=EXCLUDED.max_cpus,
		max_gpus=EXCLUDED.max_gpus, active=EXCLUDED.active, updated_at=EXCLUDED.updated_at`,
		partition.Name, partition.Description, nodeIDs, labels, partition.MaxCPUs,
		partition.MaxGPUs, partition.Active, partition.CreatedAt, partition.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save HPC partition %q: %w", partition.Name, err)
	}
	return nil
}

func (p *Postgres) GetPartition(ctx context.Context, name string) (nodes.Partition, error) {
	var partition nodes.Partition
	var nodeIDs, labels []byte
	err := p.db.QueryRowContext(ctx, `SELECT name, description, nodes, labels, max_cpus,
		max_gpus, active, created_at, updated_at FROM hpc_partitions WHERE name=$1`, name).Scan(
		&partition.Name, &partition.Description, &nodeIDs, &labels, &partition.MaxCPUs,
		&partition.MaxGPUs, &partition.Active, &partition.CreatedAt, &partition.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nodes.Partition{}, fmt.Errorf("HPC partition %q: %w", name, nodes.ErrPartitionNotFound)
	}
	if err != nil {
		return nodes.Partition{}, fmt.Errorf("get HPC partition %q: %w", name, err)
	}
	if err := json.Unmarshal(nodeIDs, &partition.Nodes); err != nil {
		return nodes.Partition{}, fmt.Errorf("decode HPC partition nodes: %w", err)
	}
	if err := json.Unmarshal(labels, &partition.Labels); err != nil {
		return nodes.Partition{}, fmt.Errorf("decode HPC partition labels: %w", err)
	}
	return partition, nil
}

func (p *Postgres) ListPartitions(ctx context.Context) ([]nodes.Partition, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT name, description, nodes, labels, max_cpus,
		max_gpus, active, created_at, updated_at FROM hpc_partitions ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list HPC partitions: %w", err)
	}
	defer rows.Close()
	result := make([]nodes.Partition, 0)
	for rows.Next() {
		var partition nodes.Partition
		var nodeIDs, labels []byte
		if err := rows.Scan(&partition.Name, &partition.Description, &nodeIDs, &labels,
			&partition.MaxCPUs, &partition.MaxGPUs, &partition.Active,
			&partition.CreatedAt, &partition.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan HPC partition: %w", err)
		}
		if err := json.Unmarshal(nodeIDs, &partition.Nodes); err != nil {
			return nil, fmt.Errorf("decode HPC partition nodes: %w", err)
		}
		if err := json.Unmarshal(labels, &partition.Labels); err != nil {
			return nil, fmt.Errorf("decode HPC partition labels: %w", err)
		}
		result = append(result, partition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate HPC partitions: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (*jobs.Job, error) {
	job := new(jobs.Job)
	var environment []byte
	var startedAt, finishedAt sql.NullTime
	err := row.Scan(&job.ID, &job.TenantID, &job.UserID, &job.Name, &job.Backend,
		&job.SchedulerID, &job.ExecutionProfile, &job.Partition, &job.Account,
		&job.ScriptPath, &job.CPUs, &job.MemoryMB, &job.GPUs, &job.WorkingDir,
		&environment, &job.State, &job.CreatedAt, &job.UpdatedAt, &startedAt, &finishedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(environment, &job.Env); err != nil {
		return nil, fmt.Errorf("decode job environment: %w", err)
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	return job, nil
}

func jsonValue(value, fallback any) ([]byte, error) {
	if value == nil {
		value = fallback
	}
	return json.Marshal(value)
}

var (
	_ jobs.Repository             = (*Postgres)(nil)
	_ jobs.EventPublisher         = (*Postgres)(nil)
	_ datasets.MetadataRepository = (*Postgres)(nil)
	_ nodes.InventoryRepository   = (*Postgres)(nil)
)
