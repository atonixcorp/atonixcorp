package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeProvider struct {
	data        map[string][]byte
	contentType string
	fail        bool
}

func (p *fakeProvider) Name() string   { return "fake" }
func (p *fakeProvider) Bucket() string { return "bucket" }
func (p *fakeProvider) Health(context.Context) error {
	if p.fail {
		return errors.New("down")
	}
	return nil
}
func (p *fakeProvider) Put(_ context.Context, k string, r io.Reader, _ int64, ct string) error {
	if p.fail {
		return errors.New("down")
	}
	b, err := io.ReadAll(r)
	if err == nil {
		p.data[k] = b
		p.contentType = ct
	}
	return err
}
func (p *fakeProvider) Get(_ context.Context, k string) (io.ReadCloser, error) {
	if p.fail {
		return nil, errors.New("down")
	}
	b, ok := p.data[k]
	if !ok {
		return nil, objectstorage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (p *fakeProvider) GetIfMatch(ctx context.Context, k, etag string) (io.ReadCloser, error) {
	if etag != "fake-etag" {
		return nil, errors.New("etag mismatch")
	}
	return p.Get(ctx, k)
}
func (p *fakeProvider) Delete(_ context.Context, k string) error {
	if p.fail {
		return errors.New("down")
	}
	delete(p.data, k)
	return nil
}
func (p *fakeProvider) Exists(_ context.Context, k string) (bool, error) {
	if p.fail {
		return false, errors.New("down")
	}
	_, ok := p.data[k]
	return ok, nil
}
func (p *fakeProvider) Stat(_ context.Context, k string) (objectstorage.Info, error) {
	if p.fail {
		return objectstorage.Info{}, errors.New("down")
	}
	b, ok := p.data[k]
	if !ok {
		return objectstorage.Info{}, objectstorage.ErrNotFound
	}
	return objectstorage.Info{Key: k, Size: int64(len(b)), ContentType: p.contentType, ETag: "fake-etag"}, nil
}
func (p *fakeProvider) List(context.Context, string) ([]objectstorage.Info, error) { return nil, nil }
func (p *fakeProvider) Copy(context.Context, string, string) error                 { return nil }
func (p *fakeProvider) CopyIfMatch(_ context.Context, source, destination, etag string) error {
	if etag != "fake-etag" {
		return errors.New("etag mismatch")
	}
	b, ok := p.data[source]
	if !ok {
		return objectstorage.ErrNotFound
	}
	p.data[destination] = append([]byte(nil), b...)
	return nil
}
func (p *fakeProvider) PresignGet(context.Context, string, time.Duration) (*url.URL, error) {
	return url.Parse("https://storage.example/test")
}
func (p *fakeProvider) PresignPut(context.Context, string, time.Duration) (*url.URL, error) {
	return nil, errors.New("not implemented")
}
func (p *fakeProvider) PresignPost(_ context.Context, key, contentType string, size int64, expires time.Time) (objectstorage.UploadForm, error) {
	if p.fail {
		return objectstorage.UploadForm{}, errors.New("down")
	}
	return objectstorage.UploadForm{URL: "https://storage.example", Fields: map[string]string{"key": key}}, nil
}

func TestFileLifecycleAndAuthorization(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.StorageObject{}, &models.ObjectAuditEvent{}, &models.TenantMembership{}); err != nil {
		t.Fatal(err)
	}
	for _, m := range []models.TenantMembership{
		{TenantID: 1, UserID: 1, Role: "owner"}, {TenantID: 1, UserID: 2, Role: "owner"},
		{TenantID: 1, UserID: 3, Role: "viewer"},
	} {
		if err := db.Create(&m).Error; err != nil {
			t.Fatal(err)
		}
	}
	p := &fakeProvider{data: map[string][]byte{}}
	s := NewService(db, p)
	ctx := context.Background()
	obj, err := s.Upload(ctx, 1, 1, "report.txt", "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if obj.Status != "active" || obj.Checksum != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" || !strings.HasPrefix(obj.ObjectKey, "uploads/1/1/") {
		t.Fatalf("unexpected metadata: %+v", obj)
	}
	_, body, err := s.Download(ctx, 1, obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(body)
	body.Close()
	if err != nil || string(b) != "hello" {
		t.Fatalf("download: %q, %v", b, err)
	}
	if _, _, err := s.Download(ctx, 2, obj.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user download: %v", err)
	}
	if _, err := s.Presign(ctx, 1, obj.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("tenant_id = ? AND user_id = ?", 1, 1).Delete(&models.TenantMembership{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Download(ctx, 1, obj.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked member download: %v", err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upload(ctx, 3, 1, "x.txt", "text/plain", strings.NewReader("hello")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("viewer upload: %v", err)
	}
	if _, err := s.Upload(ctx, 1, 1, "x.txt", "image/png", strings.NewReader("hello")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("type mismatch: %v", err)
	}
	if _, err := s.Upload(ctx, 1, 1, "x.txt", "text/plain", strings.NewReader(strings.Repeat("x", MaxUploadBytes+1))); !errors.Is(err, ErrInvalid) {
		t.Fatalf("size limit: %v", err)
	}
	if err := s.Delete(ctx, 2, obj.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user delete: %v", err)
	}
	if err := s.Delete(ctx, 1, obj.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Download(ctx, 1, obj.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted object download: %v", err)
	}
	if len(p.data) != 0 {
		t.Fatal("object not deleted")
	}
	var count int64
	if err := db.Model(&models.ObjectAuditEvent{}).Count(&count).Error; err != nil || count != 4 {
		t.Fatalf("audit count: %d %v", count, err)
	}
	p.fail = true
	if _, err := s.Upload(ctx, 1, 1, "x.txt", "text/plain", strings.NewReader("hello")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("outage: %v", err)
	}
	if err := s.Health(ctx); err == nil {
		t.Fatal("expected health failure")
	}
	if err := db.Where("status = ?", "pending").First(&models.StorageObject{}).Error; err != nil {
		t.Fatalf("pending recovery record missing: %v", err)
	}
}
