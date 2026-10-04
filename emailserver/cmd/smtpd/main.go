package main

import (
	"log"

	"atonixcorp/emailserver/internal/config"
	"atonixcorp/emailserver/internal/domain"
	"atonixcorp/emailserver/internal/logging"
	"atonixcorp/emailserver/internal/queue"
	"atonixcorp/emailserver/internal/smtp"
	"atonixcorp/emailserver/internal/spam"
	"atonixcorp/emailserver/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load("configs/emailserver.yaml")
	if err != nil {
		log.Fatal(err)
	}
	cfg.ApplyEnv()

	// Logger
	logger := logging.New(cfg.Log.Level)

	// Database
	db, err := storage.OpenPostgresDB(cfg.DB.DSN)
	if err != nil {
		logger.Error("db:", err)
		return
	}
	store := storage.NewPostgresStore(db)

	// Queue
	queueStore := queue.NewPostgresQueueStore(db)
	q := queue.New(queueStore, logger)

	// Spam filter
	spamFilter := spam.NewSimpleFilter(logger)

	// Domain service
	domainSvc := domain.New(db)

	// SMTP handler
	handler := &smtp.DefaultHandler{}

	// SMTP server
	srv := smtp.NewServer(
		smtp.Config{
			ListenAddr: cfg.SMTP.ListenAddr,
			TLSCert:    cfg.SMTP.TLSCert,
			TLSKey:     cfg.SMTP.TLSKey,
		},
		logger,
		store,
		q,
		spamFilter,
		domainSvc,
		handler,
	)

	logger.Info("Starting SMTP server...")
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("smtp:", err)
	}
}
