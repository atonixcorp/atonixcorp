package datasets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

type UploadService struct {
	storage StorageAdapter
	meta    MetadataRepository
}

func NewUploadService(storage StorageAdapter, meta MetadataRepository) *UploadService {
	return &UploadService{
		storage: storage,
		meta:    meta,
	}
}

type UploadRequest struct {
	TenantID    string
	Name        string
	ContentType string
	Labels      map[string]string
	Reader      io.Reader
	SizeBytes   int64
}

func (s *UploadService) Upload(ctx context.Context, req UploadRequest) (*Dataset, error) {
	if s == nil || s.storage == nil || s.meta == nil {
		return nil, errors.New("dataset upload service is not configured")
	}
	if req.TenantID == "" || req.Name == "" || req.Reader == nil || req.SizeBytes < 0 {
		return nil, errors.New("tenant ID, dataset name, content, and non-negative size are required")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate dataset ID: %w", err)
	}
	path := fmt.Sprintf("%s/%s", req.TenantID, id.String())

	if err := s.storage.Upload(ctx, path, req.Reader, req.SizeBytes, req.ContentType); err != nil {
		return nil, fmt.Errorf("upload dataset object: %w", err)
	}

	d := &Dataset{
		ID:          id.String(),
		TenantID:    req.TenantID,
		Name:        req.Name,
		SizeBytes:   req.SizeBytes,
		ContentType: req.ContentType,
		StoragePath: path,
		Labels:      req.Labels,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.meta.Save(ctx, d); err != nil {
		if storage, ok := s.storage.(interface {
			Delete(context.Context, string) error
		}); ok {
			if cleanupErr := storage.Delete(ctx, path); cleanupErr != nil {
				return nil, fmt.Errorf("save dataset metadata: %w (also failed to remove uploaded object: %v)", err, cleanupErr)
			}
		}
		return nil, fmt.Errorf("save dataset metadata: %w", err)
	}

	return d, nil
}
