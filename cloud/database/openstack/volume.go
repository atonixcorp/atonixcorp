package openstack

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type VolumeClient struct {
	os *Client
}

func NewVolumeClient(os *Client) *VolumeClient {
	return &VolumeClient{os}
}

func (v *VolumeClient) CreateVolume(name string, sizeGB int) (string, error) {
	if name == "" || sizeGB < 1 {
		return "", fmt.Errorf("volume name and positive capacity are required")
	}
	payload := map[string]interface{}{
		"volume": map[string]interface{}{
			"name": name,
			"size": sizeGB,
		},
	}

	data, err := v.os.do(
		"POST", v.os.VolumeEndpoint+"/"+v.os.ProjectID+"/volumes", payload,
	)
	if err != nil {
		return "", err
	}

	var resp struct {
		Volume struct {
			ID string `json:"id"`
		} `json:"volume"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("decode Cinder create-volume response: %w", err)
	}
	if resp.Volume.ID == "" {
		return "", fmt.Errorf("Cinder create-volume response omitted volume ID")
	}
	return resp.Volume.ID, nil
}

func (v *VolumeClient) FindVolumeByName(name string) (string, bool, error) {
	if name == "" {
		return "", false, fmt.Errorf("volume name is required")
	}
	endpoint := v.os.VolumeEndpoint + "/" + v.os.ProjectID + "/volumes/detail?" +
		url.Values{"name": {name}}.Encode()
	data, err := v.os.do("GET", endpoint, nil)
	if err != nil {
		return "", false, err
	}
	var response struct {
		Volumes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"volumes"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", false, fmt.Errorf("decode Cinder volume list response: %w", err)
	}
	for _, volume := range response.Volumes {
		if volume.Name == name && volume.ID != "" {
			return volume.ID, true, nil
		}
	}
	return "", false, nil
}

func (v *VolumeClient) AttachVolume(serverID, volumeID string) error {
	payload := map[string]interface{}{
		"volumeAttachment": map[string]interface{}{
			"volumeId": volumeID,
		},
	}
	if serverID == "" || volumeID == "" {
		return fmt.Errorf("server and volume IDs are required")
	}
	_, err := v.os.do("POST", fmt.Sprintf(
		"%s/servers/%s/os-volume_attachments", v.os.ComputeEndpoint, serverID,
	), payload)
	return err
}

func (v *VolumeClient) GetVolumeState(volumeID string) (string, error) {
	if volumeID == "" {
		return "", fmt.Errorf("volume ID is required")
	}
	data, err := v.os.do(
		"GET", v.os.VolumeEndpoint+"/"+v.os.ProjectID+"/volumes/"+volumeID, nil,
	)
	if err != nil {
		return "", err
	}
	var resp struct {
		Volume struct {
			Status string `json:"status"`
		} `json:"volume"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("decode Cinder volume response: %w", err)
	}
	if resp.Volume.Status == "" {
		return "", fmt.Errorf("Cinder volume response omitted status")
	}
	return resp.Volume.Status, nil
}

func (v *VolumeClient) DeleteVolume(volumeID string) error {
	if volumeID == "" {
		return fmt.Errorf("volume ID is required")
	}
	_, err := v.os.do("DELETE", v.os.VolumeEndpoint+"/"+v.os.ProjectID+"/volumes/"+volumeID, nil)
	return err
}
