package openstack

type ServiceClient struct {
	compute *ComputeClient
	volume  *VolumeClient
}

func NewServiceClient(client *Client) *ServiceClient {
	return &ServiceClient{
		compute: NewComputeClient(client),
		volume:  NewVolumeClient(client),
	}
}

func (c *ServiceClient) CreateServer(name, imageID, flavorID, networkID, securityGroupID string) (string, error) {
	return c.compute.CreateServer(name, imageID, flavorID, networkID, securityGroupID)
}

func (c *ServiceClient) FindServerByName(name string) (string, bool, error) {
	return c.compute.FindServerByName(name)
}

func (c *ServiceClient) CreateVolume(name string, sizeGB int) (string, error) {
	return c.volume.CreateVolume(name, sizeGB)
}

func (c *ServiceClient) FindVolumeByName(name string) (string, bool, error) {
	return c.volume.FindVolumeByName(name)
}

func (c *ServiceClient) AttachVolume(serverID, volumeID string) error {
	return c.volume.AttachVolume(serverID, volumeID)
}

func (c *ServiceClient) GetServerIP(serverID string) (string, error) {
	return c.compute.GetServerIP(serverID)
}

func (c *ServiceClient) GetServerState(serverID string) (string, string, error) {
	return c.compute.GetServerState(serverID)
}

func (c *ServiceClient) IsVolumeAttached(serverID, volumeID string) (bool, error) {
	return c.compute.IsVolumeAttached(serverID, volumeID)
}

func (c *ServiceClient) GetVolumeState(volumeID string) (string, error) {
	return c.volume.GetVolumeState(volumeID)
}

func (c *ServiceClient) DeleteServer(serverID string) error {
	return c.compute.DeleteServer(serverID)
}

func (c *ServiceClient) DeleteVolume(volumeID string) error {
	return c.volume.DeleteVolume(volumeID)
}
