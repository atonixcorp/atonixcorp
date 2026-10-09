package lifecycle

import (
	"context"
	"fmt"
)

type ContainerAction string

const (
	ActionStart   ContainerAction = "start"
	ActionStop    ContainerAction = "stop"
	ActionRestart ContainerAction = "restart"
	ActionDelete  ContainerAction = "delete"
)

type LifecycleManager struct{}

func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{}
}

func (l *LifecycleManager) Execute(ctx context.Context, containerID string, action ContainerAction) error {
	fmt.Printf("Executing %s on container %s\n", action, containerID)

	// TODO: call runtime package to perform actual container operations
	return nil
}
