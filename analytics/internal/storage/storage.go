package storage

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect(url string) *sqlx.DB {
	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		log.Fatalf("DB connect error: %v", err)
	}
	return db
}

func InitSchema(db *sqlx.DB) {
	schema := `
    CREATE TABLE IF NOT EXISTS analytics_events (
        id TEXT PRIMARY KEY,
        kind TEXT,
        payload JSONB,
        ts TIMESTAMP,
        source TEXT
    );
    `
	db.MustExec(schema)
}
