package repository

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"atonixcorp/cloud/hpc/datasets"
	"atonixcorp/cloud/hpc/jobs"
	"atonixcorp/cloud/hpc/nodes"
	"github.com/google/uuid"
)

func TestPostgresRepositoriesIntegration(t *testing.T) {
	dsn := integrationDSN()
	if dsn == "" {
		t.Skip("set HPC_TEST_DATABASE_URL or HPC_DB_* values to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	store, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close integration database: %v", err)
		}
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	jobID := uuid.NewString()
	datasetID := uuid.NewString()
	nodeID := uuid.NewString()
	partitionName := "test-" + uuid.NewString()
	tenantID := "integration-" + uuid.NewString()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := store.db.ExecContext(cleanupCtx, `DELETE FROM hpc_job_events WHERE job_id=$1`, jobID)
		if err != nil {
			t.Errorf("clean up integration job events: %v", err)
		}
		_, err = store.db.ExecContext(cleanupCtx, `DELETE FROM hpc_datasets WHERE id=$1`, datasetID)
		if err != nil {
			t.Errorf("clean up integration dataset: %v", err)
		}
		_, err = store.db.ExecContext(cleanupCtx, `DELETE FROM hpc_nodes WHERE id=$1`, nodeID)
		if err != nil {
			t.Errorf("clean up integration node: %v", err)
		}
		_, err = store.db.ExecContext(cleanupCtx, `DELETE FROM hpc_partitions WHERE name=$1`, partitionName)
		if err != nil {
			t.Errorf("clean up integration partition: %v", err)
		}
		_, err = store.db.ExecContext(cleanupCtx, `DELETE FROM hpc_jobs WHERE id=$1`, jobID)
		if err != nil {
			t.Errorf("clean up integration job: %v", err)
		}
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	job := &jobs.Job{
		ID: jobID, TenantID: tenantID, UserID: "integration-user", Name: "integration-job",
		Backend: "slurm", SchedulerID: "scheduler-id", ExecutionProfile: "batch",
		Partition: "batch", Account: "test-account", ScriptPath: "/tmp/job.sh",
		CPUs: 4, MemoryMB: 2048, GPUs: 1, WorkingDir: "/tmp",
		Env: map[string]string{"OMP_NUM_THREADS": "4"}, State: jobs.JobStatePending,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.SaveJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	storedJob, err := store.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if storedJob.TenantID != tenantID || storedJob.Name != job.Name ||
		storedJob.Env["OMP_NUM_THREADS"] != "4" || storedJob.State != jobs.JobStatePending {
		t.Fatalf("job did not round-trip: %+v", storedJob)
	}
	jobList, err := store.ListJobsByTenant(ctx, tenantID)
	if err != nil || len(jobList) != 1 || jobList[0].ID != jobID {
		t.Fatalf("tenant job list = %+v, error %v", jobList, err)
	}
	if err := store.SaveLog(ctx, &jobs.JobLog{JobID: jobID, Source: "stdout", Content: "ok", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	logs, err := store.ListLogs(ctx, jobID)
	if err != nil || len(logs) != 1 || logs[0].Content != "ok" {
		t.Fatalf("job logs = %+v, error %v", logs, err)
	}
	if err := store.Publish(ctx, jobs.Event{
		Type: jobs.EventJobCreated, JobID: jobID, TenantID: tenantID,
		Timestamp: now, Payload: map[string]any{"name": job.Name},
	}); err != nil {
		t.Fatal(err)
	}
	var eventCount int
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM hpc_job_events WHERE job_id=$1`, jobID).Scan(&eventCount); err != nil || eventCount != 1 {
		t.Fatalf("persisted event count = %d, error %v", eventCount, err)
	}

	dataset := &datasets.Dataset{
		ID: datasetID, TenantID: tenantID, Name: "dataset", SizeBytes: 12,
		ContentType: "application/octet-stream", StoragePath: tenantID + "/" + datasetID,
		Labels: map[string]string{"kind": "test"}, CreatedAt: now,
	}
	if err := store.Save(ctx, dataset); err != nil {
		t.Fatal(err)
	}
	storedDataset, err := store.Get(ctx, datasetID)
	if err != nil || storedDataset.Labels["kind"] != "test" {
		t.Fatalf("dataset did not round-trip: %+v, error %v", storedDataset, err)
	}
	datasetList, err := store.ListByTenant(ctx, tenantID)
	if err != nil || len(datasetList) != 1 || datasetList[0].ID != datasetID {
		t.Fatalf("tenant dataset list = %+v, error %v", datasetList, err)
	}

	node := nodes.Node{
		ID: nodeID, Hostname: "integration-node", Address: "127.0.0.1",
		Role: nodes.NodeRoleCompute, State: nodes.NodeStateOnline, CPUs: 8,
		MemoryGB: 32, Labels: map[string]string{"site": "test"},
		Partitions: []string{partitionName}, LastSeenAt: now, CreatedAt: now,
		UpdatedAt: now, HealthScore: 0.9,
	}
	if err := store.SaveNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	storedNode, err := store.GetNode(ctx, nodeID)
	if err != nil || storedNode.Labels["site"] != "test" || len(storedNode.Partitions) != 1 {
		t.Fatalf("node did not round-trip: %+v, error %v", storedNode, err)
	}
	nodeList, err := store.ListNodes(ctx)
	if err != nil || len(nodeList) == 0 {
		t.Fatalf("node list = %+v, error %v", nodeList, err)
	}
	partition := nodes.Partition{
		Name: partitionName, Description: "test partition",
		Nodes: []string{nodeID}, Labels: map[string]string{"site": "test"},
		MaxCPUs: 8, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.SavePartition(ctx, partition); err != nil {
		t.Fatal(err)
	}
	storedPartition, err := store.GetPartition(ctx, partitionName)
	if err != nil || !storedPartition.Active || len(storedPartition.Nodes) != 1 {
		t.Fatalf("partition did not round-trip: %+v, error %v", storedPartition, err)
	}
	partitionList, err := store.ListPartitions(ctx)
	if err != nil || len(partitionList) == 0 {
		t.Fatalf("partition list = %+v, error %v", partitionList, err)
	}

	if _, err := store.GetJob(ctx, uuid.NewString()); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("missing job should wrap ErrRecordNotFound, got %v", err)
	}
}

func integrationDSN() string {
	if dsn := os.Getenv("HPC_TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	user, password, database := os.Getenv("HPC_DB_USER"), os.Getenv("HPC_DB_PASSWORD"), os.Getenv("HPC_DB_NAME")
	if user == "" || password == "" || database == "" {
		return ""
	}
	host := os.Getenv("HPC_DB_HOST")
	if host == "" {
		host = "postgres"
	}
	port := os.Getenv("HPC_DB_PORT")
	if port == "" {
		port = "5432"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}
	query := url.Values{}
	query.Set("sslmode", os.Getenv("HPC_DB_SSLMODE"))
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
	}
	u.RawQuery = query.Encode()
	return u.String()
}
