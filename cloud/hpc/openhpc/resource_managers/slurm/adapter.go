package slurm

import (
	"context"

	"atonixcorp/cloud/hpc/scheduler"
)

type SlurmAdapter struct {
	client *SlurmClient
}

func NewSlurmAdapter() *SlurmAdapter {
	return &SlurmAdapter{
		client: NewSlurmClient(),
	}
}

func NewSlurmAdapterWithClient(client *SlurmClient) *SlurmAdapter {
	return &SlurmAdapter{client: client}
}

func (a *SlurmAdapter) Submit(ctx context.Context, spec scheduler.JobSpec) (string, error) {
	opts := SubmitOptions{
		Partition:  spec.Partition,
		Account:    spec.Account,
		TimeLimit:  spec.TimeLimit,
		CPUs:       spec.CPUs,
		MemoryMB:   spec.MemoryMB,
		GPUs:       spec.GPUs,
		JobName:    spec.JobName,
		ScriptPath: spec.ScriptPath,
		WorkingDir: spec.WorkingDir,
		Env:        spec.Env,
	}

	res, err := a.client.SubmitJob(ctx, opts)
	if err != nil {
		return "", err
	}

	return res.JobID, nil
}

func (a *SlurmAdapter) Status(ctx context.Context, jobID string) (*scheduler.JobStatus, error) {
	st, err := a.client.GetJobStatus(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &scheduler.JobStatus{
		JobID:     st.JobID,
		Name:      st.Name,
		State:     string(st.State),
		Partition: st.Partition,
		User:      st.User,
		Nodes:     st.Nodes,
		Raw:       st.Raw,
	}, nil
}

func (a *SlurmAdapter) Cancel(ctx context.Context, jobID string) error {
	return a.client.CancelJob(ctx, jobID)
}
