package models

// LifecycleAction represents a lifecycle operation on a container.
type LifecycleAction string

const (
	ActionStart   LifecycleAction = "start"
	ActionStop    LifecycleAction = "stop"
	ActionRestart LifecycleAction = "restart"
	ActionDelete  LifecycleAction = "delete"
)

// LifecycleRequest is used by API and scheduler.
type LifecycleRequest struct {
	ContainerID string          `json:"container_id"`
	Action      LifecycleAction `json:"action"`
}
