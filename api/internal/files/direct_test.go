package files

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDirectUploadSecurity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.StorageObject{}, &models.ObjectAuditEvent{}, &models.TenantMembership{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{ID: 1, Email: "owner@example.test", PlatformRole: models.PlatformRoleUser}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 2, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{data: map[string][]byte{}}
	s := NewService(db, p)
	ctx := context.Background()
	req := UploadRequest{TenantID: 1, Category: "uploads", Filename: "file.txt", ContentType: "text/plain", SizeBytes: 5, Checksum: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"}
	tooLarge := req
	tooLarge.SizeBytes = MaxDirectUploadBytes + 1
	if _, err := s.AuthorizeUpload(ctx, 1, tooLarge); !errors.Is(err, ErrInvalid) {
		t.Fatalf("size limit: %v", err)
	}
	atLimit := req
	atLimit.SizeBytes = MaxDirectUploadBytes
	if _, err := s.AuthorizeUpload(ctx, 1, atLimit); err != nil {
		t.Fatalf("maximum upload size rejected: %v", err)
	}
	adminOnly := req
	adminOnly.Category, adminOnly.Service = "logs", "api"
	if _, err := s.AuthorizeUpload(ctx, 1, adminOnly); !errors.Is(err, ErrNotFound) {
		t.Fatalf("log access: %v", err)
	}
	auth, err := s.AuthorizeUpload(ctx, 1, req)
	if err != nil {
		t.Fatal(err)
	}
	stage := auth.Upload.Fields["key"]
	if _, _, err := s.Download(ctx, 1, auth.ObjectID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pending download: %v", err)
	}
	if err := p.Put(ctx, stage, strings.NewReader("wrong"), 5, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteUpload(ctx, 1, auth.ObjectID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("checksum mismatch: %v", err)
	}
	if err := p.Put(ctx, stage, strings.NewReader("hello"), 5, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteUpload(ctx, 2, auth.ObjectID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other owner: %v", err)
	}
	if err := db.Where("tenant_id = ? AND user_id = ?", 1, 1).Delete(&models.TenantMembership{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteUpload(ctx, 1, auth.ObjectID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked member: %v", err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	obj, err := s.CompleteUpload(ctx, 1, auth.ObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Status != "active" || string(p.data[obj.ObjectKey]) != "hello" {
		t.Fatalf("final object: %+v", obj)
	}
	listed, err := s.List(ctx, 1, 1, 50, 0)
	if err != nil || len(listed) != 1 || listed[0].ID != obj.ID {
		t.Fatalf("owner listing: %+v %v", listed, err)
	}
	if other, err := s.List(ctx, 2, 1, 50, 0); err != nil || len(other) != 0 {
		t.Fatalf("other member listing: %+v %v", other, err)
	}
	p.data[stage] = []byte("again")
	if string(p.data[obj.ObjectKey]) != "hello" {
		t.Fatal("staging replay changed final object")
	}
	if err := db.Model(&models.StorageObject{}).Where("id = ?", obj.ID).Update("upload_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	s.cleanupExpired(ctx)
	if _, ok := p.data[stage]; ok {
		t.Fatal("replayed staging object was not removed")
	}
	var saved models.StorageObject
	if err := db.First(&saved, "id = ?", obj.ID).Error; err != nil || saved.Status != "active" || saved.StagingKey != "" {
		t.Fatalf("cleanup: %+v %v", saved, err)
	}
	auth, err = s.AuthorizeUpload(ctx, 1, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.StorageObject{}).Where("id = ?", auth.ObjectID).Update("upload_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteUpload(ctx, 1, auth.ObjectID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expired completion: %v", err)
	}
	s.cleanupExpired(ctx)
	saved = models.StorageObject{}
	if err := db.First(&saved, "id = ?", auth.ObjectID).Error; err != nil || saved.Status != "expired" {
		t.Fatalf("expired cleanup: %+v %v", saved, err)
	}
}
