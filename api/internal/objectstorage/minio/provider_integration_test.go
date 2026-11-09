package minio_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"
	adapter "atonixcorp/api/internal/objectstorage/minio"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMinIOIntegration(t *testing.T) {
	if os.Getenv("MINIO_INTEGRATION_TEST") != "true" {
		t.Skip("set MINIO_INTEGRATION_TEST=true with an isolated test bucket and application identity")
	}
	p, err := adapter.FromEnvironment()
	if err != nil || p == nil {
		t.Fatalf("provider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := p.Health(ctx); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.StorageObject{}, &models.ObjectAuditEvent{}, &models.TenantMembership{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	service := files.NewService(db, p)
	obj, err := service.Upload(ctx, 1, 1, "acceptance.txt", "text/plain", strings.NewReader("acceptance"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Delete(context.Background(), obj.ObjectKey) })
	var saved models.StorageObject
	if err := db.First(&saved, "id = ?", obj.ID).Error; err != nil || saved.Status != "active" {
		t.Fatalf("metadata: %+v %v", saved, err)
	}
	_, body, err := service.Download(ctx, 1, obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil || string(data) != "acceptance" {
		t.Fatalf("download: %q %v", data, err)
	}
	signed, err := service.Presign(ctx, 1, obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK || string(data) != "acceptance" {
		t.Fatalf("signed download: %d %q %v", resp.StatusCode, data, err)
	}
	if err := service.Delete(ctx, 1, obj.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Stat(ctx, obj.ObjectKey); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("object still present or stat failed: %v", err)
	}
	id := uuid.NewString()
	for _, category := range []string{"uploads", "images", "documents", "backups", "logs", "datasets", "models", "users"} {
		parts := objectstorage.KeyParts{Category: category, Tenant: "test", User: "test", ID: id,
			Filename: "test.txt", ImageCategory: "test", UserCategory: "test", Extension: "txt", DocumentType: "test", Service: "test",
			Project: "test", Dataset: "test", Model: "test", Version: id, Time: time.Now()}
		key, err := objectstorage.ObjectKey(parts)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = p.Delete(context.Background(), key) })
		if err := p.Put(ctx, key, strings.NewReader("test"), 4, "text/plain"); err != nil {
			t.Fatalf("%s put: %v", category, err)
		}
		info, err := p.Stat(ctx, key)
		if err != nil || info.Size != 4 {
			t.Fatalf("%s stat: %+v %v", category, info, err)
		}
		if exists, err := p.Exists(ctx, key); err != nil || !exists {
			t.Fatalf("%s exists: %v %v", category, exists, err)
		}
		r, err := p.Get(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil || !bytes.Equal(b, []byte("test")) {
			t.Fatalf("%s get: %q %v", category, b, err)
		}
		list, err := p.List(ctx, key)
		if err != nil || len(list) != 1 || list[0].Key != key {
			t.Fatalf("%s list: %+v %v", category, list, err)
		}
		copyKey := key + "-copy"
		t.Cleanup(func() { _ = p.Delete(context.Background(), copyKey) })
		if err := p.Copy(ctx, key, copyKey); err != nil {
			t.Fatalf("%s copy: %v", category, err)
		}
		if _, err := p.Stat(ctx, copyKey); err != nil {
			t.Fatalf("%s copied object: %v", category, err)
		}
		if err := p.Delete(ctx, copyKey); err != nil {
			t.Fatal(err)
		}
		if err := p.Delete(ctx, key); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Stat(ctx, key); !errors.Is(err, objectstorage.ErrNotFound) {
			t.Fatalf("%s deletion: %v", category, err)
		}
		if exists, err := p.Exists(ctx, key); err != nil || exists {
			t.Fatalf("%s deleted object exists: %v %v", category, exists, err)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := p.Health(cancelled); err == nil {
		t.Fatal("cancelled storage health should fail")
	}
}
