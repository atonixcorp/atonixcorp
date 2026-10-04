package main

import (
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/lib/pq"
)

func runMigrateDB() {
	dsn := flag.String("dsn", "postgres://user:pass@localhost:5432/emailserver?sslmode=disable", "Postgres DSN")
	flag.Parse()

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		fmt.Println("Error connecting to DB:", err)
		return
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		fmt.Println("Migration failed:", err)
		return
	}

	fmt.Println("Migration completed successfully")
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS mailboxes (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL,
            name TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT now()
        );`,
		`CREATE TABLE IF NOT EXISTS messages (
            id UUID PRIMARY KEY,
            user_id UUID NOT NULL,
            mailbox TEXT NOT NULL,
            subject TEXT,
            from_addr TEXT,
            to_addr TEXT[],
            date TIMESTAMP,
            body BYTEA,
            raw BYTEA,
            size INT
        );`,
		`CREATE TABLE IF NOT EXISTS outgoing_queue (
            id UUID PRIMARY KEY,
            from_addr TEXT NOT NULL,
            to_addr TEXT[] NOT NULL,
            raw BYTEA NOT NULL,
            tenant TEXT,
            attempts INT DEFAULT 0,
            max_retry INT DEFAULT 5,
            next_retry TIMESTAMP DEFAULT now(),
            status TEXT DEFAULT 'pending',
            last_error TEXT
        );`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}
