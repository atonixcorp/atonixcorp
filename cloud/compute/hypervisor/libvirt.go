package hypervisor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var domainNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type LibvirtDriver struct {
	uri    string
	runner commandRunner
}

func NewLibvirtDriver() *LibvirtDriver {
	uri := os.Getenv("LIBVIRT_URI")
	if uri == "" {
		uri = "qemu:///system"
	}
	return &LibvirtDriver{uri: uri, runner: execCommandRunner{}}
}

func (l *LibvirtDriver) StartVM(id string) error {
	return l.runDomainCommand("start", id)
}

func (l *LibvirtDriver) StopVM(id string) error {
	return l.runDomainCommand("shutdown", id)
}

func (l *LibvirtDriver) PauseVM(id string) error {
	return l.runDomainCommand("suspend", id)
}

func (l *LibvirtDriver) RebootVM(id string) error {
	return l.runDomainCommand("reboot", id)
}

func (l *LibvirtDriver) GetState(id string) (VMState, error) {
	if err := validateDomainName(id); err != nil {
		return "", err
	}
	output, err := l.command("domstate", "--domain", id)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(string(output))) {
	case "running":
		return VMRunning, nil
	case "in shutdown":
		return VMStopping, nil
	case "paused", "pmsuspended":
		return VMPaused, nil
	case "shut off", "shutoff":
		return VMStopped, nil
	default:
		return "", fmt.Errorf("libvirt returned an unsupported state for domain %q: %q", id, strings.TrimSpace(string(output)))
	}
}

func (l *LibvirtDriver) runDomainCommand(action, id string) error {
	if err := validateDomainName(id); err != nil {
		return err
	}
	_, err := l.command(action, id)
	return err
}

func (l *LibvirtDriver) command(args ...string) ([]byte, error) {
	if l.runner == nil {
		return nil, errors.New("libvirt command runner is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	commandArgs := append([]string{"--connect", l.uri}, args...)
	output, err := l.runner.Run(ctx, "virsh", commandArgs...)
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			return nil, fmt.Errorf("virsh %s failed: %w", strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("virsh %s failed: %s: %w", strings.Join(args, " "), details, err)
	}
	return output, nil
}

func validateDomainName(id string) error {
	if !domainNamePattern.MatchString(id) {
		return fmt.Errorf("invalid libvirt domain name %q", id)
	}
	return nil
}

var _ Hypervisor = (*LibvirtDriver)(nil)
var _ interface{ PauseVM(string) error } = (*LibvirtDriver)(nil)
