package config

import (
	"net/url"
	"testing"
)

func TestDatabaseDSNBuildsEscapedCredentials(t *testing.T) {
	t.Setenv("ATONIXCORP_DB_DSN", "")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "atonixcorp")
	t.Setenv("DB_PASSWORD", "p@ss:/word%")
	t.Setenv("DB_NAME", "atonixcorp")
	t.Setenv("DB_SSLMODE", "disable")

	dsn, err := DatabaseDSN()
	if err != nil {
		t.Fatalf("build database DSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse generated database DSN: %v", err)
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss:/word%" {
		t.Fatalf("database password was not safely encoded, got %q", password)
	}
}
