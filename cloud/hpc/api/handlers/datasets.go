package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"

	"atonixcorp/cloud/hpc/api/middleware"
	"atonixcorp/cloud/hpc/datasets"

	"github.com/go-chi/chi/v5"
)

const maxDatasetUploadBytes int64 = 1 << 30

type DatasetHandlers struct {
	Metadata datasets.MetadataRepository
	Upload   *datasets.UploadService
	Download *datasets.DownloadService
}

func (h *DatasetHandlers) List(w http.ResponseWriter, r *http.Request) {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := h.Metadata.ListByTenant(r.Context(), auth.TenantID)
	if err != nil {
		log.Printf("list HPC datasets tenant=%s: %v", auth.TenantID, err)
		http.Error(w, "failed to list datasets", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *DatasetHandlers) UploadDataset(w http.ResponseWriter, r *http.Request) {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDatasetUploadBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "invalid or oversized multipart upload", http.StatusBadRequest)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "multipart field 'file' is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSpace(header.Filename)
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var labels map[string]string
	if rawLabels := strings.TrimSpace(r.FormValue("labels")); rawLabels != "" {
		if err := json.Unmarshal([]byte(rawLabels), &labels); err != nil {
			http.Error(w, "labels must be a JSON object of strings", http.StatusBadRequest)
			return
		}
	}

	item, err := h.Upload.Upload(r.Context(), datasets.UploadRequest{
		TenantID:    auth.TenantID,
		Name:        name,
		ContentType: contentType,
		Labels:      labels,
		Reader:      file,
		SizeBytes:   header.Size,
	})
	if err != nil {
		log.Printf("upload HPC dataset tenant=%s: %v", auth.TenantID, err)
		http.Error(w, "dataset upload failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *DatasetHandlers) DownloadDataset(w http.ResponseWriter, r *http.Request) {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing dataset ID", http.StatusBadRequest)
		return
	}
	reader, item, err := h.Download.DownloadForTenant(r.Context(), id, auth.TenantID)
	if err != nil {
		if errors.Is(err, datasets.ErrTenantAccessDenied) {
			http.Error(w, "dataset access denied", http.StatusForbidden)
			return
		}
		log.Printf("download HPC dataset id=%s tenant=%s: %v", id, auth.TenantID, err)
		http.Error(w, "dataset unavailable", http.StatusNotFound)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", item.ContentType)
	if value := mime.FormatMediaType("attachment", map[string]string{"filename": item.Name}); value != "" {
		w.Header().Set("Content-Disposition", value)
	}
	w.Header().Set("Content-Length", fmt.Sprint(item.SizeBytes))
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("stream HPC dataset id=%s: %v", id, err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write HPC JSON response: %v", err)
	}
}
