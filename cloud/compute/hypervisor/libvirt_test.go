package hypervisor

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	output string
	err    error
	name   string
	args   []string
}

func (r *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.name = name
	r.args = append([]string(nil), args...)
	return []byte(r.output), r.err
}

func TestLibvirtLifecycleCommands(t *testing.T) {
	tests := []struct {
		name   string
		call   func(*LibvirtDriver) error
		action string
	}{
		{name: "start", call: func(driver *LibvirtDriver) error { return driver.StartVM("instance-01") }, action: "start"},
		{name: "stop", call: func(driver *LibvirtDriver) error { return driver.StopVM("instance-01") }, action: "shutdown"},
		{name: "pause", call: func(driver *LibvirtDriver) error { return driver.PauseVM("instance-01") }, action: "suspend"},
		{name: "reboot", call: func(driver *LibvirtDriver) error { return driver.RebootVM("instance-01") }, action: "reboot"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeCommandRunner{}
			driver := &LibvirtDriver{uri: "qemu:///session", runner: runner}
			if err := test.call(driver); err != nil {
				t.Fatal(err)
			}
			expectedArgs := []string{"--connect", "qemu:///session", test.action, "instance-01"}
			if runner.name != "virsh" || !reflect.DeepEqual(runner.args, expectedArgs) {
				t.Fatalf("unexpected virsh invocation: %s %v", runner.name, runner.args)
			}
		})
	}
}

func TestLibvirtGetState(t *testing.T) {
	tests := []struct {
		output string
		want   VMState
	}{
		{output: "running\n", want: VMRunning},
		{output: "in shutdown\n", want: VMStopping},
		{output: "shut off\n", want: VMStopped},
		{output: "paused\n", want: VMPaused},
	}

	for _, test := range tests {
		runner := &fakeCommandRunner{output: test.output}
		driver := &LibvirtDriver{uri: "qemu:///system", runner: runner}
		got, err := driver.GetState("instance-01")
		if err != nil {
			t.Fatalf("GetState(%q): %v", test.output, err)
		}
		if got != test.want {
			t.Errorf("GetState(%q) = %q, want %q", test.output, got, test.want)
		}
	}
}

func TestLibvirtRejectsInvalidDomainNamesAndSurfacesCommandErrors(t *testing.T) {
	runner := &fakeCommandRunner{}
	driver := &LibvirtDriver{uri: "qemu:///system", runner: runner}
	if err := driver.StartVM("--help"); err == nil {
		t.Fatal("expected invalid domain name to be rejected")
	}
	if runner.name != "" {
		t.Fatal("invalid domain name invoked virsh")
	}

	runner.err = errors.New("exit status 1")
	runner.output = "failed to get domain"
	err := driver.StartVM("instance-01")
	if err == nil || !strings.Contains(err.Error(), "failed to get domain") {
		t.Fatalf("expected virsh failure details, got %v", err)
	}
}
