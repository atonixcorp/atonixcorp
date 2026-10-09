package atcloud

import (
	"atnetwork/core"
	"fmt"
	"os/exec"
)

type OVNBackend struct{}

func NewOVNBackend() *OVNBackend {
	return &OVNBackend{}
}

func (b *OVNBackend) ApplyNetwork(n core.Network) error {
	fmt.Println("ATN → OVN: create logical switch", n.ID)
	return exec.Command("ovn-nbctl", "ls-add", n.ID).Run()
}

func (b *OVNBackend) ApplySubnet(s core.Subnet) error {
	fmt.Println("ATN → OVN: subnet", s.CIDR)
	return nil
}

func (b *OVNBackend) ApplyRouter(r core.Router) error {
	fmt.Println("ATN → OVN: lr-add", r.ID)
	return exec.Command("ovn-nbctl", "lr-add", r.ID).Run()
}

func (b *OVNBackend) ApplyPort(p core.Port) error {
	fmt.Println("ATN → OVN: lsp-add", p.ID)

	if err := exec.Command("ovn-nbctl", "lsp-add", p.NetworkID, p.ID).Run(); err != nil {
		return err
	}

	if p.MAC != "" {
		_ = exec.Command("ovn-nbctl", "lsp-set-addresses", p.ID, p.MAC).Run()
	}

	return nil
}

func (b *OVNBackend) ApplySecurityGroup(sg core.SecurityGroup) error {
	fmt.Println("ATN → OVN: SG placeholder", sg.Name)
	return nil
}

func (b *OVNBackend) AttachInterface(router core.Router, subnet core.Subnet, iface core.RouterInterface) error {
	fmt.Println("ATN → OVN: attach router interface", iface.ID)

	lrpName := iface.ID
	cidr := fmt.Sprintf("%s/%d", iface.IPAddress, 24)

	if err := exec.Command("ovn-nbctl", "lrp-add", router.ID, lrpName, "fa:16:3e:00:00:01", cidr).Run(); err != nil {
		return err
	}

	lspName := fmt.Sprintf("%s-lsp", iface.ID)

	if err := exec.Command("ovn-nbctl", "lsp-add", subnet.NetworkID, lspName).Run(); err != nil {
		return err
	}
	if err := exec.Command("ovn-nbctl", "lsp-set-type", lspName, "router").Run(); err != nil {
		return err
	}
	if err := exec.Command("ovn-nbctl", "lsp-set-options", lspName, "router-port="+lrpName).Run(); err != nil {
		return err
	}

	return nil
}

func (b *OVNBackend) ApplyNATRule(nat core.NATRule) error {
	fmt.Println("ATN → OVN: NAT", nat.Type, nat.ExternalIP, "→", nat.InternalIP)

	if nat.Type == "dnat" {
		return exec.Command("ovn-nbctl", "lr-nat-add",
			nat.RouterID, "dnat", nat.ExternalIP, nat.InternalIP).Run()
	}

	if nat.Type == "snat" {
		return exec.Command("ovn-nbctl", "lr-nat-add",
			nat.RouterID, "snat", nat.ExternalIP, nat.InternalIP).Run()
	}

	return nil
}

func (b *OVNBackend) ListNetworks() ([]core.Network, error) { return nil, nil }

func (b *OVNBackend) ListSubnets() ([]core.Subnet, error) { return nil, nil }

func (b *OVNBackend) ListRouters() ([]core.Router, error) { return nil, nil }

func (b *OVNBackend) ListPorts() ([]core.Port, error) { return nil, nil }

func (b *OVNBackend) ListSecurityGroups() ([]core.SecurityGroup, error) { return nil, nil }
