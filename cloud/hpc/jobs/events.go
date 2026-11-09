package jobs

import (
	"context"
	"time"
)

type EventType string

const (
	EventJobCreated  EventType = "job.created"
	EventJobQueued   EventType = "job.queued"
	EventJobStarted  EventType = "job.started"
	EventJobFinished EventType = "job.finished"
	EventJobFailed   EventType = "job.failed"
	EventJobCanceled EventType = "job.canceled"
)

type Event struct {
	Type      EventType      `json:"type"`
	JobID     string         `json:"job_id"`
	TenantID  string         `json:"tenant_id"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}
