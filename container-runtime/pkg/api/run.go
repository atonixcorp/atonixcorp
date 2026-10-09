package api

// RunRequest starts a new container instance.
type RunRequest struct {
	ImageID  string `json:"image_id"`
	CPU      int    `json:"cpu"`
	MemoryMB int    `json:"memory_mb"`
}

// RunResponse contains details about the newly started container.
type RunResponse struct {
	ContainerID string `json:"container_id"`
	Node        string `json:"node"`
	Status      string `json:"status"`
}
