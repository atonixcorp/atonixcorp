package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"atonixcorp/api/internal/models"
)

func TestLocalFileDriverCreateIsIdempotentAndDeleteIsSafe(t *testing.T) {
	driver, err := NewLocalFileDriver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	volume := models.StorageVolume{ID: "8a34f1d09cc64f8384a4b86e4c30e1d3", SizeGB: 1}
	provider, err := driver.EnsureVolume(context.Background(), volume)
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if provider.Status != "available" {
		t.Fatalf("create status = %q, want available", provider.Status)
	}
	path := filepath.Join(driver.root, volume.ID+".img")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 1024*1024*1024 {
		t.Fatalf("backing file size = %d, want 1 GiB", info.Size())
	}
	if _, err := driver.EnsureVolume(context.Background(), volume); err != nil {
		t.Fatalf("retry create volume: %v", err)
	}
	deleted, err := driver.DeleteVolume(context.Background(), volume)
	if err != nil || !deleted {
		t.Fatalf("delete volume: %v", err)
	}
	deleted, err = driver.DeleteVolume(context.Background(), volume)
	if err != nil || !deleted {
		t.Fatalf("retry delete volume: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("backing file should be removed, stat error = %v", err)
	}
}

func TestLocalFileDriverRejectsOutOfRangeCapacity(t *testing.T) {
	driver, err := NewLocalFileDriver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.EnsureVolume(context.Background(), models.StorageVolume{ID: "bad", SizeGB: 0}); err == nil {
		t.Fatal("expected invalid capacity to fail")
	}
	if _, err := os.Stat(filepath.Join(driver.root, "bad.img")); !os.IsNotExist(err) {
		t.Fatalf("invalid request created a backing file: %v", err)
	}
}

func TestLocalFileDriverRejectsEncryptionAndUnsupportedTypes(t *testing.T) {
	driver, err := NewLocalFileDriver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		region      string
		zone        string
		storageType string
		encrypted   bool
	}{
		{name: "encryption", region: "local-development", storageType: "local-file", encrypted: true},
		{name: "region", region: "region-a", storageType: "local-file"},
		{name: "type", region: "local-development", storageType: "fast"},
		{name: "zone", region: "local-development", zone: "zone-a", storageType: "local-file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := driver.ValidateVolume(
				context.Background(), test.region, test.zone, test.storageType, test.encrypted,
			); err == nil {
				t.Fatal("expected unsupported local volume configuration to be rejected")
			}
		})
	}
}
