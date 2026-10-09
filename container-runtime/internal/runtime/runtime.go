package runtime

import (
	"fmt"
)

type Runtime struct{}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (rt *Runtime) Start(imageID string) (string, error) {
	// TODO: use containerd API to start container
	containerID := fmt.Sprintf("ctr-%s", imageID)
	fmt.Printf("Starting container %s\n", containerID)
	return containerID, nil
}

func (rt *Runtime) Stop(containerID string) error {
	fmt.Printf("Stopping container %s\n", containerID)
	return nil
}

func (rt *Runtime) Delete(containerID string) error {
	fmt.Printf("Deleting container %s\n", containerID)
	return nil
}
