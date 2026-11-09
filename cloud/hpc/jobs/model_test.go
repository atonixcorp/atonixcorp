package jobs

import (
	"atonixcorp/cloud/hpc/scheduler"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJobJSONHidesSchedulerImplementationDetails(t *testing.T) {
	payload, err := json.Marshal(Job{
		ID: "atonix-job", Backend: "slurm", SchedulerID: "12345",
		Partition: "private-partition", Account: "private-account", ExecutionProfile: "batch",
	})
	if err != nil {
		t.Fatalf("marshal HPC job: %v", err)
	}
	for _, privateValue := range []string{"slurm", `"backend"`, "12345", `"scheduler_id"`, "private-partition", "private-account"} {
		if strings.Contains(string(payload), privateValue) {
			t.Fatalf("HPC job response leaked %q: %s", privateValue, payload)
		}
	}
	if !strings.Contains(string(payload), `"id":"atonix-job"`) {
		t.Fatalf("HPC job response omitted its AtonixCorp ID: %s", payload)
	}
}

type boundaryScheduler struct{ spec scheduler.JobSpec }

func (backend *boundaryScheduler) Submit(_ context.Context, spec scheduler.JobSpec) (string, error) {
	backend.spec = spec
	return "private-job-id", nil
}
func (*boundaryScheduler) Status(context.Context, string) (*scheduler.JobStatus, error) {
	return &scheduler.JobStatus{State: "RUNNING"}, nil
}
func (*boundaryScheduler) Cancel(context.Context, string) error { return nil }

type boundarySelector struct{ target ExecutionTarget }

func (selector *boundarySelector) SelectBackend(tenantID, profile string) (ExecutionTarget, error) {
	if tenantID != "tenant" || profile != "batch" {
		return ExecutionTarget{}, ErrExecutionProfileUnavailable
	}
	return selector.target, nil
}

type boundaryRepository struct {
	Repository
	job *Job
}

func (repo *boundaryRepository) SaveJob(_ context.Context, job *Job) error {
	repo.job = job
	return nil
}
func (repo *boundaryRepository) GetJob(context.Context, string) (*Job, error) { return repo.job, nil }

type boundaryEvents struct{}

func (boundaryEvents) Publish(context.Context, Event) error { return nil }

func TestDomainRequestResolvesPrivateSchedulerConfiguration(t *testing.T) {
	backend := &boundaryScheduler{}
	selector := &boundarySelector{ExecutionTarget{Scheduler: backend, Backend: "test-only", Partition: "private-partition", Account: "private-account"}}
	service := NewService(&boundaryRepository{}, boundaryEvents{}, selector)
	job, err := service.CreateAndSubmit(context.Background(), CreateJobRequest{
		TenantID: "tenant", UserID: "user", Name: "simulation", ExecutionProfile: "batch",
		ScriptPath: "/work/run.sh", CPUs: 4, MemoryMB: 8192, GPUs: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if backend.spec.Partition != "private-partition" || backend.spec.Account != "private-account" || backend.spec.GPUs != 1 || job.State != JobStateQueued {
		t.Fatalf("incorrect scheduler mapping: %+v / %+v", backend.spec, job)
	}
	if _, err := service.RefreshStatus(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	selector.target.Account = "another-account"
	if _, err := service.Cancel(context.Background(), job.ID); !errors.Is(err, ErrExecutionProfileUnavailable) {
		t.Fatalf("changed scheduler binding should fail closed: %v", err)
	}
}

func TestProfileSelectorRejectsUnavailableProfilesAndCopiesConfiguration(t *testing.T) {
	profiles := map[string]ExecutionTarget{"batch": {Scheduler: &boundaryScheduler{}, Backend: "test-only", Partition: "private"}}
	selector := NewProfileSelector(profiles)
	profiles["batch"] = ExecutionTarget{}
	target, err := selector.SelectBackend("tenant", "batch")
	if err != nil || target.Partition != "private" {
		t.Fatalf("profile configuration changed: %v", err)
	}
	for _, profile := range []string{"unknown", "slurm"} {
		if _, err := selector.SelectBackend("tenant", profile); !errors.Is(err, ErrExecutionProfileUnavailable) {
			t.Fatal("unconfigured scheduler name accepted as profile")
		}
	}
	if _, err := NewProfileSelector(map[string]ExecutionTarget{"disabled": {Backend: "test-only"}}).SelectBackend("tenant", "disabled"); !errors.Is(err, ErrExecutionProfileUnavailable) {
		t.Fatal("disabled profile accepted")
	}
}

func TestCreateJobRequestUsesDomainExecutionProfile(t *testing.T) {
	request := CreateJobRequest{
		TenantID: "tenant", UserID: "user", Name: "simulation",
		ExecutionProfile: "batch", ScriptPath: "/work/run.sh", CPUs: 4, MemoryMB: 8192,
	}
	if err := ValidateCreateJob(request); err != nil {
		t.Fatalf("domain request required scheduler credentials: %v", err)
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateField := range []string{`"backend"`, `"partition"`, `"account"`} {
		if strings.Contains(string(payload), privateField) {
			t.Fatalf("request leaked scheduler field %s", privateField)
		}
	}
	request.ExecutionProfile = ""
	if ValidateCreateJob(request) != ErrInvalidJobSpec {
		t.Fatal("missing execution profile accepted")
	}
}

func TestTenantScopedJobReadsRejectOtherTenants(t *testing.T) {
	repo := &boundaryRepository{job: &Job{ID: "job-id", TenantID: "tenant-a"}}
	service := NewService(repo, nil, nil)
	if _, err := service.GetForTenant(context.Background(), "job-id", "tenant-b"); !errors.Is(err, ErrTenantAccessDenied) {
		t.Fatalf("cross-tenant job read should fail closed, got %v", err)
	}
	if _, err := NewLogService(repo).ListForTenant(context.Background(), "job-id", "tenant-b"); !errors.Is(err, ErrTenantAccessDenied) {
		t.Fatalf("cross-tenant log read should fail closed, got %v", err)
	}
}
