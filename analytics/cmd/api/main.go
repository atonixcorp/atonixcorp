package main

import (
	"encoding/json"
	"log"
	"net/http"

	"atonixcorp/analytics/internal/config"
	"atonixcorp/analytics/internal/events"
	"atonixcorp/analytics/internal/metrics"
	"atonixcorp/analytics/internal/storage"
)

func main() {
	cfg := config.Load()
	db := storage.Connect(cfg.DBURL)
	storage.InitSchema(db)

	mux := http.NewServeMux()

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		metrics.Requests.Inc()
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var evt events.Event
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "invalid event body", http.StatusBadRequest)
			return
		}
		if evt.ID == "" || evt.Kind == "" {
			http.Error(w, "event id and kind are required", http.StatusBadRequest)
			return
		}
		payload, err := json.Marshal(evt.Payload)
		if err != nil {
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}
		_, err = db.ExecContext(r.Context(),
			`INSERT INTO analytics_events (id, kind, payload, ts, source)
             VALUES ($1, $2, $3, $4, $5)
             ON CONFLICT (id) DO NOTHING`,
			evt.ID, string(evt.Kind), string(payload), evt.Timestamp, evt.Source,
		)
		if err != nil {
			log.Printf("persist analytics event: %v", err)
			http.Error(w, "event persistence unavailable", http.StatusServiceUnavailable)
			return
		}
		metrics.EventsIngested.Inc()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		out := map[string]any{
			"requests_total":  metrics.Requests.Value(),
			"events_ingested": metrics.EventsIngested.Value(),
		}
		json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/analytics/usage", func(w http.ResponseWriter, r *http.Request) {
		metrics.Requests.Inc()
		rows := make([]struct {
			Kind  string `db:"kind" json:"kind"`
			Count int64  `db:"count" json:"count"`
		}, 0)
		if err := db.SelectContext(r.Context(), &rows, `SELECT kind, COUNT(*) AS count FROM analytics_events GROUP BY kind`); err != nil {
			log.Printf("query analytics usage: %v", err)
			http.Error(w, "analytics query unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	})

	log.Println("AtonixCorp Analytics API running on :9000")
	http.ListenAndServe(":9000", mux)
}
