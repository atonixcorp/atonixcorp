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

	mux.Handle("/", http.FileServer(http.Dir("webmail/static")))

	mux.HandleFunc("/webmail/inbox", func(w http.ResponseWriter, r *http.Request) {
		// TODO: inbox
		w.Write([]byte("[]"))
	})

	logger.Info("Webmail listening on :8081")
	http.ListenAndServe(":8081", mux)
}
