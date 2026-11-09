package slurm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ClientConfig struct {
	SbatchPath   string
	SqueuePath   string
	SacctPath    string
	ScancelPath  string
	ScontrolPath string
	Timeout      time.Duration
}

type SlurmClient struct {
	sbatchPath   string
	squeuePath   string
	sacctPath    string
	scancelPath  string
	scontrolPath string
	timeout      time.Duration
}

func NewSlurmClient() *SlurmClient {
	client, _ := NewSlurmClientWithConfig(ClientConfig{
		SbatchPath:   "sbatch",
		SqueuePath:   "squeue",
		SacctPath:    "sacct",
		ScancelPath:  "scancel",
		ScontrolPath: "scontrol",
		Timeout:      30 * time.Second,
	})
	return client
}

func NewSlurmClientWithConfig(config ClientConfig) (*SlurmClient, error) {
	if config.Timeout <= 0 {
		return nil, errors.New("Slurm command timeout must be positive")
	}
	for name, path := range map[string]string{
		"sbatch": config.SbatchPath, "squeue": config.SqueuePath,
		"sacct": config.SacctPath, "scancel": config.ScancelPath,
		"scontrol": config.ScontrolPath,
	} {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("Slurm %s path is required", name)
		}
	}
	return &SlurmClient{
		sbatchPath:   config.SbatchPath,
		squeuePath:   config.SqueuePath,
		sacctPath:    config.SacctPath,
		scancelPath:  config.ScancelPath,
		scontrolPath: config.ScontrolPath,
		timeout:      config.Timeout,
	}, nil
}

func (c *SlurmClient) CheckAvailability(ctx context.Context) error {
	if c == nil {
		return errors.New("Slurm client is not configured")
	}
	output, err := c.run(ctx, c.scontrolPath, "ping")
	if err != nil {
		return fmt.Errorf("check Slurm controller availability: %w", err)
	}
	if !strings.Contains(strings.ToLower(output), "is up") {
		return fmt.Errorf("Slurm controller did not report an available state: %q", strings.TrimSpace(output))
	}
	return nil
}

func (c *SlurmClient) run(ctx context.Context, name string, args ...string) (string, error) {
	if c == nil {
		return "", errors.New("Slurm client is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &CommandError{
			Cmd:    name,
			Args:   args,
			Stderr: stderr.String(),
			Err:    fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String())),
		}
	}

	return stdout.String(), nil
}

type CommandError struct {
	Cmd    string
	Args   []string
	Stderr string
	Err    error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("Slurm command %q failed: %v", e.Cmd, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}
