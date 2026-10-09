package main

import (
	"encoding/json"
	"net/http"
	"time"

	"atonixcorp/container-runtime/internal/config"
	"atonixcorp/container-runtime/internal/logging"
	"atonixcorp/container-runtime/internal/metrics"
	"atonixcorp/container-runtime/internal/registry"
	"atonixcorp/container-runtime/internal/runtime"

	"atonixcorp/container-runtime/pkg/models"
	"atonixcorp/container-runtime/pkg/utils"
)

type NodeAgent struct {
	NodeName         string
	Registry         *registry.RegistryClient
	Runtime          *runtime.Runtime
	LogStreamer      *logging.LogStreamer
	MetricsCollector *metrics.MetricsCollector
	Logger           *utils.Logger
}

func main() {
	cfg := config.Load()
	log := utils.NewLogger("NodeAgent")

	agent := &NodeAgent{
		NodeName:         cfg.NodeName,
		Registry:         registry.NewRegistryClient(cfg.RegistryURL),
		Runtime:          runtime.NewRuntime(),
		LogStreamer:      logging.NewLogStreamer(),
		MetricsCollector: metrics.NewMetricsCollector(),
		Logger:           log,
	}

	log.Info("Node Agent starting for node: " + cfg.NodeName)

	// Background heartbeat to scheduler
	go agent.heartbeatLoop()

	// API endpoints for scheduler → node-agent communication
	http.HandleFunc("/agent/run", agent.handleRunContainer)
	http.HandleFunc("/agent/stop", agent.handleStopContainer)
	http.HandleFunc("/agent/logs", agent.handleLogs)
	http.HandleFunc("/agent/metrics", agent.handleMetrics)

	log.Info("Node Agent listening on :9090")
	http.ListenAndServe(":9090", nil)
}

//
// ─────────────────────────────────────────────────────────────
//   HEARTBEAT LOOP
// ─────────────────────────────────────────────────────────────
//

func (a *NodeAgent) heartbeatLoop() {
	for {
		a.Logger.Info("Heartbeat: node is healthy")

		// TODO: send heartbeat to scheduler service
		// Example:
		// POST /scheduler/heartbeat { node: a.NodeName, cpu: X, mem: Y }

		time.Sleep(5 * time.Second)
	}
}

//
// ─────────────────────────────────────────────────────────────
//   HANDLERS
// ─────────────────────────────────────────────────────────────
//

// handleRunContainer starts a container on this node.
func (a *NodeAgent) handleRunContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImageID string `json:"image_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", err.Error())
		return
	}

	a.Logger.Info("Pulling image: " + req.ImageID)
	if err := a.Registry.Pull(req.ImageID); err != nil {
		writeError(w, "registry_error", err.Error())
		return
	}

	a.Logger.Info("Starting container for image: " + req.ImageID)
	containerID, err := a.Runtime.Start(req.ImageID)
	if err != nil {
		writeError(w, "runtime_error", err.Error())
		return
	}

	resp := models.Container{
		ID:        containerID,
		ImageID:   req.ImageID,
		Node:      a.NodeName,
		Status:    models.ContainerRunning,
		CreatedAt: time.Now(),
	}

	writeJSON(w, resp)
}

// handleStopContainer stops a running container.
func (a *NodeAgent) handleStopContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContainerID string `json:"container_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", err.Error())
		return
	}

	a.Logger.Info("Stopping container: " + req.ContainerID)

	if err := a.Runtime.Stop(req.ContainerID); err != nil {
		writeError(w, "runtime_error", err.Error())
		return
	}

	writeJSON(w, map[string]string{
		"container_id": req.ContainerID,
		"status":       "stopped",
	})
}

// handleLogs streams logs from a container.
func (a *NodeAgent) handleLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		writeError(w, "missing_container_id", "container id is required")
		return
	}

	stream, err := a.LogStreamer.Stream(containerID)
	if err != nil {
		writeError(w, "log_stream_error", err.Error())
		return
	}

	var logs []string
	for entry := range stream {
		logs = append(logs, entry.Message)
	}

	writeJSON(w, map[string]interface{}{
		"container_id": containerID,
		"logs":         logs,
	})
}

// handleMetrics returns container metrics.
func (a *NodeAgent) handleMetrics(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		writeError(w, "missing_container_id", "container id is required")
		return
	}

	m, err := a.MetricsCollector.Collect(containerID)
	if err != nil {
		writeError(w, "metrics_error", err.Error())
		return
	}

	writeJSON(w, m)
}

//
// ─────────────────────────────────────────────────────────────
//   HELPERS
// ─────────────────────────────────────────────────────────────
//

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"details": details,
	})
}
