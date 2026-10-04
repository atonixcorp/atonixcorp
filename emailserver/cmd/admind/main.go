package main

import (
	"log"
	"net/http"

	"atonixcorp/emailserver/internal/analytics"
	"atonixcorp/emailserver/internal/config"
	"atonixcorp/emailserver/internal/logging"
	"atonixcorp/emailserver/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load("configs/emailserver.yaml")
	if err != nil {
		log.Fatal("config:", err)
	}
	cfg.ApplyEnv()

	// Logger
	logger := logging.New(cfg.Log.Level)

	// DB
	db, err := storage.OpenPostgres(cfg.DB.DSN)
	if err != nil {
		logger.Error("db:", err)
		return
	}
	_ = db
	// Analytics
	transport := analytics.NewHTTPTransport("http://localhost:9090/events")
	analyticsClient := analytics.NewClient(transport, logger)

	// Admin HTTP router
	mux := http.NewServeMux()

	mux.HandleFunc("/admin/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		// TODO: list users
		w.Write([]byte("[]"))
	})

	mux.HandleFunc("/admin/domains", func(w http.ResponseWriter, r *http.Request) {
		// TODO: list domains
		w.Write([]byte("[]"))
	})

	mux.HandleFunc("/admin/queue", func(w http.ResponseWriter, r *http.Request) {
		// TODO: queue stats
		w.Write([]byte("[]"))
	})

	logger.Info("Admin API listening on ", cfg.API.ListenAddr)
	http.ListenAndServe(cfg.API.ListenAddr, mux)

	analyticsClient.Close()
}
