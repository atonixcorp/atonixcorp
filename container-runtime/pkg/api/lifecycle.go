package api

// LifecycleRequest performs lifecycle operations on a container.
type LifecycleRequest struct {
	ContainerID string `json:"container_id"`
	Action      string `json:"action"` // start, stop, restart, delete
}

// LifecycleResponse returns the result of a lifecycle operation.
type LifecycleResponse struct {
	ContainerID string `json:"container_id"`
	Action      string `json:"action"`
	Status      string `json:"status"`
}
