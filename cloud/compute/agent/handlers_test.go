package agent

import (
	"errors"
	"testing"

	"atonixcorp/cloud/compute/hypervisor"
	"atonixcorp/cloud/compute/rpc"
)

type fakeHypervisor struct {
	action string
	vmID   string
	err    error
}

func (h *fakeHypervisor) StartVM(id string) error {
	h.action, h.vmID = "start", id
	return h.err
}

func (h *fakeHypervisor) StopVM(id string) error {
	h.action, h.vmID = "stop", id
	return h.err
}

func (h *fakeHypervisor) PauseVM(id string) error {
	h.action, h.vmID = "pause", id
	return h.err
}

func (h *fakeHypervisor) RebootVM(id string) error {
	h.action, h.vmID = "reboot", id
	return h.err
}

func (*fakeHypervisor) GetState(string) (hypervisor.VMState, error) {
	return hypervisor.VMStopped, nil
}

func TestHandleMessageExecutesLifecycleActionsAndSurfacesErrors(t *testing.T) {
	driver := &fakeHypervisor{}
	agent := &Agent{HV: driver}

	if err := agent.HandleMessage(rpc.RPCMessage{Type: "vm.pause", Payload: "instance-01"}); err != nil {
		t.Fatal(err)
	}
	if driver.action != "pause" || driver.vmID != "instance-01" {
		t.Fatalf("pause was not dispatched to the hypervisor: %+v", driver)
	}

	driver.err = errors.New("libvirt unavailable")
	if err := agent.HandleMessage(rpc.RPCMessage{Type: "vm.start", Payload: "instance-01"}); err == nil {
		t.Fatal("expected hypervisor error to be returned")
	}
}

func TestHandleMessageRejectsMalformedAndUnknownMessages(t *testing.T) {
	agent := &Agent{HV: &fakeHypervisor{}}
	tests := []rpc.RPCMessage{
		{Type: "vm.start", Payload: 42},
		{Type: "vm.unknown", Payload: "instance-01"},
	}
	for _, message := range tests {
		if err := agent.HandleMessage(message); err == nil {
			t.Errorf("expected message %q to return an error", message.Type)
		}
	}
}
