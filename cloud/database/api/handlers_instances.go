package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"atonixcorp/cloud/database/core"

	"github.com/go-chi/chi/v5"
)

func (s *APIServer) ListEngines(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.controller.EngineCatalog())
}

func (s *APIServer) ListPlans(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.controller.PlanCatalog())
}

func (s *APIServer) ListInstances(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeProject(w, r) {
		return
	}
	instances, err := s.controller.ListInstances(
		chi.URLParam(r, "tenant_id"), chi.URLParam(r, "project_id"),
	)
	if err != nil {
		http.Error(w, "could not list databases", http.StatusInternalServerError)
		return
	}
	if instances == nil {
		instances = []core.DBInstance{}
	}
	writeJSON(w, http.StatusOK, instances)
}

func (s *APIServer) CreateInstance(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeProject(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req core.CreateInstanceRequest
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid database request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid database request", http.StatusBadRequest)
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(idempotencyKey) < 16 || len(idempotencyKey) > 128 {
		writeError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key must be between 16 and 128 characters.")
		return
	}
	req.TenantID = chi.URLParam(r, "tenant_id")
	req.ProjectID = chi.URLParam(r, "project_id")

	result, err := s.provisioning.CreateDatabase(req, idempotencyKey)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, core.ErrProviderUnavailable) {
			status = http.StatusServiceUnavailable
			writeError(w, status, "DATABASE_PROVISIONING_UNAVAILABLE", "Database provisioning is currently unavailable. Please contact your platform administrator.")
			return
		} else if errors.Is(err, core.ErrInvalidRequest) {
			status = http.StatusBadRequest
		} else if errors.Is(err, core.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", core.ErrIdempotencyConflict.Error())
			return
		}
		writeError(w, status, "DATABASE_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *APIServer) GetInstance(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeProject(w, r) {
		return
	}
	instance, err := s.provisioning.GetInstance(chi.URLParam(r, "instance_id"))
	if err != nil {
		http.Error(w, "database not found", http.StatusNotFound)
		return
	}
	if instance.TenantID != chi.URLParam(r, "tenant_id") ||
		instance.ProjectID != chi.URLParam(r, "project_id") {
		http.Error(w, "database not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, instance)
}

func (s *APIServer) GetOperation(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeProject(w, r) {
		return
	}
	operation, err := s.store.GetOperation(chi.URLParam(r, "operation_id"))
	if err != nil || operation.TenantID != chi.URLParam(r, "tenant_id") ||
		operation.ProjectID != chi.URLParam(r, "project_id") {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, operation)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "could not encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(payload, '\n')); err != nil {
		log.Printf("write database API response: %v", err)
	}
}
