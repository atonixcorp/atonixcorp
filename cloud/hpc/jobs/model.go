package jobs

import (
	"time"
)

type JobState string

const (
	JobStatePending   JobState = "PENDING"
	JobStateQueued    JobState = "QUEUED"
	JobStateRunning   JobState = "RUNNING"
	JobStateSucceeded JobState = "SUCCEEDED"
	JobStateFailed    JobState = "FAILED"
	JobStateCanceled  JobState = "CANCELED"
)

type Job struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	UserID           string            `json:"user_id"`
	Name             string            `json:"name"`
	Backend          string            `json:"-"`
	SchedulerID      string            `json:"-"`
	ExecutionProfile string            `json:"execution_profile"`
	Partition        string            `json:"-"`
	Account          string            `json:"-"`
	ScriptPath       string            `json:"script_path"`
	CPUs             int               `json:"cpus"`
	MemoryMB         int               `json:"memory_mb"`
	GPUs             int               `json:"gpus"`
	WorkingDir       string            `json:"working_dir"`
	Env              map[string]string `json:"env"`
	State            JobState          `json:"state"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	FinishedAt       *time.Time        `json:"finished_at,omitempty"`
}

type JobLog struct {
	JobID     string    `json:"job_id"`
	Source    string    `json:"source"` // "stdout", "stderr", "system"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
