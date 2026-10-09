package models

import "time"

// Container represents a running or stopped container instance.
type Container struct {
	ID        string          `json:"id"`
	ImageID   string          `json:"image_id"`
	Node      string          `json:"node"`
	Status    ContainerStatus `json:"status"`
	CPU       int             `json:"cpu"`
	MemoryMB  int             `json:"memory_mb"`
	CreatedAt time.Time       `json:"created_at"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	StoppedAt *time.Time      `json:"stopped_at,omitempty"`
}

type ContainerStatus string

const (
	ContainerPending ContainerStatus = "pending"
	ContainerRunning ContainerStatus = "running"
	ContainerStopped ContainerStatus = "stopped"
	ContainerFailed  ContainerStatus = "failed"
)
