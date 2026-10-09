package api

// LogsResponse contains streamed logs for a container.
type LogsResponse struct {
	ContainerID string   `json:"container_id"`
	Logs        []string `json:"logs"`
}
