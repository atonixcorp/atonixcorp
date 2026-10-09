package models

import "time"

type BuildJob struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	SourcePath  string      `json:"source_path"`
	Runtime     string      `json:"runtime"`
	Status      BuildStatus `json:"status"`
	ImageID     string      `json:"image_id"`
	Logs        []string    `json:"logs"`
	CreatedAt   time.Time   `json:"created_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

type BuildStatus string

const (
	BuildQueued  BuildStatus = "queued"
	BuildRunning BuildStatus = "running"
	BuildSuccess BuildStatus = "success"
	BuildFailed  BuildStatus = "failed"
)
