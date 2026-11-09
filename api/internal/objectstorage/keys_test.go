package objectstorage

import (
	"strings"
	"testing"
	"time"
)

func TestObjectKeys(t *testing.T) {
	now := time.Date(2026, 11, 9, 8, 7, 6, 0, time.UTC)
	base := KeyParts{Tenant: "tenant-1", User: "user-1", ID: "unique-id", Filename: "file.pdf",
		ImageCategory: "avatars", UserCategory: "avatars", Extension: "jpg", DocumentType: "reports", Service: "api", Project: "project",
		Dataset: "data", Model: "model", Version: "v1", Time: now}
	expected := map[string]string{
		"uploads":   "uploads/tenant-1/user-1/unique-id/file.pdf",
		"images":    "images/avatars/tenant-1/unique-id.jpg",
		"documents": "documents/tenant-1/reports/2026/unique-id/file.pdf",
		"backups":   "backups/api/2026/11/20261109T080706.000000000Z/file.pdf",
		"logs":      "logs/api/2026/11/09/file.pdf",
		"datasets":  "datasets/project/data/v1/file.pdf",
		"models":    "models/project/model/v1/file.pdf",
		"users":     "users/user-1/avatars/unique-id/file.pdf",
	}
	for category, want := range expected {
		p := base
		p.Category = category
		got, err := ObjectKey(p)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", category, got, err, want)
		}
		p.Filename = "../escape"
		p.Extension = "../escape"
		if _, err := ObjectKey(p); err == nil {
			t.Errorf("%s accepted unsafe filename", category)
		}
	}
	base.Category = "unknown"
	if _, err := ObjectKey(base); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatal("unknown category accepted")
	}
}
