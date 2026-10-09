package atcloud

import (
	"atnetwork/core"
	"fmt"
	"os"
)

type DnsmasqBackend struct {
	ConfigDir string
}

func NewDnsmasqBackend(dir string) *DnsmasqBackend {
	return &DnsmasqBackend{ConfigDir: dir}
}

func (b *DnsmasqBackend) ApplySubnet(s core.Subnet) error {
	fmt.Println("ATN → dnsmasq: configure DHCP for", s.CIDR)

	cfg := fmt.Sprintf("dhcp-range=%s,static", s.CIDR)
	path := fmt.Sprintf("%s/%s.conf", b.ConfigDir, s.ID)

	return os.WriteFile(path, []byte(cfg), 0644)
}

func (b *DnsmasqBackend) ApplyNetwork(core.Network) error { return nil }

func (b *DnsmasqBackend) ApplyRouter(core.Router) error { return nil }

func (b *DnsmasqBackend) ApplyPort(core.Port) error { return nil }

func (b *DnsmasqBackend) ApplySecurityGroup(core.SecurityGroup) error { return nil }

func (b *DnsmasqBackend) ListNetworks() ([]core.Network, error) { return nil, nil }

func (b *DnsmasqBackend) ListSubnets() ([]core.Subnet, error) { return nil, nil }

func (b *DnsmasqBackend) ListRouters() ([]core.Router, error) { return nil, nil }

func (b *DnsmasqBackend) ListPorts() ([]core.Port, error) { return nil, nil }

func (b *DnsmasqBackend) ListSecurityGroups() ([]core.SecurityGroup, error) { return nil, nil }
