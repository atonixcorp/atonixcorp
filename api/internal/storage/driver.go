package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"atonixcorp/api/internal/models"
)

var ErrDriverUnavailable = errors.New("storage driver is unavailable")

var ErrProviderNotFound = errors.New("storage provider resource not found")

type VolumeType struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Driver            string   `json:"driver"`
	Region            string   `json:"region"`
	AvailabilityZones []string `json:"availability_zones"`
	Encrypted         bool     `json:"encrypted"`
}

type ProviderVolume struct {
	ID     string
	Status string
}

type BlockDriver interface {
	Name() string
	VolumeTypes(context.Context) ([]VolumeType, error)
	ValidateVolume(context.Context, string, string, string, bool) error
	EnsureVolume(context.Context, models.StorageVolume) (ProviderVolume, error)
	InspectVolume(context.Context, string) (ProviderVolume, error)
	DeleteVolume(context.Context, models.StorageVolume) (bool, error)
}

type LocalFileDriver struct {
	root string
}

func NewLocalFileDriver(root string) (*LocalFileDriver, error) {
	if root == "" {
		return nil, errors.New("storage data directory is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage data directory: %w", err)
	}
	if err := os.MkdirAll(absolute, 0700); err != nil {
		return nil, fmt.Errorf("create storage data directory: %w", err)
	}
	return &LocalFileDriver{root: absolute}, nil
}

func (d *LocalFileDriver) Name() string { return "local-file" }

func (d *LocalFileDriver) VolumeTypes(context.Context) ([]VolumeType, error) {
	return []VolumeType{{
		ID: "local-file", Name: "Local development volume", Driver: d.Name(),
		Region: "local-development", AvailabilityZones: []string{},
	}}, nil
}

func (d *LocalFileDriver) ValidateVolume(
	_ context.Context, region, zone, storageClass string, encrypted bool,
) error {
	if region != "local-development" || storageClass != "local-file" ||
		(zone != "" && zone != "auto") {
		return errors.New("the local development driver only supports the local-development region")
	}
	if encrypted {
		return errors.New("the local development driver does not support volume encryption")
	}
	return nil
}

func (d *LocalFileDriver) EnsureVolume(ctx context.Context, volume models.StorageVolume) (ProviderVolume, error) {
	if err := ctx.Err(); err != nil {
		return ProviderVolume{}, err
	}
	if volume.SizeGB < 1 || volume.SizeGB > 1024 {
		return ProviderVolume{}, errors.New("volume size is outside local driver limits")
	}
	path := d.volumePath(volume.ID)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if errors.Is(err, os.ErrExist) {
		info, statErr := os.Stat(path)
		if statErr == nil && info.Size() == volume.SizeGB*1024*1024*1024 {
			return ProviderVolume{ID: volume.ID, Status: "available"}, nil
		}
		return ProviderVolume{}, errors.New("existing volume backing file has unexpected size")
	}
	if err != nil {
		return ProviderVolume{}, fmt.Errorf("create volume backing file: %w", err)
	}
	if err := file.Truncate(volume.SizeGB * 1024 * 1024 * 1024); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return ProviderVolume{}, fmt.Errorf("size volume backing file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return ProviderVolume{}, fmt.Errorf("close volume backing file: %w", err)
	}
	return ProviderVolume{ID: volume.ID, Status: "available"}, nil
}

func (d *LocalFileDriver) InspectVolume(ctx context.Context, id string) (ProviderVolume, error) {
	if err := ctx.Err(); err != nil {
		return ProviderVolume{}, err
	}
	_, err := os.Stat(d.volumePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return ProviderVolume{}, ErrProviderNotFound
	}
	if err != nil {
		return ProviderVolume{}, err
	}
	return ProviderVolume{ID: id, Status: "available"}, nil
}

func (d *LocalFileDriver) DeleteVolume(ctx context.Context, volume models.StorageVolume) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	err := os.Remove(d.volumePath(volume.ID))
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("delete volume backing file: %w", err)
	}
	return true, nil
}

func (d *LocalFileDriver) volumePath(id string) string {
	return filepath.Join(d.root, id+".img")
}
