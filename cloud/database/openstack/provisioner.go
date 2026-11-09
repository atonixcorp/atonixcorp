package openstack

import (
	"atonixcorp/cloud/database/core"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type InfrastructureClient interface {
	FindServerByName(string) (string, bool, error)
	CreateServer(string, string, string, string, string) (string, error)
	FindVolumeByName(string) (string, bool, error)
	CreateVolume(string, int) (string, error)
	GetServerState(string) (string, string, error)
	IsVolumeAttached(string, string) (bool, error)
	GetVolumeState(string) (string, error)
	AttachVolume(string, string) error
}

type ProvisionerConfig struct {
	EngineImages    map[string]string
	PlanFlavors     map[string]string
	NetworkID       string
	SecurityGroupID string
	Region          string
}

type DatabaseProvisioner struct {
	client InfrastructureClient
	config ProvisionerConfig
}

type ProvisioningState struct {
	ServerID       string `json:"server_id"`
	ServerStatus   string `json:"server_status"`
	ServerIP       string `json:"server_ip"`
	VolumeID       string `json:"volume_id"`
	VolumeStatus   string `json:"volume_status"`
	VolumeAttached bool   `json:"volume_attached"`
	NetworkID      string `json:"network_id"`
}

func NewDatabaseProvisioner(client InfrastructureClient, config ProvisionerConfig) *DatabaseProvisioner {
	return &DatabaseProvisioner{client: client, config: config}
}

func (*DatabaseProvisioner) Name() string { return "openstack" }
func (provider *DatabaseProvisioner) Supports(engine, version, plan, region string) bool {
	return provider != nil && provider.client != nil && strings.TrimSpace(provider.config.NetworkID) != "" &&
		strings.TrimSpace(provider.config.EngineImages[engine+":"+version]) != "" &&
		strings.TrimSpace(provider.config.PlanFlavors[plan]) != "" &&
		(region == "" || provider.config.Region == "" || region == provider.config.Region)
}

func (provider *DatabaseProvisioner) Provision(instance *core.DBInstance, checkpoint func(string) error) (core.ProvisioningProgress, error) {
	progress := core.ProvisioningProgress{}
	if !provider.Supports(instance.Engine, instance.EngineVersion, instance.Plan, instance.Region) {
		return progress, core.ErrProviderUnavailable
	}
	var state ProvisioningState
	if len(instance.ProviderMetadata) > 0 {
		if err := json.Unmarshal(instance.ProviderMetadata, &state); err != nil {
			return progress, errors.Join(core.ErrProvisioningFailed, err)
		}
	}
	if instance.Provider != provider.Name() || state.NetworkID != "" && state.NetworkID != provider.config.NetworkID {
		return progress, core.ErrProviderUnavailable
	}
	state.NetworkID = provider.config.NetworkID
	save := func(stage string) error {
		payload, err := json.Marshal(state)
		if err != nil {
			return err
		}
		instance.ProviderMetadata = payload
		return checkpoint(stage)
	}
	resourceName := instance.Name + "-" + instance.ID
	if state.ServerID == "" {
		serverID, found, err := provider.client.FindServerByName(resourceName)
		if err != nil {
			return progress, fmt.Errorf("find Nova server: %w", err)
		}
		if !found {
			serverID, err = provider.client.CreateServer(resourceName, provider.config.EngineImages[instance.Engine+":"+instance.EngineVersion], provider.config.PlanFlavors[instance.Plan], provider.config.NetworkID, provider.config.SecurityGroupID)
			if err != nil {
				return progress, fmt.Errorf("create Nova server: %w", err)
			}
		}
		state.ServerID = serverID
		if err := save("ALLOCATING_COMPUTE"); err != nil {
			return progress, err
		}
	}
	status, ip, err := provider.client.GetServerState(state.ServerID)
	if err != nil {
		return progress, fmt.Errorf("check Nova server status: %w", err)
	}
	state.ServerStatus, state.ServerIP = status, ip
	if status == "ERROR" {
		if err := save("ALLOCATING_COMPUTE"); err != nil {
			return progress, err
		}
		return progress, errors.Join(core.ErrProvisioningFailed, errors.New("Nova server entered ERROR state"))
	}
	if status != "ACTIVE" {
		return progress, save("ALLOCATING_COMPUTE")
	}
	if err := save("PROVISIONING_STORAGE"); err != nil {
		return progress, err
	}
	if state.VolumeID == "" {
		volumeName := resourceName + "-data"
		volumeID, found, err := provider.client.FindVolumeByName(volumeName)
		if err != nil {
			return progress, fmt.Errorf("find Cinder volume: %w", err)
		}
		if !found {
			volumeID, err = provider.client.CreateVolume(volumeName, instance.StorageGB)
			if err != nil {
				return progress, fmt.Errorf("create Cinder volume: %w", err)
			}
		}
		state.VolumeID = volumeID
		if err := save("PROVISIONING_STORAGE"); err != nil {
			return progress, err
		}
	}
	volumeStatus, err := provider.client.GetVolumeState(state.VolumeID)
	if err != nil {
		return progress, fmt.Errorf("check Cinder volume status: %w", err)
	}
	state.VolumeStatus = volumeStatus
	switch volumeStatus {
	case "error":
		if err := save("PROVISIONING_STORAGE"); err != nil {
			return progress, err
		}
		return progress, errors.Join(core.ErrProvisioningFailed, errors.New("Cinder volume entered error state"))
	case "available", "in-use":
		attached, err := provider.client.IsVolumeAttached(state.ServerID, state.VolumeID)
		if err != nil {
			return progress, fmt.Errorf("check Nova volume attachment: %w", err)
		}
		if !attached && volumeStatus == "in-use" {
			if err := save("PROVISIONING_STORAGE"); err != nil {
				return progress, err
			}
			return progress, errors.Join(core.ErrProvisioningFailed, errors.New("Cinder volume is attached to another server"))
		}
		if !attached {
			if err := provider.client.AttachVolume(state.ServerID, state.VolumeID); err != nil {
				return progress, fmt.Errorf("attach Cinder volume: %w", err)
			}
		}
		state.VolumeAttached = true
	default:
		return progress, save("PROVISIONING_STORAGE")
	}
	if err := save("PROVISIONING_STORAGE"); err != nil {
		return progress, err
	}
	progress.InfrastructureReady = ip != ""
	return progress, nil
}
