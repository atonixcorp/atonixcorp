package openstack

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type ComputeClient struct {
	os *Client
}

func NewComputeClient(os *Client) *ComputeClient {
	return &ComputeClient{os}
}

func (c *ComputeClient) CreateServer(
	name, imageID, flavorID, networkID, sgID string,
) (string, error) {
	if name == "" || imageID == "" || flavorID == "" || networkID == "" {
		return "", fmt.Errorf("server name, image, flavor, and private network are required")
	}
	server := map[string]interface{}{
		"name":      name,
		"flavorRef": flavorID,
		"networks":  []map[string]string{{"uuid": networkID}},
		"imageRef":  imageID,
	}
	if sgID != "" {
		server["security_groups"] = []map[string]string{{"name": sgID}}
	}
	data, err := c.os.do("POST", c.os.ComputeEndpoint+"/servers", map[string]interface{}{
		"server": server,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Server struct {
			ID string `json:"id"`
		} `json:"server"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("decode Nova create-server response: %w", err)
	}
	if resp.Server.ID == "" {
		return "", fmt.Errorf("Nova create-server response omitted server ID")
	}
	return resp.Server.ID, nil
}

func (c *ComputeClient) FindServerByName(name string) (string, bool, error) {
	if name == "" {
		return "", false, fmt.Errorf("server name is required")
	}
	data, err := c.os.do(
		"GET", c.os.ComputeEndpoint+"/servers/detail?"+url.Values{"name": {name}}.Encode(), nil,
	)
	if err != nil {
		return "", false, err
	}
	var response struct {
		Servers []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"servers"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", false, fmt.Errorf("decode Nova server list response: %w", err)
	}
	for _, server := range response.Servers {
		if server.Name == name && server.ID != "" {
			return server.ID, true, nil
		}
	}
	return "", false, nil
}

func (c *ComputeClient) GetServerState(serverID string) (string, string, error) {
	if serverID == "" {
		return "", "", fmt.Errorf("server ID is required")
	}
	data, err := c.os.do("GET", c.os.ComputeEndpoint+"/servers/"+serverID, nil)
	if err != nil {
		return "", "", err
	}

	var resp struct {
		Server struct {
			Status    string `json:"status"`
			Addresses map[string][]struct {
				Addr string `json:"addr"`
			} `json:"addresses"`
			VolumesAttached []struct {
				ID string `json:"id"`
			} `json:"volumes_attached"`
		} `json:"server"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", "", fmt.Errorf("decode Nova server response: %w", err)
	}
	var ip string
	for _, networks := range resp.Server.Addresses {
		for _, address := range networks {
			if address.Addr != "" {
				ip = address.Addr
				break
			}
		}
		if ip != "" {
			break
		}
	}
	return resp.Server.Status, ip, nil
}

func (c *ComputeClient) IsVolumeAttached(serverID, volumeID string) (bool, error) {
	if serverID == "" || volumeID == "" {
		return false, fmt.Errorf("server and volume IDs are required")
	}
	data, err := c.os.do("GET", c.os.ComputeEndpoint+"/servers/"+serverID, nil)
	if err != nil {
		return false, err
	}
	var resp struct {
		Server struct {
			VolumesAttached []struct {
				ID string `json:"id"`
			} `json:"os-extended-volumes:volumes_attached"`
		} `json:"server"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return false, fmt.Errorf("decode Nova volume-attachment response: %w", err)
	}
	for _, attached := range resp.Server.VolumesAttached {
		if attached.ID == volumeID {
			return true, nil
		}
	}
	return false, nil
}

func (c *ComputeClient) GetServerIP(serverID string) (string, error) {
	status, ip, err := c.GetServerState(serverID)
	if err != nil {
		return "", err
	}
	if status != "ACTIVE" {
		return "", fmt.Errorf("Nova server %s is %s, not ACTIVE", serverID, status)
	}
	if ip == "" {
		return "", fmt.Errorf("no IP found for Nova server %s", serverID)
	}
	return ip, nil
}

func (c *ComputeClient) DeleteServer(serverID string) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}
	_, err := c.os.do("DELETE", c.os.ComputeEndpoint+"/servers/"+serverID, nil)
	return err
}
