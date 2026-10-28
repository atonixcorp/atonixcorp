package controlplane

import (
	"path/filepath"
	"testing"
)

func TestFileStoreCopiesInstanceSlices(t *testing.T) {
	store, err := OpenFileStore(filepath.Join(t.TempDir(), "instances.json"))
	if err != nil {
		t.Fatal(err)
	}

	instance := Instance{
		ID:        "instance-1",
		SSHKeyIDs: []string{"key-1"},
		Tags:      []string{"production"},
		Volumes:   []VolumeRequest{{Name: "data", Type: "nvme_ssd", SizeGB: 100}},
		Provisioning: []ProvisioningStep{{
			Name:   "creating_instance",
			Status: "queued",
		}},
	}
	if err := store.Create([]Instance{instance}); err != nil {
		t.Fatal(err)
	}

	instance.Tags[0] = "changed"
	instance.Volumes[0].SizeGB = 1
	instance.Provisioning[0].Status = "running"
	stored, err := store.Get(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Tags[0] != "production" || stored.Volumes[0].SizeGB != 100 || stored.Provisioning[0].Status != "queued" {
		t.Fatalf("caller mutations changed persisted data: %+v", stored)
	}

	stored.Tags[0] = "also-changed"
	listed, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if listed[0].Tags[0] != "production" {
		t.Fatalf("mutating a returned instance changed stored data: %+v", listed[0])
	}
}

func TestFileStoreRejectsDuplicateIDsWithinBatch(t *testing.T) {
	store, err := OpenFileStore(filepath.Join(t.TempDir(), "instances.json"))
	if err != nil {
		t.Fatal(err)
	}
	instances := []Instance{{ID: "duplicate"}, {ID: "duplicate"}}
	if err := store.Create(instances); err == nil {
		t.Fatal("expected duplicate IDs in a batch to be rejected")
	}
	listed, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("duplicate batch partially created instances: %+v", listed)
	}
}
