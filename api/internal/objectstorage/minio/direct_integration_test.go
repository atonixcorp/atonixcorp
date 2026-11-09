package minio_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage"
	adapter "atonixcorp/api/internal/objectstorage/minio"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDirectUploadIntegration(t *testing.T) {
	if os.Getenv("MINIO_INTEGRATION_TEST") != "true" {
		t.Skip("isolated MinIO not configured")
	}
	p, err := adapter.FromEnvironment()
	if err != nil || p == nil {
		t.Fatalf("provider: %v", err)
	}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.StorageObject{}, &models.ObjectAuditEvent{}, &models.TenantMembership{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.User{ID: 1, Email: "admin@example.test", PlatformRole: models.PlatformRoleSuperAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	s := files.NewService(db, p)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, category := range []string{"uploads", "images", "documents", "backups", "logs", "datasets", "models", "users"} {
		t.Run(category, func(t *testing.T) {
			data := []byte("hello")
			contentType := "text/plain"
			if category == "images" {
				data = []byte("\x89PNG\r\n\x1a\nmore")
				contentType = "image/png"
			}
			sum := sha256.Sum256(data)
			request := files.UploadRequest{TenantID: 1, Category: category, Filename: "file.txt", ContentType: contentType,
				SizeBytes: int64(len(data)), Checksum: hex.EncodeToString(sum[:]), Subcategory: "test",
				Service: "test", Project: "test", Dataset: "test", Model: "test", Version: "v1"}
			auth, err := s.AuthorizeUpload(ctx, 1, request)
			if err != nil {
				t.Fatal(err)
			}
			stage := auth.Upload.Fields["key"]
			t.Cleanup(func() { _ = p.Delete(context.Background(), stage) })
			status, body := postObject(t, ctx, auth.Upload, append(append([]byte(nil), data...), 'x'))
			if status < 400 {
				t.Fatalf("oversized signed POST: %d %s", status, body)
			}
			status, body = postObject(t, ctx, auth.Upload, data)
			if status < 200 || status >= 300 {
				t.Fatalf("signed POST: %d %s", status, body)
			}
			obj, err := s.CompleteUpload(ctx, 1, auth.ObjectID)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = p.Delete(context.Background(), obj.ObjectKey) })
			if !strings.HasPrefix(obj.ObjectKey, category+"/") {
				t.Fatalf("unexpected key %s", obj.ObjectKey)
			}
			var metadata models.StorageObject
			if err := db.First(&metadata, "id = ?", obj.ID).Error; err != nil || metadata.Status != "active" || metadata.Checksum != request.Checksum {
				t.Fatalf("metadata: %+v %v", metadata, err)
			}
			_, reader, err := s.Download(ctx, 1, obj.ID)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || !bytes.Equal(got, data) {
				t.Fatalf("download: %q %v", got, err)
			}
			status, body = postObject(t, ctx, auth.Upload, data)
			if status < 200 || status >= 300 {
				t.Fatalf("replayed staging POST: %d %s", status, body)
			}
			if err := s.Delete(ctx, 1, obj.ID); err != nil {
				t.Fatal(err)
			}
		})
	}
	key := "staging/conditional-" + time.Now().UTC().Format("20060102150405.000000000")
	destination := key + "-copy"
	t.Cleanup(func() { _ = p.Delete(context.Background(), key); _ = p.Delete(context.Background(), destination) })
	if err := p.Put(ctx, key, strings.NewReader("before"), 6, "text/plain"); err != nil {
		t.Fatal(err)
	}
	info, err := p.Stat(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Put(ctx, key, strings.NewReader("after!"), 6, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := p.CopyIfMatch(ctx, key, destination, info.ETag); err == nil {
		t.Fatal("conditional copy accepted a changed staging object")
	}
	reader, err := p.GetIfMatch(ctx, key, info.ETag)
	if err == nil {
		_, err = io.ReadAll(reader)
		reader.Close()
	}
	if err == nil {
		t.Fatal("conditional read accepted a changed staging object")
	}
}

func postObject(t *testing.T, ctx context.Context, form objectstorage.UploadForm, data []byte) (int, string) {
	t.Helper()
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for k, v := range form.Fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", "file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, form.URL, &payload)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}
