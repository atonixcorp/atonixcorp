package minio

import (
	"context"
	"testing"
	"time"
)

func TestConfigurationAndUnavailableStorage(t *testing.T) {
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("MINIO_ENDPOINT", "127.0.0.1:1")
	t.Setenv("MINIO_ACCESS_KEY", "samorca")
	t.Setenv("MINIO_SECRET_KEY", "test-secret")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("MINIO_ALLOW_INSECURE_DEVELOPMENT", "true")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("administrator identity accepted")
	}
	t.Setenv("MINIO_ACCESS_KEY", "application")
	t.Setenv("MINIO_ALLOW_INSECURE_DEVELOPMENT", "false")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("insecure transport accepted without development opt-in")
	}
	t.Setenv("MINIO_ALLOW_INSECURE_DEVELOPMENT", "true")
	p, err := FromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Health(ctx); err == nil {
		t.Fatal("unavailable server reported healthy")
	}
}
