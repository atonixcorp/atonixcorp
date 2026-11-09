package nodes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type DiscoverySource string

const (
	DiscoverySourceSlurm DiscoverySource = "slurm"
	DiscoverySourceK8s   DiscoverySource = "k8s"
	DiscoverySourceAgent DiscoverySource = "agent"
)

type DiscoveryClient interface {
	DiscoverNodes(ctx context.Context) ([]Node, error)
}

type DiscoveryService struct {
	inventory *InventoryService
	client    DiscoveryClient
}

func NewDiscoveryService(inv *InventoryService, client DiscoveryClient) *DiscoveryService {
	return &DiscoveryService{
		inventory: inv,
		client:    client,
	}
}

func (s *DiscoveryService) SyncClusterNodes(ctx context.Context) error {
	if s == nil || s.inventory == nil || s.client == nil {
		return errors.New("node discovery service is not configured")
	}
	nodes, err := s.client.DiscoverNodes(ctx)
	if err != nil {
		return fmt.Errorf("discover cluster nodes: %w", err)
	}

	for _, node := range nodes {
		if err := s.inventory.RegisterNode(ctx, node); err != nil {
			return fmt.Errorf("register discovered node %q: %w", node.ID, err)
		}
	}
	if partitionDiscovery, ok := s.client.(interface {
		DiscoverPartitions(context.Context) ([]Partition, error)
	}); ok {
		partitions, err := partitionDiscovery.DiscoverPartitions(ctx)
		if err != nil {
			return fmt.Errorf("discover cluster partitions: %w", err)
		}
		for _, partition := range partitions {
			if err := s.inventory.RegisterPartition(ctx, partition); err != nil {
				return fmt.Errorf("register discovered partition %q: %w", partition.Name, err)
			}
		}
	}
	return nil
}

func (s *DiscoveryService) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		log.Printf("node discovery loop not started: interval must be positive")
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.SyncClusterNodes(ctx); err != nil {
					log.Printf("periodic node discovery failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
