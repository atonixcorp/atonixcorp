package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"atonixcorp/cloud/hpc/jobs"

	"github.com/go-chi/chi/v5"
)

type JobStatusHandler struct {
	Service *jobs.Service
}

func NewJobStatusHandler(s *jobs.Service) *JobStatusHandler {
	return &JobStatusHandler{Service: s}
}

func (h *JobStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	job, err := h.Service.RefreshStatus(r.Context(), id)
	if err != nil {
		log.Printf("refresh HPC job status id=%s: %v", id, err)
		writeJobError(w, http.StatusNotFound, "JOB_STATUS_UNAVAILABLE", "The job status could not be retrieved.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
