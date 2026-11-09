package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/authorization"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const MaxDirectUploadBytes = 100 << 20
const directUploadLifetime = 15 * time.Minute

type UploadRequest struct {
	TenantID    uint   `json:"tenant_id"`
	Category    string `json:"category"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Checksum    string `json:"checksum"`
	Subcategory string `json:"subcategory"`
	Service     string `json:"service"`
	Project     string `json:"project"`
	Dataset     string `json:"dataset"`
	Model       string `json:"model"`
	Version     string `json:"version"`
}

type UploadAuthorization struct {
	ObjectID  string                   `json:"object_id"`
	ExpiresAt time.Time                `json:"expires_at"`
	Upload    objectstorage.UploadForm `json:"upload"`
}

func allowedContentType(category, contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	case "application/pdf", "text/plain", "application/zip", "application/x-gzip", "application/octet-stream":
		return category != "images"
	}
	return false
}

func (s *Service) AuthorizeUpload(ctx context.Context, userID uint, req UploadRequest) (*UploadAuthorization, error) {
	if userID == 0 || req.TenantID == 0 || !filenamePattern.MatchString(req.Filename) ||
		req.SizeBytes < 1 || req.SizeBytes > MaxDirectUploadBytes ||
		!allowedContentType(req.Category, req.ContentType) || len(req.Checksum) != sha256.Size*2 {
		return nil, ErrInvalid
	}
	checksum, err := hex.DecodeString(req.Checksum)
	if err != nil || len(checksum) != sha256.Size || req.Checksum != strings.ToLower(req.Checksum) {
		return nil, ErrInvalid
	}
	if (req.Category == "datasets" || req.Category == "models") && !filenamePattern.MatchString(req.Project) {
		return nil, ErrInvalid
	}
	if err := s.authorize(ctx, userID, req.TenantID, true); err != nil {
		return nil, err
	}
	if req.Category == "backups" || req.Category == "logs" {
		admin, err := authorization.IsPlatformSuperAdmin(ctx, s.db, userID)
		if err != nil {
			return nil, err
		}
		if !admin {
			return nil, ErrNotFound
		}
	}
	if s.provider == nil {
		return nil, ErrUnavailable
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	tenant := strconv.FormatUint(uint64(req.TenantID), 10)
	user := strconv.FormatUint(uint64(userID), 10)
	// Names used in project-like prefixes are owner-scoped labels, not project grants.
	project := "t" + tenant + "-u" + user + "-" + req.Project
	filename := req.Filename
	if req.Category == "backups" || req.Category == "logs" || req.Category == "datasets" || req.Category == "models" {
		filename = id.String() + "-" + filename
	}
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp"}[req.ContentType]
	key, err := objectstorage.ObjectKey(objectstorage.KeyParts{
		Category: req.Category, Tenant: tenant, User: user, ID: id.String(), Filename: filename,
		ImageCategory: req.Subcategory, UserCategory: req.Subcategory, DocumentType: req.Subcategory,
		Extension: extension, Service: req.Service, Project: project, Dataset: req.Dataset,
		Model: req.Model, Version: req.Version, Time: time.Now().UTC(),
	})
	if err != nil || len(key) > 1024 {
		return nil, ErrInvalid
	}
	expires := time.Now().UTC().Add(directUploadLifetime)
	obj := &models.StorageObject{
		ID: id.String(), Provider: s.provider.Name(), Bucket: s.provider.Bucket(), ObjectKey: key,
		OwnerID: userID, TenantID: req.TenantID, OriginalFilename: req.Filename,
		ContentType: req.ContentType, SizeBytes: req.SizeBytes, Checksum: req.Checksum,
		Category: req.Category, Visibility: "private", Status: "pending",
		StagingKey: "staging/" + id.String(), UploadExpiresAt: &expires,
	}
	form, err := s.provider.PresignPost(ctx, obj.StagingKey, obj.ContentType, obj.SizeBytes, expires)
	if err != nil {
		return nil, fmt.Errorf("%w: authorize upload: %v", ErrUnavailable, err)
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(obj).Error; err != nil {
			return err
		}
		return tx.Create(&models.ObjectAuditEvent{ObjectID: obj.ID, ActorID: userID, TenantID: obj.TenantID, Action: "upload_authorize"}).Error
	}); err != nil {
		return nil, err
	}
	return &UploadAuthorization{ObjectID: obj.ID, ExpiresAt: expires, Upload: form}, nil
}

func (s *Service) CompleteUpload(ctx context.Context, userID uint, id string) (*models.StorageObject, error) {
	if _, err := uuid.Parse(id); err != nil || userID == 0 {
		return nil, ErrNotFound
	}
	var obj models.StorageObject
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ? AND status = ?", id, userID, "pending").First(&obj).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if obj.StagingKey == "" || obj.UploadExpiresAt == nil || time.Now().After(*obj.UploadExpiresAt) {
		return nil, ErrInvalid
	}
	if err := s.authorize(ctx, userID, obj.TenantID, true); err != nil {
		return nil, err
	}
	if obj.Category == "backups" || obj.Category == "logs" {
		admin, err := authorization.IsPlatformSuperAdmin(ctx, s.db, userID)
		if err != nil {
			return nil, err
		}
		if !admin {
			return nil, ErrNotFound
		}
	}
	if s.provider == nil || s.provider.Name() != obj.Provider || s.provider.Bucket() != obj.Bucket {
		return nil, ErrUnavailable
	}
	result := s.db.WithContext(ctx).Model(&models.StorageObject{}).Where("id = ? AND status = ?", id, "pending").Update("status", "verifying")
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrNotFound
	}
	activated := false
	defer func() {
		if !activated {
			recovery, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.db.WithContext(recovery).Model(&models.StorageObject{}).Where("id = ? AND status = ?", id, "verifying").Update("status", "pending").Error; err != nil {
				log.Printf("reset verification state for object %s: %v", id, err)
			}
		}
	}()
	info, err := s.provider.Stat(ctx, obj.StagingKey)
	if errors.Is(err, objectstorage.ErrNotFound) {
		return nil, ErrInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("%w: stat upload: %v", ErrUnavailable, err)
	}
	if info.Size != obj.SizeBytes || info.ContentType != obj.ContentType || info.ETag == "" {
		return nil, ErrInvalid
	}
	body, err := s.provider.GetIfMatch(ctx, obj.StagingKey, info.ETag)
	if err != nil {
		return nil, fmt.Errorf("%w: read upload: %v", ErrUnavailable, err)
	}
	hash := sha256.New()
	first := make([]byte, 512)
	n, readErr := io.ReadFull(body, first)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		body.Close()
		return nil, fmt.Errorf("%w: read upload: %v", ErrUnavailable, readErr)
	}
	_, _ = hash.Write(first[:n])
	size, err := io.Copy(hash, io.LimitReader(body, obj.SizeBytes+1))
	closeErr := body.Close()
	if err != nil || closeErr != nil {
		return nil, fmt.Errorf("%w: read upload: %v %v", ErrUnavailable, err, closeErr)
	}
	detected := strings.Split(http.DetectContentType(first[:n]), ";")[0]
	if int64(n)+size != obj.SizeBytes || detected != obj.ContentType || hex.EncodeToString(hash.Sum(nil)) != obj.Checksum {
		return nil, ErrInvalid
	}
	if err := s.provider.CopyIfMatch(ctx, obj.StagingKey, obj.ObjectKey, info.ETag); err != nil {
		return nil, fmt.Errorf("%w: finalize upload: %v", ErrUnavailable, err)
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.StorageObject{}).Where("id = ? AND status = ?", id, "verifying").Update("status", "active")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		return tx.Create(&models.ObjectAuditEvent{ObjectID: obj.ID, ActorID: userID, TenantID: obj.TenantID, Action: "upload"}).Error
	})
	if err != nil {
		return nil, err
	}
	activated = true
	obj.Status = "active"
	if err := s.provider.Delete(ctx, obj.StagingKey); err != nil {
		log.Printf("remove staging object %s: %v", obj.ID, err)
	}
	return &obj, nil
}
