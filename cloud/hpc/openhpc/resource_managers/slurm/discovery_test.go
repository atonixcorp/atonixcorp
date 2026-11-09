package slurm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atonixcorp/cloud/hpc/nodes"
)

func TestParseScontrolNodes(t *testing.T) {
	raw := "NodeName=node01 Arch=x86_64 CPUTot=32 CPUEfctv=32 RealMemory=65536 Gres=gpu:a100:4 NodeAddr=10.0.0.1 NodeHostName=node01 State=IDLE Partitions=gpu,compute " +
		"NodeName=node02 Arch=x86_64 CPUTot=16 RealMemory=32768 Gres=(null) NodeAddr=(null) NodeHostName=node02 State=DOWN Partitions=compute"

	result, err := ParseScontrolNodes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("discovered %d nodes, want 2", len(result))
	}
	first := result[0]
	if first.ID != "node01" || first.Address != "10.0.0.1" || first.CPUs != 32 ||
		first.MemoryGB != 64 || first.GPUs != 4 || first.Role != nodes.NodeRoleGPU ||
		first.State != nodes.NodeStateOnline || len(first.Partitions) != 2 {
		t.Fatalf("unexpected first node: %+v", first)
	}
	if result[1].Address != "" || result[1].State != nodes.NodeStateOffline || result[1].GPUs != 0 {
		t.Fatalf("unexpected second node: %+v", result[1])
	}
}

func TestParseScontrolPartitions(t *testing.T) {
	partitions, err := ParseScontrolPartitions(
		"PartitionName=compute Nodes=node01,node02 State=UP MaxCPUsPerNode=32 MaxTime=INFINITE",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(partitions) != 1 || partitions[0].Name != "compute" ||
		!partitions[0].Active || partitions[0].MaxCPUs != 32 ||
		len(partitions[0].Nodes) != 2 {
		t.Fatalf("unexpected partitions: %+v", partitions)
	}
}

func TestCheckAvailabilityUsesConfiguredScontrol(t *testing.T) {
	command := filepath.Join(t.TempDir(), "scontrol")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf '%s\\n' 'Slurmctld(primary) at test is UP'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	client, err := NewSlurmClientWithConfig(testSlurmConfig(command))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckAvailability(context.Background()); err != nil {
		t.Fatalf("configured controller should be available: %v", err)
	}
}

func TestCheckAvailabilityReportsCommandFailure(t *testing.T) {
	command := filepath.Join(t.TempDir(), "scontrol")
	if err := os.WriteFile(command, []byte("#!/bin/sh\necho controller-down >&2\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	client, err := NewSlurmClientWithConfig(testSlurmConfig(command))
	if err != nil {
		t.Fatal(err)
	}
	err = client.CheckAvailability(context.Background())
	if err == nil || !strings.Contains(err.Error(), "controller-down") {
		t.Fatalf("expected useful Slurm availability error, got %v", err)
	}
}

func TestSlurmClientRejectsMissingCommandPath(t *testing.T) {
	config := testSlurmConfig("/usr/bin/scontrol")
	config.ScontrolPath = " "
	if _, err := NewSlurmClientWithConfig(config); err == nil {
		t.Fatal("empty scontrol path accepted")
	}
}

func testSlurmConfig(scontrol string) ClientConfig {
	return ClientConfig{
		SbatchPath:   "/usr/bin/sbatch",
		SqueuePath:   "/usr/bin/squeue",
		SacctPath:    "/usr/bin/sacct",
		ScancelPath:  "/usr/bin/scancel",
		ScontrolPath: scontrol,
		Timeout:      2 * time.Second,
	}
}
