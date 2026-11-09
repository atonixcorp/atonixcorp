package datasets

import (
	"context"
	"errors"
	"io"
)

var ErrTenantAccessDenied = errors.New("dataset belongs to another tenant")

type DownloadService struct {
	storage StorageAdapter
	meta    MetadataRepository
}

func NewDownloadService(storage StorageAdapter, meta MetadataRepository) *DownloadService {
	return &DownloadService{
		storage: storage,
		meta:    meta,
	}
}

func (s *DownloadService) Download(ctx context.Context, datasetID string) (io.ReadCloser, *Dataset, error) {
	if s == nil || s.storage == nil || s.meta == nil {
		return nil, nil, errors.New("dataset download service is not configured")
	}
	d, err := s.meta.Get(ctx, datasetID)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.Download(ctx, d.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, d, nil
}

func (s *DownloadService) DownloadForTenant(ctx context.Context, datasetID, tenantID string) (io.ReadCloser, *Dataset, error) {
	if s == nil || s.storage == nil || s.meta == nil {
		return nil, nil, errors.New("dataset download service is not configured")
	}
	dataset, err := s.meta.Get(ctx, datasetID)
	if err != nil {
		return nil, nil, err
	}
	if dataset.TenantID != tenantID {
		return nil, nil, ErrTenantAccessDenied
	}
	reader, err := s.storage.Download(ctx, dataset.StoragePath)
	if err != nil {
		return nil, nil, err
	}
	return reader, dataset, nil
}
