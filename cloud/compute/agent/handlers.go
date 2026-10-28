package agent

import (
	"fmt"

	"atonixcorp/cloud/compute/hypervisor"
	"atonixcorp/cloud/compute/rpc"
)

func (a *Agent) HandleMessage(msg rpc.RPCMessage) error {
	if a.HV == nil {
		return fmt.Errorf("hypervisor is not configured")
	}

	switch msg.Type {
	case "vm.start":
		vmID, err := stringPayload(msg.Payload)
		if err != nil {
			return err
		}
		return a.HV.StartVM(vmID)

	case "vm.pause":
		vmID, err := stringPayload(msg.Payload)
		if err != nil {
			return err
		}
		return a.HV.PauseVM(vmID)

	case "vm.stop":
		vmID, err := stringPayload(msg.Payload)
		if err != nil {
			return err
		}
		return a.HV.StopVM(vmID)

	case "vm.reboot":
		vmID, err := stringPayload(msg.Payload)
		if err != nil {
			return err
		}
		return a.HV.RebootVM(vmID)

	case "vm.pin":
		pin, ok := msg.Payload.(hypervisor.CPUPinning)
		if !ok {
			return fmt.Errorf("vm.pin payload must be hypervisor.CPUPinning")
		}
		return hypervisor.PinCPU(pin.VMID, pin)

	case "vm.numa":
		numa, ok := msg.Payload.(hypervisor.NUMANode)
		if !ok {
			return fmt.Errorf("vm.numa payload must be hypervisor.NUMANode")
		}
		if numa.VMID == "" {
			return fmt.Errorf("vm.numa payload must include a VMID")
		}
		return hypervisor.PlaceNUMA(numa.VMID, numa)

	default:
		return fmt.Errorf("unknown compute agent message type %q", msg.Type)
	}
}

func stringPayload(payload any) (string, error) {
	value, ok := payload.(string)
	if !ok || value == "" {
		return "", fmt.Errorf("VM action payload must be a non-empty VM ID string")
	}
	return value, nil
}
