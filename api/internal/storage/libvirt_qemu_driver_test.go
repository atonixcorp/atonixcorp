package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"atonixcorp/api/internal/models"
)

type fakeStorageCommandRunner struct {
	calls []string
}

func (r *fakeStorageCommandRunner) Run(
	ctx context.Context, name string, args ...string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	switch name {
	case "virsh":
		if len(args) != 3 || args[0] != "--connect" || args[2] != "uri" {
			return nil, fmt.Errorf("unexpected virsh arguments: %v", args)
		}
		return []byte(args[1]), nil
	case "qemu-img":
		switch args[0] {
		case "create":
			if len(args) != 7 || args[1] != "-f" || args[2] != "qcow2" ||
				args[3] != "-o" || args[4] != "preallocation=metadata" {
				return nil, fmt.Errorf("unexpected qemu-img create arguments: %v", args)
			}
			size, err := strconv.ParseInt(strings.TrimSuffix(args[6], "G"), 10, 64)
			if err != nil {
				return nil, err
			}
			file, err := os.OpenFile(args[5], os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			if err := file.Truncate(size * 1024 * 1024 * 1024); err != nil {
				return nil, err
			}
			return []byte("created"), nil
		case "info":
			if len(args) != 3 || args[1] != "--output=json" {
				return nil, fmt.Errorf("unexpected qemu-img info arguments: %v", args)
			}
			info, err := os.Stat(args[2])
			if err != nil {
				return nil, err
			}
			return []byte(fmt.Sprintf(
				`{"format":"qcow2","virtual-size":%d}`, info.Size(),
			)), nil
		default:
			return nil, fmt.Errorf("unexpected qemu-img operation: %v", args)
		}
	default:
		return nil, fmt.Errorf("unexpected executable %q", name)
	}
}

func TestLibvirtQEMUDriverVolumeLifecycle(t *testing.T) {
	root := t.TempDir()
	runner := &fakeStorageCommandRunner{}
	driver := newLibvirtQEMUDriver(
		"qemu+unix:///system?socket=/run/libvirt/libvirt-sock", root, runner,
	)
	ctx := context.Background()
	types, err := driver.VolumeTypes(ctx)
	if err != nil {
		t.Fatalf("list volume types: %v", err)
	}
	if len(types) != 1 || types[0].ID != "libvirt-qcow2" || types[0].Encrypted {
		t.Fatalf("unexpected volume types: %+v", types)
	}
	if err := driver.ValidateVolume(ctx, "libvirt-host", "auto", "libvirt-qcow2", false); err != nil {
		t.Fatalf("validate supported volume: %v", err)
	}
	if err := driver.ValidateVolume(ctx, "libvirt-host", "auto", "libvirt-qcow2", true); err == nil {
		t.Fatal("expected unsupported encryption to be rejected")
	}
	volume := models.StorageVolume{ID: "8a34f1d09cc64f8384a4b86e4c30e1d3", SizeGB: 2}
	provider, err := driver.EnsureVolume(ctx, volume)
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if provider.ID != volume.ID || provider.Status != "available" {
		t.Fatalf("unexpected provider volume: %+v", provider)
	}
	path := filepath.Join(root, volume.ID+".qcow2")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("volume image not created: %v", err)
	}
	provider, err = driver.EnsureVolume(ctx, volume)
	if err != nil || provider.Status != "available" {
		t.Fatalf("idempotent create returned %+v, %v", provider, err)
	}
	if len(runner.calls) != 6 {
		t.Fatalf("commands called = %d, want three connection checks, one create, and two inspections", len(runner.calls))
	}
	if _, err := driver.InspectVolume(ctx, volume.ID); err != nil {
		t.Fatalf("inspect volume: %v", err)
	}
	deleted, err := driver.DeleteVolume(ctx, volume)
	if err != nil || !deleted {
		t.Fatalf("delete volume = %t, %v", deleted, err)
	}
	deleted, err = driver.DeleteVolume(ctx, volume)
	if err != nil || !deleted {
		t.Fatalf("idempotent delete = %t, %v", deleted, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("volume image remains after deletion: %v", err)
	}
}

func TestNewLibvirtQEMUDriverRequiresDedicatedMountedPathAndLocalURI(t *testing.T) {
	root := t.TempDir()
	for _, uri := range []string{
		"qemu:///system",
		"qemu+unix:///system?socket=/run/libvirt/libvirt-sock",
	} {
		if err := validateLibvirtStorageURI(uri); err != nil {
			t.Errorf("valid local URI %q rejected: %v", uri, err)
		}
	}
	for _, test := range []struct {
		name string
		uri  string
		path string
	}{
		{name: "missing URI", path: root},
		{name: "remote URI", uri: "qemu+ssh://host/system", path: root},
		{name: "invalid socket URI", uri: "qemu+unix:///system?socket=relative", path: root},
		{name: "relative storage path", uri: "qemu:///system", path: "relative"},
		{name: "missing storage path", uri: "qemu:///system", path: filepath.Join(root, "missing")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewLibvirtQEMUDriver(LibvirtQEMUConfig{
				URI: test.uri, StoragePath: test.path,
			}); err == nil {
				t.Fatal("expected invalid Libvirt configuration to fail")
			}
		})
	}
}

func TestLibvirtQEMUDriverRejectsUnsafeIdentifiersAndSymlinks(t *testing.T) {
	root := t.TempDir()
	driver := newLibvirtQEMUDriver("qemu:///system", root, &fakeStorageCommandRunner{})
	if _, err := driver.InspectVolume(context.Background(), "../outside"); err == nil {
		t.Fatal("expected invalid volume identifier to fail")
	}
	id := "8a34f1d09cc64f8384a4b86e4c30e1d3"
	if err := os.Symlink(filepath.Join(root, "missing-target"), filepath.Join(root, id+".qcow2")); err != nil {
		t.Fatal(err)
	}
	if _, err := driver.EnsureVolume(context.Background(), models.StorageVolume{ID: id, SizeGB: 1}); err == nil {
		t.Fatal("expected symlink volume path to be rejected")
	}
}
