package atcloud

import (
	"atnetwork/core"
	"fmt"
)

type ATCloudBackend struct{}

func NewATCloudBackend() *ATCloudBackend {
	return &ATCloudBackend{}
}

func (b *ATCloudBackend) ApplyNetwork(n core.Network) error {
	fmt.Println("ATN → ATCloud: create network", n.Name)
	// TODO: call your cloud API
	return nil
}

func (b *ATCloudBackend) ApplySubnet(s core.Subnet) error {
	fmt.Println("ATN → ATCloud: create subnet", s.CIDR)
	return nil
}

func (b *ATCloudBackend) ApplyRouter(r core.Router) error {
	fmt.Println("ATN → ATCloud: create router", r.Name)
	return nil
}

func (b *ATCloudBackend) ApplyPort(p core.Port) error {
	fmt.Println("ATN → ATCloud: create port", p.ID)
	return nil
}

func (b *ATCloudBackend) ApplySecurityGroup(sg core.SecurityGroup) error {
	fmt.Println("ATN → ATCloud: create SG", sg.Name)
	return nil
}
