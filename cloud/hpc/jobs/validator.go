package jobs

import "errors"

var (
	ErrInvalidJobSpec              = errors.New("invalid job specification")
	ErrExecutionProfileUnavailable = errors.New("execution profile is not configured")
)

type CreateJobRequest struct {
	TenantID         string            `json:"tenant_id"`
	UserID           string            `json:"user_id"`
	Name             string            `json:"name"`
	ExecutionProfile string            `json:"execution_profile"`
	ScriptPath       string            `json:"script_path"`
	CPUs             int               `json:"cpus"`
	MemoryMB         int               `json:"memory_mb"`
	GPUs             int               `json:"gpus"`
	WorkingDir       string            `json:"working_dir"`
	Env              map[string]string `json:"env"`
}

func ValidateCreateJob(req CreateJobRequest) error {
	if req.TenantID == "" || req.UserID == "" {
		return ErrInvalidJobSpec
	}
	if req.Name == "" {
		return ErrInvalidJobSpec
	}
	if req.ScriptPath == "" {
		return ErrInvalidJobSpec
	}
	if req.CPUs <= 0 {
		return ErrInvalidJobSpec
	}
	if req.MemoryMB <= 0 {
		return ErrInvalidJobSpec
	}
	if req.ExecutionProfile == "" || req.GPUs < 0 {
		return ErrInvalidJobSpec
	}
	return nil
}
