package core

import "fmt"
import (
	"regexp"
	"strings"
)

type Policy struct{}

var databaseNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

func NewPolicy() *Policy {
	return &Policy{}
}

func (p *Policy) ValidateCreate(req CreateInstanceRequest) error {
	if req.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !databaseNamePattern.MatchString(strings.ToLower(strings.TrimSpace(req.Name))) {
		return fmt.Errorf("name must be 2-63 lowercase letters, numbers, or hyphens")
	}

	switch req.Engine {
	case EnginePostgres, EngineMySQL, EngineMariaDB, EngineRedis:
	default:
		return fmt.Errorf("unsupported engine: %s", req.Engine)
	}

	switch req.WorkloadProfile {
	case "", "general-purpose", "transaction-processing", "analytics", "ai-ml", "hpc":
	default:
		return fmt.Errorf("unsupported workload profile: %s", req.WorkloadProfile)
	}
	if req.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	return nil
}
