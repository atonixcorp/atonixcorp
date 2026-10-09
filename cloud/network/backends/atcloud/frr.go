package atcloud

import (
	"atnetwork/core"
	"fmt"
	"os"
	"os/exec"
)

type FRRBackend struct {
	ConfigPath string
}

func NewFRRBackend(path string) *FRRBackend {
	return &FRRBackend{ConfigPath: path}
}

func (b *FRRBackend) ApplyRouter(r core.Router) error {
	fmt.Println("ATN → FRR: configure router", r.Name)

	cfg := fmt.Sprintf(`
router bgp 65001
  neighbor 192.0.2.1 remote-as 65000
  address-family ipv4 unicast
    network 203.0.113.0/24
`)

	if err := os.WriteFile(b.ConfigPath, []byte(cfg), 0644); err != nil {
		return err
	}

	return exec.Command("systemctl", "reload", "frr").Run()
}

func (b *FRRBackend) ApplyNetwork(core.Network) error { return nil }

func (b *FRRBackend) ApplySubnet(core.Subnet) error { return nil }

func (b *FRRBackend) ApplyPort(core.Port) error { return nil }

func (b *FRRBackend) ApplySecurityGroup(core.SecurityGroup) error { return nil }

func (b *FRRBackend) ListNetworks() ([]core.Network, error) { return nil, nil }

func (b *FRRBackend) ListSubnets() ([]core.Subnet, error) { return nil, nil }

func (b *FRRBackend) ListRouters() ([]core.Router, error) { return nil, nil }

func (b *FRRBackend) ListPorts() ([]core.Port, error) { return nil, nil }

func (b *FRRBackend) ListSecurityGroups() ([]core.SecurityGroup, error) { return nil, nil }

func (b *FRRBackend) renderConfig() error {
	// TODO: query store for all FloatingIPs
	// Example:
	//   network 203.0.113.10/32
	cfg := `
router bgp 65001
  neighbor 192.0.2.1 remote-as 65000
  address-family ipv4 unicast
    network 203.0.113.10/32
`
	if err := os.WriteFile(b.ConfigPath, []byte(cfg), 0644); err != nil {
		return err
	}
	return exec.Command("systemctl", "reload", "frr").Run()
}
