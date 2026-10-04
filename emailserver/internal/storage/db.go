package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// OpenPostgres opens a PostgreSQL database connection using the DSN.
func OpenPostgres(dsn string) (*sql.DB, error) {
	return OpenPostgresDB(dsn)
}

// OpenPostgresDB opens a PostgreSQL database connection using the DSN.
func OpenPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
