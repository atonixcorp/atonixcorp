package jobs

import (
	"context"
	"fmt"
	"time"

	"atonixcorp/cloud/hpc/scheduler"
	"github.com/google/uuid"
)

type BackendSelector interface {
	SelectBackend(tenantID, executionProfile string) (ExecutionTarget, error)
}

type ExecutionTarget struct {
	Scheduler scheduler.Scheduler
	Backend   string
	Partition string
	Account   string
}

type ProfileSelector struct{ profiles map[string]ExecutionTarget }

func NewProfileSelector(profiles map[string]ExecutionTarget) *ProfileSelector {
	copyOfProfiles := make(map[string]ExecutionTarget, len(profiles))
	for name, target := range profiles {
		copyOfProfiles[name] = target
	}
	return &ProfileSelector{profiles: copyOfProfiles}
}

func (selector *ProfileSelector) SelectBackend(tenantID, profile string) (ExecutionTarget, error) {
	if selector == nil || tenantID == "" {
		return ExecutionTarget{}, ErrExecutionProfileUnavailable
	}
	target, ok := selector.profiles[profile]
	if !ok || target.Scheduler == nil || target.Backend == "" {
		return ExecutionTarget{}, ErrExecutionProfileUnavailable
	}
	return target, nil
}

type Service struct {
	repo     Repository
	events   EventPublisher
	selector BackendSelector
}

func NewService(repo Repository, events EventPublisher, selector BackendSelector) *Service {
	return &Service{
		repo:     repo,
		events:   events,
		selector: selector,
	}
}

func (s *Service) CreateAndSubmit(ctx context.Context, req CreateJobRequest) (*Job, error) {
	if err := ValidateCreateJob(req); err != nil {
		return nil, err
	}

	target, err := s.selector.SelectBackend(req.TenantID, req.ExecutionProfile)
	if err != nil {
		return nil, err
	}
	if target.Scheduler == nil {
		return nil, ErrExecutionProfileUnavailable
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate job ID: %w", err)
	}

	now := time.Now().UTC()
	job := &Job{
		ID:               id.String(),
		TenantID:         req.TenantID,
		UserID:           req.UserID,
		Name:             req.Name,
		Backend:          target.Backend,
		ExecutionProfile: req.ExecutionProfile,
		Partition:        target.Partition,
		Account:          target.Account,
		ScriptPath:       req.ScriptPath,
		CPUs:             req.CPUs,
		MemoryMB:         req.MemoryMB,
		GPUs:             req.GPUs,
		WorkingDir:       req.WorkingDir,
		Env:              req.Env,
		State:            JobStatePending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	if s.events != nil {
		if err := s.events.Publish(ctx, Event{
			Type:      EventJobCreated,
			JobID:     job.ID,
			TenantID:  job.TenantID,
			Timestamp: now,
			Payload:   map[string]any{"name": job.Name},
		}); err != nil {
			return nil, fmt.Errorf("job %s was persisted but its created event could not be published: %w", job.ID, err)
		}
	}

	spec := scheduler.JobSpec{
		JobName:    job.ID,
		ScriptPath: job.ScriptPath,
		Partition:  job.Partition,
		Account:    job.Account,
		TimeLimit:  "", // extend later
		CPUs:       job.CPUs,
		MemoryMB:   job.MemoryMB,
		GPUs:       job.GPUs,
		WorkingDir: job.WorkingDir,
		Env:        job.Env,
	}

	schedulerID, err := target.Scheduler.Submit(ctx, spec)
	if err != nil {
		job.State = JobStateFailed
		job.UpdatedAt = time.Now().UTC()
		if saveErr := s.repo.SaveJob(ctx, job); saveErr != nil {
			return nil, fmt.Errorf("submit job: %w (also failed to persist failed state: %v)", err, saveErr)
		}
		return nil, err
	}

	job.SchedulerID = schedulerID
	job.State = JobStateQueued
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	if s.events != nil {
		if err := s.events.Publish(ctx, Event{
			Type:      EventJobQueued,
			JobID:     job.ID,
			TenantID:  job.TenantID,
			Timestamp: time.Now().UTC(),
			Payload:   map[string]any{"scheduler_id": schedulerID},
		}); err != nil {
			return nil, fmt.Errorf("job %s was queued but its queued event could not be published: %w", job.ID, err)
		}
	}

	return job, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Job, error) {
	return s.repo.GetJob(ctx, id)
}

func (s *Service) GetForTenant(ctx context.Context, id, tenantID string) (*Job, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.TenantID != tenantID {
		return nil, ErrTenantAccessDenied
	}
	return job, nil
}

func (s *Service) RefreshStatusForTenant(ctx context.Context, id, tenantID string) (*Job, error) {
	if _, err := s.GetForTenant(ctx, id, tenantID); err != nil {
		return nil, err
	}
	return s.RefreshStatus(ctx, id)
}

func (s *Service) RefreshStatus(ctx context.Context, id string) (*Job, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}

	target, err := s.selector.SelectBackend(job.TenantID, job.ExecutionProfile)
	if err != nil {
		return nil, err
	}
	if target.Scheduler == nil || target.Backend != job.Backend || target.Partition != job.Partition || target.Account != job.Account {
		return nil, ErrExecutionProfileUnavailable
	}

	status, err := target.Scheduler.Status(ctx, job.SchedulerID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job.UpdatedAt = now

	switch status.State {
	case "PENDING":
		job.State = JobStateQueued
	case "RUNNING":
		job.State = JobStateRunning
		if job.StartedAt == nil {
			job.StartedAt = &now
		}
	case "COMPLETED":
		job.State = JobStateSucceeded
		if job.FinishedAt == nil {
			job.FinishedAt = &now
		}
	case "FAILED":
		job.State = JobStateFailed
		if job.FinishedAt == nil {
			job.FinishedAt = &now
		}
	case "CANCELLED":
		job.State = JobStateCanceled
		if job.FinishedAt == nil {
			job.FinishedAt = &now
		}
	}

	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Service) Cancel(ctx context.Context, id string) (*Job, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}

	target, err := s.selector.SelectBackend(job.TenantID, job.ExecutionProfile)
	if err != nil {
		return nil, err
	}
	if target.Scheduler == nil || target.Backend != job.Backend || target.Partition != job.Partition || target.Account != job.Account {
		return nil, ErrExecutionProfileUnavailable
	}

	if err := target.Scheduler.Cancel(ctx, job.SchedulerID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job.State = JobStateCanceled
	job.UpdatedAt = now
	job.FinishedAt = &now

	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	if s.events != nil {
		if err := s.events.Publish(ctx, Event{
			Type:      EventJobCanceled,
			JobID:     job.ID,
			TenantID:  job.TenantID,
			Timestamp: now,
			Payload:   map[string]any{},
		}); err != nil {
			return nil, fmt.Errorf("job %s was canceled but its cancellation event could not be published: %w", job.ID, err)
		}
	}

	return job, nil
}
