package models

import "time"

// LogEntry represents a single log line from a container.
type LogEntry struct {
	ContainerID string    `json:"container_id"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
}
