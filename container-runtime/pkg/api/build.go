package api

// BuildRequest is sent when a user wants to build a container image.
type BuildRequest struct {
	ProjectID  string `json:"project_id"`
	SourcePath string `json:"source_path"`
	Runtime    string `json:"runtime"`
}

// BuildResponse is returned after a build job completes.
type BuildResponse struct {
	BuildID string   `json:"build_id"`
	ImageID string   `json:"image_id"`
	Logs    []string `json:"logs"`
	Status  string   `json:"status"`
}
