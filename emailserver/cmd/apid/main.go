package main

import (
	"log"
	"net/http"

	"atonixcorp/emailserver/internal/config"
	"atonixcorp/emailserver/internal/logging"
	"atonixcorp/emailserver/internal/storage"
)

func main() {
	cfg, err := config.Load("configs/emailserver.yaml")
	if err != nil {
		log.Fatal(err)
	}
	cfg.ApplyEnv()

	logger := logging.New(cfg.Log.Level)

	db, err := storage.OpenPostgres(cfg.DB.DSN)
	if err != nil {
		logger.Error("db:", err)
		return
	}
	_ = db
	mux := http.NewServeMux()

	mux.HandleFunc("/api/inbox", func(w http.ResponseWriter, r *http.Request) {
		// TODO: list inbox messages
		w.Write([]byte("[]"))
	})

	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		// TODO: get message
		w.Write([]byte("{}"))
	})

	logger.Info("API listening on ", cfg.API.ListenAddr)
	http.ListenAndServe(cfg.API.ListenAddr, mux)
}
