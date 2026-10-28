package core

import "testing"

func TestValidateNetworkRequiresCanonicalCIDRAndTenant(t *testing.T) {
	policy := NewPolicy()
	tests := []struct {
		name    string
		network Network
		wantErr bool
	}{
		{
			name:    "valid IPv4",
			network: Network{Name: "app", CIDR: "10.40.0.0/16", TenantID: "tenant-1", ProjectID: "project-1"},
		},
		{
			name:    "valid IPv6",
			network: Network{Name: "app-v6", CIDR: "2001:db8:40::/48", TenantID: "tenant-1", ProjectID: "project-1"},
		},
		{
			name:    "missing tenant",
			network: Network{Name: "app", CIDR: "10.40.0.0/16", TenantID: "tenant-1"},
			wantErr: true,
		},
		{
			name:    "host bits set",
			network: Network{Name: "app", CIDR: "10.40.1.1/16", TenantID: "tenant-1", ProjectID: "project-1"},
			wantErr: true,
		},
		{
			name:    "invalid prefix",
			network: Network{Name: "app", CIDR: "not-a-cidr", TenantID: "tenant-1", ProjectID: "project-1"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.ValidateNetwork(test.network)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateNetwork() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestValidateSubnetInNetwork(t *testing.T) {
	policy := NewPolicy()
	network := Network{ID: "network-1", CIDR: "10.40.0.0/16"}
	tests := []struct {
		name    string
		network Network
		subnet  Subnet
		wantErr bool
	}{
		{
			name:    "contained IPv4 subnet",
			network: network,
			subnet:  Subnet{NetworkID: network.ID, CIDR: "10.40.1.0/24", GatewayIP: "10.40.1.1"},
		},
		{
			name:    "contained IPv6 subnet",
			network: Network{ID: "network-v6", CIDR: "2001:db8:40::/48"},
			subnet:  Subnet{NetworkID: "network-v6", CIDR: "2001:db8:40:1::/64"},
		},
		{
			name:    "subnet outside parent",
			network: network,
			subnet:  Subnet{NetworkID: network.ID, CIDR: "10.41.1.0/24"},
			wantErr: true,
		},
		{
			name:    "subnet broader than parent",
			network: network,
			subnet:  Subnet{NetworkID: network.ID, CIDR: "10.0.0.0/8"},
			wantErr: true,
		},
		{
			name:    "gateway outside subnet",
			network: network,
			subnet:  Subnet{NetworkID: network.ID, CIDR: "10.40.1.0/24", GatewayIP: "10.40.2.1"},
			wantErr: true,
		},
		{
			name:    "gateway broadcast address",
			network: network,
			subnet:  Subnet{NetworkID: network.ID, CIDR: "10.40.1.0/24", GatewayIP: "10.40.1.255"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.ValidateSubnetInNetwork(test.subnet, test.network)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateSubnetInNetwork() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestGenerateIDReturnsUUID(t *testing.T) {
	id := GenerateID("net")
	if len(id) != 36 {
		t.Fatalf("GenerateID() = %q, want UUID length 36", id)
	}
	for _, index := range []int{8, 13, 18, 23} {
		if id[index] != '-' {
			t.Fatalf("GenerateID() = %q, missing UUID separator at %d", id, index)
		}
	}
}

func TestCreateNetworkFailsClosedWithoutVerifiedBackend(t *testing.T) {
	controller := NewController(nil, nil)
	err := controller.CreateNetwork(Network{
		ID:        "86c6aa12-791b-40ef-96b8-3af8cc07607f",
		Name:      "app",
		CIDR:      "10.40.0.0/16",
		TenantID:  "12",
		ProjectID: "34",
	})
	if err != ErrBackendUnavailable {
		t.Fatalf("CreateNetwork() error = %v, want %v", err, ErrBackendUnavailable)
	}
}
