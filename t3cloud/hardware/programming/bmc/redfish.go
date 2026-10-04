package bmc

// RedfishClient is a placeholder for future native Redfish integration.
type RedfishClient struct {
	// TODO: add endpoint, credentials, HTTP client, etc.
}

// NewRedfishClient creates a new Redfish client.
func NewRedfishClient() *RedfishClient {
	return &RedfishClient{}
}

// In future you can implement:
// func (c *RedfishClient) ReadSensors() ([]Sensor, error) { ... }
// func (c *RedfishClient) PowerCycle() error { ... }
// func (c *RedfishClient) PowerReset() error { ... }
