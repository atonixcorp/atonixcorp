package logging

import (
	"fmt"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Container string
	Message   string
}

type LogStreamer struct{}

func NewLogStreamer() *LogStreamer {
	return &LogStreamer{}
}

func (ls *LogStreamer) Stream(containerID string) (<-chan LogEntry, error) {
	ch := make(chan LogEntry)

	go func() {
		defer close(ch)
		// TODO: attach to container logs via containerd API
		for i := 0; i < 5; i++ {
			ch <- LogEntry{
				Timestamp: time.Now(),
				Container: containerID,
				Message:   fmt.Sprintf("log line %d", i),
			}
			time.Sleep(1 * time.Second)
		}
	}()

	return ch, nil
}
