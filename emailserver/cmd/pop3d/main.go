package main

import (
	"log"

	"atonixcorp/emailserver/internal/auth"
	"atonixcorp/emailserver/internal/config"
	"atonixcorp/emailserver/internal/logging"
	"atonixcorp/emailserver/internal/pop3"
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

	srv := pop3.NewServer(pop3.Config{ListenAddr: cfg.POP3.ListenAddr}, logger, store, authSvc)

	logger.Info("Starting POP3 server...")
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("pop3:", err)
	}
}
