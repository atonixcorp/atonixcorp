package slurm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atonixcorp/cloud/hpc/nodes"
)

var gpuCountPattern = regexp.MustCompile(`(?i)gpu(?::[^:,()]+)?:(\d+)`)

func (c *SlurmClient) DiscoverNodes(ctx context.Context) ([]nodes.Node, error) {
	if err := c.CheckAvailability(ctx); err != nil {
		return nil, err
	}
	output, err := c.run(ctx, c.scontrolPath, "show", "nodes", "--oneliner")
	if err != nil {
		return nil, fmt.Errorf("discover Slurm nodes: %w", err)
	}
	result, err := ParseScontrolNodes(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *SlurmClient) DiscoverPartitions(ctx context.Context) ([]nodes.Partition, error) {
	if err := c.CheckAvailability(ctx); err != nil {
		return nil, err
	}
	output, err := c.run(ctx, c.scontrolPath, "show", "partitions", "--oneliner")
	if err != nil {
		return nil, fmt.Errorf("discover Slurm partitions: %w", err)
	}
	result, err := ParseScontrolPartitions(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ParseScontrolNodes(output string) ([]nodes.Node, error) {
	records := splitScontrolRecords(output, "NodeName")
	result := make([]nodes.Node, 0, len(records))
	now := time.Now().UTC()
	for _, fields := range records {
		name := fields["NodeName"]
		if name == "" {
			return nil, errors.New("Slurm node record has no NodeName")
		}
		hostname := fields["NodeHostName"]
		if hostname == "" || hostname == "(null)" {
			hostname = name
		}
		cpus := parseIntField(fields, "CPUEfctv")
		if cpus == 0 {
			cpus = parseIntField(fields, "CPUTot")
		}
		state := parseNodeState(fields["State"])
		role := nodes.NodeRoleCompute
		gpus := gpuCountPattern.FindStringSubmatch(fields["Gres"])
		gpuCount := 0
		if len(gpus) == 2 {
			gpuCount, _ = strconv.Atoi(gpus[1])
		}
		if gpuCount > 0 {
			role = nodes.NodeRoleGPU
		}
		partitions := splitCommaField(fields["Partitions"])
		labels := map[string]string{}
		for _, key := range []string{"Arch", "ActiveFeatures", "Gres", "State"} {
			if value := fields[key]; value != "" && value != "(null)" {
				labels[strings.ToLower(key)] = value
			}
		}
		result = append(result, nodes.Node{
			ID:          name,
			Hostname:    hostname,
			Address:     normalizedScontrolValue(fields["NodeAddr"]),
			Role:        role,
			State:       state,
			CPUs:        cpus,
			MemoryGB:    parseIntField(fields, "RealMemory") / 1024,
			GPUs:        gpuCount,
			Labels:      labels,
			Partitions:  partitions,
			LastSeenAt:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
			HealthScore: 0,
		})
	}
	return result, nil
}

func ParseScontrolPartitions(output string) ([]nodes.Partition, error) {
	records := splitScontrolRecords(output, "PartitionName")
	result := make([]nodes.Partition, 0, len(records))
	now := time.Now().UTC()
	for _, fields := range records {
		name := fields["PartitionName"]
		if name == "" {
			return nil, errors.New("Slurm partition record has no PartitionName")
		}
		result = append(result, nodes.Partition{
			Name:        name,
			Description: normalizedScontrolValue(fields["MaxTime"]),
			Nodes:       splitCommaField(fields["Nodes"]),
			Labels:      map[string]string{"state": normalizedScontrolValue(fields["State"])},
			MaxCPUs:     parseIntField(fields, "MaxCPUsPerNode"),
			Active:      strings.EqualFold(fields["State"], "UP"),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return result, nil
}

func splitScontrolRecords(output, key string) []map[string]string {
	var records []map[string]string
	var current map[string]string
	for _, field := range strings.Fields(output) {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		if name == key {
			if current != nil {
				records = append(records, current)
			}
			current = make(map[string]string)
		}
		if current != nil {
			current[name] = value
		}
	}
	if current != nil {
		records = append(records, current)
	}
	return records
}

func parseIntField(fields map[string]string, name string) int {
	value, _ := strconv.Atoi(fields[name])
	return value
}

func splitCommaField(value string) []string {
	value = normalizedScontrolValue(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func normalizedScontrolValue(value string) string {
	if value == "(null)" {
		return ""
	}
	return value
}

func parseNodeState(value string) nodes.NodeState {
	value = strings.ToUpper(value)
	switch {
	case strings.Contains(value, "DOWN"), strings.Contains(value, "FAIL"),
		strings.Contains(value, "NO_RESPOND"), strings.Contains(value, "NOT_RESPOND"):
		return nodes.NodeStateOffline
	case strings.Contains(value, "DRAIN"):
		return nodes.NodeStateDraining
	case strings.Contains(value, "IDLE"), strings.Contains(value, "ALLOCATED"),
		strings.Contains(value, "MIXED"), strings.Contains(value, "COMPLETING"):
		return nodes.NodeStateOnline
	default:
		return nodes.NodeStateUnknown
	}
}
