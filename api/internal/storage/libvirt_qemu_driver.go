package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
)

var libvirtVolumeIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type LibvirtQEMUConfig struct {
	URI         string
	StoragePath string
}

type storageCommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execStorageCommandRunner struct{}

func (execStorageCommandRunner) Run(
	ctx context.Context, name string, args ...string,
) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type LibvirtQEMUDriver struct {
	uri    string
	root   string
	runner storageCommandRunner
}

type qemuImageInfo struct {
	Format      string `json:"format"`
	VirtualSize int64  `json:"virtual-size"`
}

func NewLibvirtQEMUDriver(config LibvirtQEMUConfig) (*LibvirtQEMUDriver, error) {
	config.URI = strings.TrimSpace(config.URI)
	if config.URI == "" {
		return nil, errors.New("Libvirt URI is required")
	}
	if err := validateLibvirtStorageURI(config.URI); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.StoragePath) == "" || !filepath.IsAbs(config.StoragePath) {
		return nil, errors.New("Libvirt storage path must be an absolute mounted host path")
	}
	root, err := filepath.Abs(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("resolve Libvirt storage path: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect Libvirt storage path: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Libvirt storage path must be an existing directory, not a symlink")
	}
	return newLibvirtQEMUDriver(config.URI, root, execStorageCommandRunner{}), nil
}

func newLibvirtQEMUDriver(uri, root string, runner storageCommandRunner) *LibvirtQEMUDriver {
	return &LibvirtQEMUDriver{uri: uri, root: root, runner: runner}
}

func (d *LibvirtQEMUDriver) Name() string { return "libvirt-qemu" }

func (d *LibvirtQEMUDriver) VolumeTypes(ctx context.Context) ([]VolumeType, error) {
	if err := d.checkConnection(ctx); err != nil {
		return nil, err
	}
	return []VolumeType{{
		ID: "libvirt-qcow2", Name: "Libvirt/QEMU qcow2", Driver: d.Name(),
		Region: "libvirt-host", AvailabilityZones: []string{},
	}}, nil
}

func (d *LibvirtQEMUDriver) ValidateVolume(
	_ context.Context, region, zone, storageClass string, encrypted bool,
) error {
	if region != "libvirt-host" || storageClass != "libvirt-qcow2" ||
		(zone != "" && zone != "auto") {
		return errors.New("Libvirt/QEMU only supports the configured libvirt-host region and qcow2 type")
	}
	if encrypted {
		return errors.New("Libvirt/QEMU qcow2 volumes do not provide configured encryption")
	}
	return nil
}

func (d *LibvirtQEMUDriver) EnsureVolume(
	ctx context.Context, volume models.StorageVolume,
) (ProviderVolume, error) {
	if err := validateLibvirtVolume(volume); err != nil {
		return ProviderVolume{}, err
	}
	if err := d.checkConnection(ctx); err != nil {
		return ProviderVolume{}, err
	}
	path := d.volumePath(volume.ID)
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return ProviderVolume{}, errors.New("existing Libvirt volume is not a regular file")
		}
		return d.inspectImage(ctx, volume.ID, volume.SizeGB)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ProviderVolume{}, fmt.Errorf("inspect Libvirt volume path: %w", err)
	}
	size := strconv.FormatInt(volume.SizeGB, 10) + "G"
	if _, err := d.run(ctx, "qemu-img", "create", "-f", "qcow2", "-o", "preallocation=metadata", path, size); err != nil {
		if _, statErr := os.Lstat(path); statErr == nil {
			return d.inspectImage(ctx, volume.ID, volume.SizeGB)
		}
		return ProviderVolume{}, fmt.Errorf("create Libvirt qcow2 volume: %w", err)
	}
	if err := os.Chmod(path, 0660); err != nil {
		return ProviderVolume{}, fmt.Errorf("set Libvirt volume permissions: %w", err)
	}
	return d.inspectImage(ctx, volume.ID, volume.SizeGB)
}

