package build

import (
	"context"
	"fmt"
)

type BuildRequest struct {
	ProjectID string
	SourceDir string
	Runtime   string
}

type BuildResult struct {
	ImageID string
	Logs    []string
}

type Builder interface {
	Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
}

type DefaultBuilder struct{}

func NewDefaultBuilder() *DefaultBuilder {
	return &DefaultBuilder{}
}

func (b *DefaultBuilder) Build(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	// TODO: Auto-detect language, generate Dockerfile, run BuildKit/Docker API
	fmt.Printf("Building project %s using runtime %s\n", req.ProjectID, req.Runtime)

	return &BuildResult{
		ImageID: fmt.Sprintf("registry.local/%s:latest", req.ProjectID),
		Logs:    []string{"build started", "build completed"},
	}, nil
}
