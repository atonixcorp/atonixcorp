package openstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"atnetwork/core"
)

type NeutronClient struct {
	Endpoint string
	Token    string
}

func NewNeutronClient(endpoint, token string) *NeutronClient {
	return &NeutronClient{
		Endpoint: endpoint,
		Token:    token,
	}
}

type NeutronBackend struct {
	client *NeutronClient
}

func NewNeutronBackend(client *NeutronClient) *NeutronBackend {
	return &NeutronBackend{client: client}
}

// -----------------------------
// NETWORK
// -----------------------------

func (b *NeutronBackend) ApplyNetwork(n core.Network) error {
	fmt.Printf("[NEUTRON] ApplyNetwork: ID=%s Name=%s CIDR=%s Tenant=%s External=%v\n",
		n.ID, n.Name, n.CIDR, n.TenantID, n.External)

	payload := map[string]any{
		"network": map[string]any{
			"name":           n.Name,
			"tenant_id":      n.TenantID,
			"admin_state_up": true,
		},
	}

	return b.post("/v2.0/networks", payload)
}

// -----------------------------
// SUBNET
// -----------------------------

func (b *NeutronBackend) ApplySubnet(s core.Subnet) error {
	fmt.Printf("[NEUTRON] ApplySubnet: ID=%s NetworkID=%s CIDR=%s Gateway=%s\n",
		s.ID, s.NetworkID, s.CIDR, s.GatewayIP)

	payload := map[string]any{
		"subnet": map[string]any{
			"network_id": s.NetworkID,
			"cidr":       s.CIDR,
			"gateway_ip": s.GatewayIP,
			"ip_version": 4,
		},
	}

	return b.post("/v2.0/subnets", payload)
}

// -----------------------------
// ROUTER
// -----------------------------

func (b *NeutronBackend) ApplyRouter(r core.Router) error {
	fmt.Printf("[NEUTRON] ApplyRouter: ID=%s Name=%s ExternalNetworkID=%s Tenant=%s\n",
		r.ID, r.Name, r.ExternalNetworkID, r.TenantID)

	payload := map[string]any{
		"router": map[string]any{
			"name":      r.Name,
			"tenant_id": r.TenantID,
		},
	}

	if r.ExternalNetworkID != "" {
		payload["router"].(map[string]any)["external_gateway_info"] = map[string]any{
			"network_id": r.ExternalNetworkID,
		}
	}

	return b.post("/v2.0/routers", payload)
}

// -----------------------------
// PORT
// -----------------------------

func (b *NeutronBackend) ApplyPort(p core.Port) error {
	fmt.Printf("[NEUTRON] ApplyPort: ID=%s NetworkID=%s DeviceID=%s MAC=%s\n",
		p.ID, p.NetworkID, p.DeviceID, p.MAC)

	payload := map[string]any{
		"port": map[string]any{
			"network_id":     p.NetworkID,
			"device_id":      p.DeviceID,
			"mac_address":    p.MAC,
			"admin_state_up": true,
		},
	}

	return b.post("/v2.0/ports", payload)
}

// -----------------------------
// SECURITY GROUP
// -----------------------------

func (b *NeutronBackend) ApplySecurityGroup(sg core.SecurityGroup) error {
	fmt.Printf("[NEUTRON] ApplySecurityGroup: ID=%s Name=%s Tenant=%s\n",
		sg.ID, sg.Name, sg.TenantID)

	payload := map[string]any{
		"security_group": map[string]any{
			"name":      sg.Name,
			"tenant_id": sg.TenantID,
		},
	}

	if err := b.post("/v2.0/security-groups", payload); err != nil {
		return err
	}

	// Apply rules
	for _, rule := range sg.Rules {
		fmt.Printf("[NEUTRON]   Rule: %s %s %s %s\n",
			rule.Direction, rule.Protocol, rule.PortRange, rule.CIDR)

		rulePayload := map[string]any{
			"security_group_rule": map[string]any{
				"security_group_id": sg.ID,
				"direction":         rule.Direction,
				"protocol":          rule.Protocol,
				"remote_ip_prefix":  rule.CIDR,
			},
		}

		if rule.PortRange != "" {
			rulePayload["security_group_rule"].(map[string]any)["port_range_min"] = rule.PortRange
			rulePayload["security_group_rule"].(map[string]any)["port_range_max"] = rule.PortRange
		}

		if err := b.post("/v2.0/security-group-rules", rulePayload); err != nil {
			return err
		}
	}

	return nil
}

// -----------------------------
// INTERNAL HTTP CLIENT
// -----------------------------

func (b *NeutronBackend) post(path string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	url := b.client.Endpoint + path

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("X-Auth-Token", b.client.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("neutron request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("neutron error: %s", resp.Status)
	}

	return nil
}

// -----------------------------
// OPTIONAL LIST METHODS (for reconciliation)
// -----------------------------

func (b *NeutronBackend) ListNetworks() ([]core.Network, error) {
	return nil, nil
}

func (b *NeutronBackend) ListSubnets() ([]core.Subnet, error) {
	return nil, nil
}

func (b *NeutronBackend) ListRouters() ([]core.Router, error) {
	return nil, nil
}

func (b *NeutronBackend) ListPorts() ([]core.Port, error) {
	return nil, nil
}

func (b *NeutronBackend) ListSecurityGroups() ([]core.SecurityGroup, error) {
	return nil, nil
}