func (d *LibvirtQEMUDriver) InspectVolume(ctx context.Context, id string) (ProviderVolume, error) {
	if !libvirtVolumeIDPattern.MatchString(id) {
		return ProviderVolume{}, errors.New("invalid Libvirt volume identifier")
	}
	if err := d.checkConnection(ctx); err != nil {
		return ProviderVolume{}, err
	}
	return d.inspectImage(ctx, id, 0)
}

func (d *LibvirtQEMUDriver) DeleteVolume(
	ctx context.Context, volume models.StorageVolume,
) (bool, error) {
	if err := validateLibvirtVolume(volume); err != nil {
		return false, err
	}
	if err := d.checkConnection(ctx); err != nil {
		return false, err
	}
	path := d.volumePath(volume.ID)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect Libvirt volume path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("refusing to delete a Libvirt volume that is not a regular file")
	}
	if _, err := d.inspectImage(ctx, volume.ID, volume.SizeGB); err != nil {
		return false, fmt.Errorf("verify Libvirt volume before deletion: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("delete Libvirt volume: %w", err)
	}
	return true, nil
}

func (d *LibvirtQEMUDriver) inspectImage(
	ctx context.Context, id string, expectedSizeGB int64,
) (ProviderVolume, error) {
	path := d.volumePath(id)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ProviderVolume{}, ErrProviderNotFound
	}
	if err != nil {
		return ProviderVolume{}, fmt.Errorf("inspect Libvirt volume path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return ProviderVolume{}, errors.New("Libvirt volume is not a regular file")
	}
	output, err := d.run(ctx, "qemu-img", "info", "--output=json", path)
	if err != nil {
		return ProviderVolume{}, fmt.Errorf("inspect Libvirt qcow2 volume: %w", err)
	}
	var image qemuImageInfo
	if err := json.Unmarshal(output, &image); err != nil {
		return ProviderVolume{}, fmt.Errorf("decode qemu-img information: %w", err)
	}
	if image.Format != "qcow2" || image.VirtualSize <= 0 ||
		(expectedSizeGB > 0 && image.VirtualSize != expectedSizeGB*1024*1024*1024) {
		return ProviderVolume{}, errors.New("existing Libvirt qcow2 volume has an unexpected format or capacity")
	}
	return ProviderVolume{ID: id, Status: "available"}, nil
}

func (d *LibvirtQEMUDriver) checkConnection(ctx context.Context) error {
	if _, err := d.run(ctx, "virsh", "--connect", d.uri, "uri"); err != nil {
		return fmt.Errorf("connect to configured Libvirt host: %w", err)
	}
	return nil
}

func (d *LibvirtQEMUDriver) run(
	ctx context.Context, name string, args ...string,
) ([]byte, error) {
	if d.runner == nil {
		return nil, errors.New("Libvirt command runner is not configured")
	}
	commandCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	output, err := d.runner.Run(commandCtx, name, args...)
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			return nil, fmt.Errorf("%s failed: %w", name, err)
		}
		return nil, fmt.Errorf("%s failed: %s: %w", name, details, err)
	}
	return output, nil
}

func (d *LibvirtQEMUDriver) volumePath(id string) string {
	return filepath.Join(d.root, id+".qcow2")
}

func validateLibvirtStorageURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "qemu" && parsed.Scheme != "qemu+unix") ||
		parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" ||
		parsed.Path != "/system" {
		return errors.New("Libvirt URI must be a local qemu:///system or qemu+unix:///system connection")
	}
	if parsed.Scheme == "qemu" && parsed.RawQuery != "" {
		return errors.New("Libvirt qemu URI must not include query parameters")
	}
	if parsed.Scheme == "qemu+unix" {
		query := parsed.Query()
		if len(query) != 1 || !filepath.IsAbs(query.Get("socket")) {
			return errors.New("Libvirt qemu+unix URI must specify one absolute socket path")
		}
	}
	return nil
}

func validateLibvirtVolume(volume models.StorageVolume) error {
	if !libvirtVolumeIDPattern.MatchString(volume.ID) {
		return errors.New("invalid Libvirt volume identifier")
	}
	if volume.SizeGB < 1 || volume.SizeGB > 1024 {
		return errors.New("Libvirt volume size must be between 1 and 1024 GB")
	}
	return nil
}

var _ BlockDriver = (*LibvirtQEMUDriver)(nil)
