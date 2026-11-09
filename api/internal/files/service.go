package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/authorization"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const MaxUploadBytes = 10 << 20

var (
	ErrInvalid      = errors.New("invalid file")
	ErrNotFound     = errors.New("file not found")
	ErrUnavailable  = errors.New("object storage unavailable")
	filenamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,254}$`)
)

type Service struct {
	db       *gorm.DB
	provider objectstorage.Provider
}

func NewService(db *gorm.DB, provider objectstorage.Provider) *Service {
	return &Service{db: db, provider: provider}
}

func (s *Service) Health(ctx context.Context) error {
	if s.provider == nil {
		return ErrUnavailable
	}
	return s.provider.Health(ctx)
}

func (s *Service) List(ctx context.Context, userID, tenantID uint, limit, offset int) ([]models.StorageObject, error) {
	if userID == 0 || tenantID == 0 || limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrInvalid
	}
	if err := s.authorize(ctx, userID, tenantID, false); err != nil {
		return nil, err
	}
	objects := make([]models.StorageObject, 0)
	err := s.db.WithContext(ctx).Where("owner_id = ? AND tenant_id = ? AND status = ?", userID, tenantID, "active").
		Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&objects).Error
	return objects, err
}

func (s *Service) authorize(ctx context.Context, userID, tenantID uint, write bool) error {
	var membership models.TenantMembership
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if write && membership.Role != "owner" && membership.Role != "admin" && membership.Role != "deployer" {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Upload(ctx context.Context, userID, tenantID uint, filename, declaredType string, input io.Reader) (*models.StorageObject, error) {
	if userID == 0 || tenantID == 0 || !filenamePattern.MatchString(filename) || filename == "." || filename == ".." {
		return nil, ErrInvalid
	}
	if err := s.authorize(ctx, userID, tenantID, true); err != nil {
		return nil, err
	}
	if s.provider == nil {
		return nil, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(input, MaxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if len(data) == 0 || len(data) > MaxUploadBytes {
		return nil, ErrInvalid
	}
	detected := strings.Split(http.DetectContentType(data), ";")[0]
	declaredType = strings.TrimSpace(strings.Split(declaredType, ";")[0])
	if !allowedContentType("uploads", detected) || detected != declaredType {
		return nil, ErrInvalid
	}
	checksum := sha256.Sum256(data)
	return s.persistUpload(ctx, userID, tenantID, filename, detected, int64(len(data)), hex.EncodeToString(checksum[:]), bytes.NewReader(data), nil, userID)
}

func (s *Service) UploadLegacy(ctx context.Context, actorID, userID, tenantID, legacyID uint, filename, declaredType string, input io.ReadSeeker, size int64) (*models.StorageObject, error) {
	admin, err := authorization.IsPlatformSuperAdmin(ctx, s.db, actorID)
	if err != nil {
		return nil, err
	}
	if !admin || legacyID == 0 {
		return nil, ErrNotFound
	}
	if userID == 0 || tenantID == 0 || !filenamePattern.MatchString(filename) || size <= 0 {
		return nil, ErrInvalid
	}
	if err := s.authorize(ctx, userID, tenantID, true); err != nil {
		return nil, err
	}
	if s.provider == nil {
		return nil, ErrUnavailable
	}
	first := make([]byte, 512)
	n, err := io.ReadFull(input, first)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("read legacy file: %w", err)
	}
	detected := strings.Split(http.DetectContentType(first[:n]), ";")[0]
	declaredType = strings.TrimSpace(strings.Split(declaredType, ";")[0])
	if detected != declaredType || !allowedContentType("uploads", detected) {
		return nil, ErrInvalid
	}
	hash := sha256.New()
	_, _ = hash.Write(first[:n])
	remaining, err := io.Copy(hash, io.LimitReader(input, size+1))
	if err != nil {
		return nil, fmt.Errorf("hash legacy file: %w", err)
	}
	if int64(n)+remaining != size {
		return nil, ErrInvalid
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind legacy file: %w", err)
	}
	return s.persistUpload(ctx, userID, tenantID, filename, detected, size, hex.EncodeToString(hash.Sum(nil)), input, &legacyID, actorID)
}

func (s *Service) persistUpload(ctx context.Context, userID, tenantID uint, filename, detected string, size int64, checksum string, data io.Reader, legacyID *uint, actorID uint) (*models.StorageObject, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	key, err := objectstorage.ObjectKey(objectstorage.KeyParts{
		Category: "uploads", Tenant: strconv.FormatUint(uint64(tenantID), 10),
		User: strconv.FormatUint(uint64(userID), 10), ID: id.String(), Filename: filename,
	})
	if err != nil {
		return nil, ErrInvalid
	}
	obj := &models.StorageObject{
		ID: id.String(), Provider: s.provider.Name(), Bucket: s.provider.Bucket(),
		ObjectKey: key,
		OwnerID:   userID, TenantID: tenantID, OriginalFilename: filename, ContentType: detected,
		SizeBytes: size, Checksum: checksum,
		Category: "uploads", Visibility: "private", Status: "pending",
		LegacyFileID: legacyID,
	}
	if err := s.db.WithContext(ctx).Create(obj).Error; err != nil {
		return nil, err
	}
	uploadedHash := sha256.New()
	if err := s.provider.Put(ctx, obj.ObjectKey, io.TeeReader(data, uploadedHash), obj.SizeBytes, obj.ContentType); err != nil {
		// Retain the pending row for operator reconciliation, including ambiguous uploads.
		return nil, fmt.Errorf("%w: upload: %v", ErrUnavailable, err)
	}
	if hex.EncodeToString(uploadedHash.Sum(nil)) != obj.Checksum {
		return nil, fmt.Errorf("uploaded content changed during transfer")
	}
	info, err := s.provider.Stat(ctx, obj.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("%w: verify upload: %v", ErrUnavailable, err)
	}
	if info.Size != obj.SizeBytes || info.ContentType != obj.ContentType {
		return nil, fmt.Errorf("%w: uploaded object metadata mismatch", ErrUnavailable)
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(obj).Update("status", "active").Error; err != nil {
			return err
		}
		action := "upload"
		if legacyID != nil {
			action = "legacy_import"
		}
		return tx.Create(&models.ObjectAuditEvent{ObjectID: obj.ID, ActorID: actorID, TenantID: tenantID, Action: action}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("activate object metadata: %w", err)
	}
	obj.Status = "active"
	return obj, nil
}

func (s *Service) getAuthorized(ctx context.Context, userID uint, id string, write bool) (*models.StorageObject, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var obj models.StorageObject
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ? AND status = ?", id, userID, "active").First(&obj).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, userID, obj.TenantID, write); err != nil {
		return nil, err
	}
	if s.provider == nil || obj.Provider != s.provider.Name() || obj.Bucket != s.provider.Bucket() {
		return nil, ErrUnavailable
	}
	return &obj, nil
}

func (s *Service) audit(ctx context.Context, obj *models.StorageObject, userID uint, action string) error {
	return s.db.WithContext(ctx).Create(&models.ObjectAuditEvent{
		ObjectID: obj.ID, ActorID: userID, TenantID: obj.TenantID, Action: action,
	}).Error
}

func (s *Service) Download(ctx context.Context, userID uint, id string) (*models.StorageObject, io.ReadCloser, error) {
	obj, err := s.getAuthorized(ctx, userID, id, false)
	if err != nil {
		return nil, nil, err
	}
	if err := s.audit(ctx, obj, userID, "download"); err != nil {
		return nil, nil, err
	}
	body, err := s.provider.Get(ctx, obj.ObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: download: %v", ErrUnavailable, err)
	}
	return obj, body, nil
}

func (s *Service) Presign(ctx context.Context, userID uint, id string) (string, error) {
	obj, err := s.getAuthorized(ctx, userID, id, false)
	if err != nil {
		return "", err
	}
	if err := s.audit(ctx, obj, userID, "presign_get"); err != nil {
		return "", err
	}
	u, err := s.provider.PresignGet(ctx, obj.ObjectKey, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("%w: presign: %v", ErrUnavailable, err)
	}
	return u.String(), nil
}

func (s *Service) Delete(ctx context.Context, userID uint, id string) error {
	obj, err := s.getAuthorized(ctx, userID, id, true)
	if err != nil {
		return err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.StorageObject{}).Where("id = ? AND status = ?", id, "active").Update("status", "deleting")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		return tx.Create(&models.ObjectAuditEvent{ObjectID: id, ActorID: userID, TenantID: obj.TenantID, Action: "delete"}).Error
	})
	if err != nil {
		return err
	}
	if err := s.provider.Delete(ctx, obj.ObjectKey); err != nil {
		// Leave deletion pending because the provider may have completed it.
		return fmt.Errorf("%w: delete: %v", ErrUnavailable, err)
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(obj).Updates(map[string]interface{}{"status": "deleted", "deleted_at": now}).Error; err != nil {
		log.Printf("object %s was deleted but metadata update failed: %v", id, err)
		return err
	}
	return nil
}
