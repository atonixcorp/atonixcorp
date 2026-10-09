package main

import (
	"encoding/json"
	"net/http"

	"atonixcorp/container-runtime/internal/build"
	"atonixcorp/container-runtime/internal/config"
	"atonixcorp/container-runtime/internal/lifecycle"
	"atonixcorp/container-runtime/internal/logging"
	"atonixcorp/container-runtime/internal/metrics"
	"atonixcorp/container-runtime/internal/registry"
	"atonixcorp/container-runtime/internal/runtime"
	"atonixcorp/container-runtime/internal/scheduler"

	"atonixcorp/container-runtime/pkg/api"
	"atonixcorp/container-runtime/pkg/utils"
)

func main() {
	cfg := config.Load()
	log := utils.NewLogger("API")

	// Core services
	builder := build.NewDefaultBuilder()
	reg := registry.NewRegistryClient(cfg.RegistryURL)
	rt := runtime.NewRuntime()
	lc := lifecycle.NewLifecycleManager()
	logStreamer := logging.NewLogStreamer()
	metricsCollector := metrics.NewMetricsCollector()

	// Scheduler with static nodes (replace with DB or discovery later)
	sched := scheduler.NewScheduler([]scheduler.Node{
		{Name: "node-1", CPU: 8, Memory: 16384},
		{Name: "node-2", CPU: 16, Memory: 32768},
	})

	http.HandleFunc("/api/v1/build", func(w http.ResponseWriter, r *http.Request) {
		var req api.BuildRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid_request", err.Error())
			return
		}

		result, err := builder.Build(r.Context(), build.BuildRequest{
			ProjectID: req.ProjectID,
			SourceDir: req.SourcePath,
			Runtime:   req.Runtime,
		})
		if err != nil {
			writeError(w, "build_failed", err.Error())
			return
		}

		_ = reg.Push(result.ImageID)

		resp := api.BuildResponse{
			BuildID: req.ProjectID,
			ImageID: result.ImageID,
			Logs:    result.Logs,
			Status:  "success",
		}

		writeJSON(w, resp)
	})

	http.HandleFunc("/api/v1/run", func(w http.ResponseWriter, r *http.Request) {
		var req api.RunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid_request", err.Error())
			return
		}

		node, err := sched.SelectNode(req.ImageID)
		if err != nil {
			writeError(w, "no_nodes_available", err.Error())
			return
		}

		_ = reg.Pull(req.ImageID)

		containerID, err := rt.Start(req.ImageID)
		if err != nil {
			writeError(w, "runtime_error", err.Error())
			return
		}

		resp := api.RunResponse{
			ContainerID: containerID,
			Node:        node.Name,
			Status:      "running",
		}

		writeJSON(w, resp)
	})

	http.HandleFunc("/api/v1/lifecycle", func(w http.ResponseWriter, r *http.Request) {
		var req api.LifecycleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid_request", err.Error())
			return
		}

		err := lc.Execute(r.Context(), req.ContainerID, lifecycle.ContainerAction(req.Action))
		if err != nil {
			writeError(w, "lifecycle_error", err.Error())
			return
		}

		resp := api.LifecycleResponse{
			ContainerID: req.ContainerID,
			Action:      req.Action,
			Status:      "ok",
		}

		writeJSON(w, resp)
	})

	http.HandleFunc("/api/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		containerID := r.URL.Query().Get("id")
		if containerID == "" {
			writeError(w, "missing_container_id", "container id is required")
			return
		}

		stream, err := logStreamer.Stream(containerID)
		if err != nil {
			writeError(w, "log_stream_error", err.Error())
			return
		}

		var logs []string
		for entry := range stream {
			logs = append(logs, entry.Message)
		}

		writeJSON(w, api.LogsResponse{
			ContainerID: containerID,
			Logs:        logs,
		})
	})

	http.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		containerID := r.URL.Query().Get("id")
		if containerID == "" {
			writeError(w, "missing_container_id", "container id is required")
			return
		}

		m, err := metricsCollector.Collect(containerID)
		if err != nil {
			writeError(w, "metrics_error", err.Error())
			return
		}

		writeJSON(w, api.MetricsResponse{
			ContainerID: containerID,
			CPUPercent:  m.CPUPercent,
			MemoryMB:    m.MemoryMB,
			NetworkIn:   m.NetworkIn,
			NetworkOut:  m.NetworkOut,
			Timestamp:   m.Timestamp.String(),
		})
	})

	http.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes := sched.Nodes
		var resp api.NodesResponse

		for _, n := range nodes {
			resp.Nodes = append(resp.Nodes, api.NodeInfo{
				Name:       n.Name,
				CPU:        n.CPU,
				MemoryMB:   n.Memory,
				UsedCPU:    0,
				UsedMemory: 0,
				Status:     "healthy",
			})
		}

		writeJSON(w, resp)
	})

	log.Info("API server running on :8080")
	http.ListenAndServe(":8080", nil)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(api.ErrorResponse{
		Error:   code,
		Details: details,
	})
}
