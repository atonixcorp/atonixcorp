package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"atonixcorp/cloud/hpc/api/middleware"
	"atonixcorp/cloud/hpc/jobs"

	"github.com/go-chi/chi/v5"
)

type JobLogsHandler struct {
	Logs *jobs.LogService
}

func NewJobLogsHandler(l *jobs.LogService) *JobLogsHandler {
	return &JobLogsHandler{Logs: l}
}

func (h *JobLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	logs, err := h.Logs.ListForTenant(r.Context(), id, auth.TenantID)
	if err != nil {
		log.Printf("list HPC job logs id=%s: %v", id, err)
		writeJobError(w, http.StatusNotFound, "JOB_LOGS_UNAVAILABLE", "Job logs could not be retrieved.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
