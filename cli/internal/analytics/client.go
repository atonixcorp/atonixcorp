package analytics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

var AnalyticsURL = "http://localhost:9000"

type Event struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"ts"`
	Source    string         `json:"source"`
}

func SendEvent(kind string, payload map[string]any) error {
	evt := Event{
		ID:        time.Now().Format("20060102150405"),
		Kind:      kind,
		Payload:   payload,
		Timestamp: time.Now(),
		Source:    "atcli",
	}

	b, _ := json.Marshal(evt)
	_, err := http.Post(AnalyticsURL+"/events", "application/json", bytes.NewBuffer(b))
	return err
}

func Query(path string) (map[string]any, error) {
	resp, err := http.Get(AnalyticsURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return out, nil
}
