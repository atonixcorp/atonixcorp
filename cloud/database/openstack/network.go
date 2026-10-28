package openstack

import (
	"encoding/json"
	"fmt"
)

type NetworkClient struct {
	os *Client
}

func NewNetworkClient(os *Client) *NetworkClient {
	return &NetworkClient{os}
}

func (n *NetworkClient) CreatePort(networkID, sgID string) (string, error) {
	if networkID == "" || n.os.NetworkEndpoint == "" {
		return "", fmt.Errorf("Neutron endpoint and network ID are required")
	}
	payload := map[string]interface{}{
		"port": map[string]interface{}{
			"network_id": networkID,
		},
	}
	if sgID != "" {
		payload["port"].(map[string]interface{})["security_groups"] = []string{sgID}
	}

	data, err := n.os.do("POST", n.os.NetworkEndpoint+"/ports", payload)
	if err != nil {
		return "", err
	}

	var resp struct {
		Port struct {
			ID string `json:"id"`
		} `json:"port"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("decode Neutron port response: %w", err)
	}
	if resp.Port.ID == "" {
		return "", fmt.Errorf("Neutron response omitted port ID")
	}
	return resp.Port.ID, nil
}
