package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestLoadUsesEnvironmentSecretsAndBuildsEscapedDatabaseDSN(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("HPC_DB_PASSWORD", "p@ss:/?word")
	t.Setenv("HPC_S3_SECURE", "true")
	t.Setenv("HPC_SLURM_ENABLED", "false")

	config, err := Load("hpc.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if config.Auth.JWTSigningKey != strings.Repeat("k", 32) || !config.Storage.S3.Secure {
		t.Fatal("environment configuration was not applied")
	}
	parsed, err := url.Parse(config.DatabaseDSN())
	if err != nil {
		t.Fatalf("parse generated database DSN: %v", err)
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss:/?word" {
		t.Fatalf("database password was not safely escaped: %q", password)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("unexpected database sslmode: %q", parsed.Query().Get("sslmode"))
	}
}

func TestLoadRejectsInvalidBooleanEnvironmentOverride(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("HPC_S3_SECURE", "sometimes")
	if _, err := Load("hpc.yaml"); err == nil {
		t.Fatal("invalid boolean environment value accepted")
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HPC_JWT_SIGNING_KEY", strings.Repeat("k", 32))
	t.Setenv("HPC_S3_ENDPOINT", "minio:9000")
	t.Setenv("HPC_S3_BUCKET", "datasets")
	t.Setenv("HPC_S3_ACCESS_KEY", "access-key")
	t.Setenv("HPC_S3_SECRET_KEY", "secret-key")
	t.Setenv("HPC_DB_HOST", "postgres")
	t.Setenv("HPC_DB_PORT", "5432")
	t.Setenv("HPC_DB_USER", "hpc")
	t.Setenv("HPC_DB_PASSWORD", "database-password")
	t.Setenv("HPC_DB_NAME", "hpc")
	t.Setenv("HPC_DB_SSLMODE", "disable")
}
