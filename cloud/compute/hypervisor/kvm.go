package hypervisor

type KVM struct {
	libvirt *LibvirtDriver
}

func NewKVM() *KVM {
	return &KVM{libvirt: NewLibvirtDriver()}
}

func (k *KVM) StartVM(id string) error {
	return k.libvirt.StartVM(id)
}

func (k *KVM) StopVM(id string) error {
	return k.libvirt.StopVM(id)
}

func (k *KVM) PauseVM(id string) error {
	return k.libvirt.PauseVM(id)
}

func (k *KVM) RebootVM(id string) error {
	return k.libvirt.RebootVM(id)
}

func (k *KVM) GetState(id string) (VMState, error) {
	return k.libvirt.GetState(id)
}

var _ Hypervisor = (*KVM)(nil)
