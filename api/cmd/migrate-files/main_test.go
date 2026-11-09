package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testProvider struct{ objects map[string][]byte }

func (p *testProvider) Name() string                 { return "test" }
func (p *testProvider) Bucket() string               { return "test" }
func (p *testProvider) Health(context.Context) error { return nil }
func (p *testProvider) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	p.objects[key] = b
	return err
}
func (p *testProvider) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(p.objects[key])), nil
}
func (p *testProvider) GetIfMatch(ctx context.Context, key, _ string) (io.ReadCloser, error) {
	return p.Get(ctx, key)
}
func (p *testProvider) Delete(_ context.Context, key string) error {
	delete(p.objects, key)
	return nil
}
func (p *testProvider) Exists(_ context.Context, key string) (bool, error) {
	_, ok := p.objects[key]
	return ok, nil
}
func (p *testProvider) Stat(_ context.Context, key string) (objectstorage.Info, error) {
	return objectstorage.Info{Size: int64(len(p.objects[key])), ContentType: "text/plain"}, nil
}
func (p *testProvider) List(context.Context, string) ([]objectstorage.Info, error) { return nil, nil }
func (p *testProvider) Copy(context.Context, string, string) error                 { return nil }
func (p *testProvider) CopyIfMatch(context.Context, string, string, string) error  { return nil }
func (p *testProvider) PresignGet(context.Context, string, time.Duration) (*url.URL, error) {
	return nil, nil
}
func (p *testProvider) PresignPut(context.Context, string, time.Duration) (*url.URL, error) {
	return nil, nil
}
func (p *testProvider) PresignPost(context.Context, string, string, int64, time.Time) (objectstorage.UploadForm, error) {
	return objectstorage.UploadForm{}, nil
}

func TestMigrationMappingAndPathSafety(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.File{}, &models.TenantMembership{}, &models.StorageObject{}, &models.ObjectAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []models.User{{ID: 1, Email: "owner@test", PlatformRole: models.PlatformRoleUser}, {ID: 2, Email: "admin@test", PlatformRole: models.PlatformRoleSuperAdmin}} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	path := t.TempDir()
	if err := os.Mkdir(filepath.Join(path, "uploads"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "uploads", "file.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := db.Create(&models.File{ID: 1, FileName: "file.txt", FilePath: "uploads/file.txt", Size: 5, MimeType: "text/plain"}).Error; err != nil {
		t.Fatal(err)
	}
	p := &testProvider{objects: map[string][]byte{}}
	s := files.NewService(db, p)
	ctx := context.Background()
	if err := migrateOne(ctx, db, s, root, 1, 1, 1, 1); !errors.Is(err, files.ErrNotFound) {
		t.Fatalf("non-admin migration: %v", err)
	}
	if err := migrateOne(ctx, db, s, root, 2, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := migrateOne(ctx, db, s, root, 2, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := migrateOne(ctx, db, s, root, 2, 1, 2, 1); err == nil {
		t.Fatal("conflicting ownership accepted")
	}
	var obj models.StorageObject
	if err := db.First(&obj, "legacy_file_id = ?", 1).Error; err != nil || string(p.objects[obj.ObjectKey]) != "hello" {
		t.Fatalf("import: %+v %v", obj, err)
	}
	var audit models.ObjectAuditEvent
	if err := db.First(&audit, "object_id = ?", obj.ID).Error; err != nil || audit.ActorID != 2 || audit.Action != "legacy_import" {
		t.Fatalf("audit: %+v %v", audit, err)
	}
	if _, err := os.Stat(filepath.Join(path, "uploads", "file.txt")); err != nil {
		t.Fatal("source removed")
	}
	if err := db.Create(&models.File{ID: 2, FileName: "file.txt", FilePath: "../file.txt", Size: 5, MimeType: "text/plain"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateOne(ctx, db, s, root, 2, 2, 1, 1); err == nil {
		t.Fatal("path traversal accepted")
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(path, "uploads", "linked.txt")); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.File{ID: 3, FileName: "linked.txt", FilePath: "uploads/linked.txt", Size: 5, MimeType: "text/plain"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateOne(ctx, db, s, root, 2, 3, 1, 1); err == nil {
		t.Fatal("symlink escaping legacy root accepted")
	}
}
