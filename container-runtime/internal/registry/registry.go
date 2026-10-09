package registry

import "fmt"

type RegistryClient struct {
	BaseURL string
}

func NewRegistryClient(url string) *RegistryClient {
	return &RegistryClient{BaseURL: url}
}

func (r *RegistryClient) Push(imageID string) error {
	// TODO: integrate with Harbor or Docker Registry v2 API
	fmt.Printf("Pushing image %s to registry %s\n", imageID, r.BaseURL)
	return nil
}

func (r *RegistryClient) Pull(imageID string) error {
	fmt.Printf("Pulling image %s from registry %s\n", imageID, r.BaseURL)
	return nil
}
