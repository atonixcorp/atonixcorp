package nodes

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrPartitionNotFound = errors.New("partition not found")
)

type InventoryRepository interface {
	SaveNode(ctx context.Context, node Node) error
	GetNode(ctx context.Context, id string) (Node, error)
	ListNodes(ctx context.Context) ([]Node, error)
	SavePartition(ctx context.Context, p Partition) error
	GetPartition(ctx context.Context, name string) (Partition, error)
	ListPartitions(ctx context.Context) ([]Partition, error)
}

type InventoryService struct {
	repo InventoryRepository
	mu   sync.RWMutex
}

func NewInventoryService(repo InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) RegisterNode(ctx context.Context, node Node) error {
	if s == nil || s.repo == nil {
		return errors.New("node inventory repository is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if node.CreatedAt.IsZero() {
		node.CreatedAt = now
	}
	node.UpdatedAt = now
	node.LastSeenAt = now

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	return s.repo.SaveNode(ctx, node)
}

func (s *InventoryService) RegisterPartition(ctx context.Context, partition Partition) error {
	if s == nil || s.repo == nil {
		return errors.New("node inventory repository is not configured")
	}
	now := time.Now().UTC()
	if partition.CreatedAt.IsZero() {
		partition.CreatedAt = now
	}
	partition.UpdatedAt = now
	if partition.Labels == nil {
		partition.Labels = make(map[string]string)
	}
	return s.repo.SavePartition(ctx, partition)
}

func (s *InventoryService) UpdateNodeState(ctx context.Context, id string, state NodeState) error {
	if s == nil || s.repo == nil {
		return errors.New("node inventory repository is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.repo.GetNode(ctx, id)
	if err != nil {
		return err
	}

	node.State = state
	node.UpdatedAt = time.Now().UTC()
	return s.repo.SaveNode(ctx, node)
}

func (s *InventoryService) TouchNode(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return errors.New("node inventory repository is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.repo.GetNode(ctx, id)
	if err != nil {
		return err
	}

	node.LastSeenAt = time.Now().UTC()
	node.UpdatedAt = node.LastSeenAt
	return s.repo.SaveNode(ctx, node)
}

func (s *InventoryService) ListAllNodes(ctx context.Context) ([]Node, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("node inventory repository is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repo.ListNodes(ctx)
}

func (s *InventoryService) ListPartitions(ctx context.Context) ([]Partition, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("node inventory repository is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repo.ListPartitions(ctx)
}

func (s *InventoryService) ListHealthyNodes(ctx context.Context, minHealth float64) ([]Node, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("node inventory repository is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes, err := s.repo.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	var result []Node
	for _, n := range nodes {
		if n.HealthScore >= minHealth && n.State == NodeStateOnline {
			result = append(result, n)
		}
	}
	return result, nil
}
