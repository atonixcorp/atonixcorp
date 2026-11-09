package mpi

import "fmt"

// JobMPIConfig represents MPI settings for a scheduled job.
type JobMPIConfig struct {
	ImplementationName string
	Nodes              []string
	TasksPerNode       int
}

// GenerateEnv generates environment hints for the scheduler / job launcher.
func GenerateEnv(cfg JobMPIConfig, svc *MPIService) (map[string]string, error) {
	if svc == nil {
		return nil, fmt.Errorf("MPI service is not configured")
	}
	impl := svc.Get(cfg.ImplementationName)
	if impl == nil {
		return nil, fmt.Errorf("MPI implementation '%s' not found", cfg.ImplementationName)
	}
	if len(cfg.Nodes) == 0 || cfg.TasksPerNode <= 0 {
		return nil, fmt.Errorf("MPI node list and positive tasks per node are required")
	}

	env := map[string]string{
		"MPI_IMPL":           impl.Name,
		"MPI_VERSION":        impl.Version,
		"MPI_MODULE":         impl.Module,
		"MPI_CONTAINER":      impl.Container,
		"MPI_NODES":          fmt.Sprintf("%v", cfg.Nodes),
		"MPI_TASKS_PER_NODE": fmt.Sprintf("%d", cfg.TasksPerNode),
	}

	return env, nil
}
