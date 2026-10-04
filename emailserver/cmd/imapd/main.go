package main

import (
	"log"

	"atonixcorp/emailserver/internal/auth"
	"atonixcorp/emailserver/internal/config"
	"atonixcorp/emailserver/internal/imap"
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
	store := storage.NewPostgresStore(db)

	authSvc := auth.NewUserStore(db)

	srv := imap.NewServer(imap.Config{ListenAddr: cfg.IMAP.ListenAddr}, logger, store, authSvc)

	logger.Info("Starting IMAP server...")
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("imap:", err)
	}
}
