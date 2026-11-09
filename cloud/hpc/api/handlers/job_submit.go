package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"atonixcorp/cloud/hpc/jobs"
)

type JobSubmitHandler struct {
	Service *jobs.Service
}

func NewJobSubmitHandler(s *jobs.Service) *JobSubmitHandler {
	return &JobSubmitHandler{Service: s}
}

func (h *JobSubmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req jobs.CreateJobRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	job, err := h.Service.CreateAndSubmit(r.Context(), req)
	if err != nil {
		if errors.Is(err, jobs.ErrInvalidJobSpec) {
			writeJobError(w, http.StatusBadRequest, "INVALID_JOB", "Invalid job specification.")
			return
		}
		if errors.Is(err, jobs.ErrExecutionProfileUnavailable) {
			writeJobError(w, http.StatusServiceUnavailable, "EXECUTION_PROFILE_UNAVAILABLE", "The execution profile is not available.")
			return
		}
		log.Printf("submit HPC job: %v", err)
		writeJobError(w, http.StatusInternalServerError, "JOB_SUBMISSION_FAILED", "The job could not be submitted.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func writeJobError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "error": message})
}
