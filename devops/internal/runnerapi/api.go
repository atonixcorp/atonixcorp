package runnerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Pipeline struct {
	ID          string   `json:"id"`
	Project     string   `json:"project"`
	Ref         string   `json:"ref"`
	Status      string   `json:"status"`
	StartedAt   string   `json:"started_at"`
	Trigger     string   `json:"trigger"`
	Jobs        []Job    `json:"jobs"`
	ArtifactURL string   `json:"artifact_url,omitempty"`
	Labels      []string `json:"labels"`
}

type Job struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Stage       string `json:"stage"`
	Status      string `json:"status"`
	Executor    string `json:"executor"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	LogEndpoint string `json:"log_endpoint"`
}

type RunnerSummary struct {
	Capacity        int    `json:"capacity"`
	BusyExecutors   int    `json:"busy_executors"`
	QueuedJobs      int    `json:"queued_jobs"`
	LiveUpdatesMode string `json:"live_updates_mode"`
}

type createPipelineRequest struct {
	Project  string `json:"project"`
	Ref      string `json:"ref"`
	Trigger  string `json:"trigger"`
	Workflow string `json:"workflow"`
}

type jobRunner interface {
	Run(ctx context.Context, workflow string) (string, error)
}

type commandRunner struct {
	workspace string
}

func (runner commandRunner) Run(ctx context.Context, workflow string) (string, error) {
	if workflow != "api-test" {
		return "", errors.New("unsupported workflow")
	}
	command := exec.CommandContext(ctx, "go", "test", "./...")
	command.Dir = runner.workspace
	output, err := command.CombinedOutput()
	return string(output), err
}

type service struct {
	mu        sync.RWMutex
	pipelines []Pipeline
	logs      map[string]string
	runner    jobRunner
}

type agentProxy struct {
	baseURL string
	token   string
	client  *http.Client
}

func Register(mux *http.ServeMux) {
	workspace := os.Getenv("ORCASTACK_RUNNER_WORKSPACE")
	if workspace == "" {
		workspace = "/workspace/orcastackapi"
	}
	registerWithRunnerAndAgent(
		mux,
		commandRunner{workspace: workspace},
		agentProxy{
			baseURL: os.Getenv("ATONIX_AGENT_API_URL"),
			token:   os.Getenv("ATONIX_AGENT_API_TOKEN"),
			client:  &http.Client{Timeout: 10 * time.Second},
		},
	)
}

func registerWithRunner(mux *http.ServeMux, runner jobRunner) {
	registerWithRunnerAndAgent(mux, runner, agentProxy{})
}

func registerWithRunnerAndAgent(mux *http.ServeMux, runner jobRunner, agent agentProxy) {
	service := &service{logs: make(map[string]string), runner: runner}
	mux.HandleFunc("/pipelines", service.handlePipelines)
	mux.HandleFunc("/pipelines/", service.handlePipeline)
	mux.HandleFunc("/jobs", service.handleJobs)
	mux.HandleFunc("/jobs/", service.handleJob)
	mux.HandleFunc("/events/stream", service.handleEvents)
	mux.HandleFunc("/agent/registry", agent.handleRegistry)
	mux.HandleFunc("/agent/predict", agent.handlePredict)
}

func (agent agentProxy) proxy(w http.ResponseWriter, r *http.Request, method, path string) {
	if agent.baseURL == "" || agent.token == "" {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "AtonixCorp Agent API is not configured on the runner",
		})
		return
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var body []byte
	if method == http.MethodPost {
		if !strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]), "application/json") {
			respondJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
			return
		}
		var err error
		body, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			respondJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body exceeds the 1 MiB limit"})
			return
		}
	}

	target := strings.TrimRight(agent.baseURL, "/") + path
	upstreamRequest, err := http.NewRequestWithContext(
		r.Context(),
		method,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "invalid Agent API URL"})
		return
	}
	upstreamRequest.Header.Set("Accept", "application/json")
	upstreamRequest.Header.Set("Authorization", "Bearer "+agent.token)
	if method == http.MethodPost {
		upstreamRequest.Header.Set("Content-Type", "application/json")
	}

	client := agent.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(upstreamRequest)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("Agent API request failed: %v", err),
		})
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "could not read Agent API response"})
		return
	}
	if len(responseBody) > 1<<20 {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": "Agent API response exceeds the 1 MiB limit"})
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(responseBody)
}

func (agent agentProxy) handleRegistry(w http.ResponseWriter, r *http.Request) {
	agent.proxy(w, r, http.MethodGet, "/v1/agent/registry")
}

func (agent agentProxy) handlePredict(w http.ResponseWriter, r *http.Request) {
	agent.proxy(w, r, http.MethodPost, "/v1/agent/predict")
}

func (service *service) handlePipelines(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/pipelines" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		pipelines, summary := service.snapshot()
		respondJSON(w, http.StatusOK, map[string]any{"pipelines": pipelines, "summary": summary})
	case http.MethodPost:
		service.createPipeline(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (service *service) createPipeline(w http.ResponseWriter, r *http.Request) {
	var request createPipelineRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pipeline payload"})
		return
	}
	request.Project = strings.TrimSpace(request.Project)
	request.Ref = strings.TrimSpace(request.Ref)
	request.Trigger = strings.TrimSpace(request.Trigger)
	request.Workflow = strings.TrimSpace(request.Workflow)
	if request.Project == "" || request.Ref == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "project and ref are required"})
		return
	}
	if request.Trigger == "" {
		request.Trigger = "manual"
	}
	if request.Workflow == "" {
		request.Workflow = "api-test"
	}
	if request.Workflow != "api-test" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported workflow"})
		return
	}

	pipelineID := "pipe-" + uuid.NewString()
	jobID := "job-" + uuid.NewString()
	pipeline := Pipeline{
		ID: pipelineID, Project: request.Project, Ref: request.Ref, Status: "queued",
		StartedAt: time.Now().UTC().Format(time.RFC3339), Trigger: request.Trigger,
		Jobs: []Job{{
			ID: jobID, Name: request.Workflow, Stage: "test", Status: "queued",
			Executor: "orcastack-runner", LogEndpoint: "/jobs/" + jobID + "/logs",
		}},
		Labels: []string{"go", "ci"},
	}
	service.mu.Lock()
	service.pipelines = append(service.pipelines, pipeline)
	service.mu.Unlock()
	respondJSON(w, http.StatusAccepted, pipeline)
	go service.executePipeline(pipelineID, jobID, request.Workflow)
}

func (service *service) executePipeline(pipelineID, jobID, workflow string) {
	startedAt := time.Now().UTC().Format(time.RFC3339)
	service.updateJob(pipelineID, jobID, "running", startedAt, "")
	logs, err := service.runner.Run(context.Background(), workflow)
	status := "success"
	if err != nil {
		status = "failed"
		if logs == "" {
			logs = err.Error()
		}
	}
	service.mu.Lock()
	service.logs[jobID] = logs
	service.mu.Unlock()
	service.updateJob(pipelineID, jobID, status, startedAt, time.Now().UTC().Format(time.RFC3339))
}

func (service *service) updateJob(pipelineID, jobID, status, startedAt, finishedAt string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	for pipelineIndex := range service.pipelines {
		if service.pipelines[pipelineIndex].ID != pipelineID {
			continue
		}
		service.pipelines[pipelineIndex].Status = status
		for jobIndex := range service.pipelines[pipelineIndex].Jobs {
			job := &service.pipelines[pipelineIndex].Jobs[jobIndex]
			if job.ID == jobID {
				job.Status = status
				job.StartedAt = startedAt
				job.FinishedAt = finishedAt
				return
			}
		}
	}
}

func (service *service) handlePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/pipelines/")
	service.mu.RLock()
	defer service.mu.RUnlock()
	for _, pipeline := range service.pipelines {
		if pipeline.ID == id {
			respondJSON(w, http.StatusOK, pipeline)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, map[string]string{"error": "pipeline not found"})
}

func (service *service) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/jobs" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	pipelines, summary := service.snapshot()
	respondJSON(w, http.StatusOK, map[string]any{"jobs": flattenJobs(pipelines), "summary": summary})
}

func (service *service) handleJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if !strings.HasSuffix(path, "/logs") {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "job endpoint not found"})
		return
	}
	jobID := strings.TrimSuffix(path, "/logs")
	service.mu.RLock()
	logs, ok := service.logs[jobID]
	service.mu.RUnlock()
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "job logs not found"})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(logs))
}

func (service *service) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	pipelines, _ := service.snapshot()
	for _, pipeline := range pipelines {
		payload, _ := json.Marshal(map[string]string{"pipeline_id": pipeline.ID, "status": pipeline.Status})
		_, _ = w.Write([]byte("event: pipeline-status\n"))
		_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
		flusher.Flush()
	}
}

func (service *service) snapshot() ([]Pipeline, RunnerSummary) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	pipelines := append([]Pipeline(nil), service.pipelines...)
	jobs := flattenJobs(pipelines)
	summary := RunnerSummary{Capacity: 1, LiveUpdatesMode: "server-sent-events"}
	for _, job := range jobs {
		switch job.Status {
		case "running":
			summary.BusyExecutors++
		case "queued":
			summary.QueuedJobs++
		}
	}
	return pipelines, summary
}

func flattenJobs(pipelines []Pipeline) []Job {
	jobs := make([]Job, 0)
	for _, pipeline := range pipelines {
		jobs = append(jobs, pipeline.Jobs...)
	}
	return jobs
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
