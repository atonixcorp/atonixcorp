package nodes

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type HealthProbeResult struct {
	NodeID      string
	Reachable   bool
	CPUUsage    float64
	MemoryUsage float64
	GPUUsage    float64
	TempOK      bool
	Timestamp   time.Time
}

type HealthProbeClient interface {
	ProbeNode(ctx context.Context, node Node) (HealthProbeResult, error)
}

type HealthService struct {
	inventory *InventoryService
	prober    HealthProbeClient
}

func NewHealthService(inv *InventoryService, prober HealthProbeClient) *HealthService {
	return &HealthService{
		inventory: inv,
		prober:    prober,
	}
}

func (s *HealthService) EvaluateNodeHealth(ctx context.Context, node Node) (float64, error) {
	if s == nil || s.inventory == nil || s.inventory.repo == nil || s.prober == nil {
		return 0, errors.New("node health service is not configured")
	}
	res, err := s.prober.ProbeNode(ctx, node)
	if err != nil {
		return 0, fmt.Errorf("probe node %q: %w", node.ID, err)
	}

	score := 1.0

	if !res.Reachable {
		score = 0.0
	} else {
		if res.CPUUsage > 0.95 {
			score -= 0.2
		}
		if res.MemoryUsage > 0.95 {
			score -= 0.2
		}
		if res.GPUUsage > 0.95 {
			score -= 0.2
		}
		if !res.TempOK {
			score -= 0.2
		}
		if score < 0 {
			score = 0
		}
	}

	node.HealthScore = score
	if !res.Timestamp.IsZero() {
		node.LastSeenAt = res.Timestamp
	}
	node.UpdatedAt = time.Now().UTC()

	if err := s.inventory.repo.SaveNode(ctx, node); err != nil {
		return 0, fmt.Errorf("save health state for node %q: %w", node.ID, err)
	}

	return score, nil
}

func (s *HealthService) RefreshAllNodeHealth(ctx context.Context) error {
	nodes, err := s.inventory.ListAllNodes(ctx)
	if err != nil {
		return err
	}

	var failures []error
	for _, n := range nodes {
		if _, err := s.EvaluateNodeHealth(ctx, n); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
