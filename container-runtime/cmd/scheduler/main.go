package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"atonixcorp/container-runtime/internal/config"
	"atonixcorp/container-runtime/internal/scheduler"

	"atonixcorp/container-runtime/pkg/models"
	"atonixcorp/container-runtime/pkg/utils"
)

type SchedulerService struct {
	Logger *utils.Logger
	Nodes  map[string]*models.Node
	Lock   sync.RWMutex
	Sched  *scheduler.Scheduler
}

func main() {
	cfg := config.Load()
	log := utils.NewLogger("Scheduler")

	service := &SchedulerService{
		Logger: log,
		Nodes:  make(map[string]*models.Node),
		Sched:  scheduler.NewScheduler([]scheduler.Node{}),
	}

	log.Info("Scheduler service starting...")

	// API endpoints
	http.HandleFunc("/scheduler/heartbeat", service.handleHeartbeat)
	http.HandleFunc("/scheduler/nodes", service.handleNodes)
	http.HandleFunc("/scheduler/schedule", service.handleSchedule)

	log.Info("Scheduler listening on :7070")
	http.ListenAndServe(":7070", nil)
}

//
// ─────────────────────────────────────────────────────────────
//   HEARTBEAT HANDLER
// ─────────────────────────────────────────────────────────────
//

// handleHeartbeat receives node health updates from node-agent.
func (s *SchedulerService) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb struct {
		NodeName   string `json:"node"`
		CPU        int    `json:"cpu"`
		MemoryMB   int    `json:"memory_mb"`
		UsedCPU    int    `json:"used_cpu"`
		UsedMemory int    `json:"used_memory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		writeError(w, "invalid_heartbeat", err.Error())
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()

	s.Nodes[hb.NodeName] = &models.Node{
		Name:          hb.NodeName,
		CPU:           hb.CPU,
		MemoryMB:      hb.MemoryMB,
		UsedCPU:       hb.UsedCPU,
		UsedMemory:    hb.UsedMemory,
		Status:        models.NodeHealthy,
		LastHeartbeat: time.Now(),
	}

	s.Logger.Info("Heartbeat received from node: " + hb.NodeName)

	writeJSON(w, map[string]string{
		"status": "ok",
	})
}

//
// ─────────────────────────────────────────────────────────────
//   NODES HANDLER
// ─────────────────────────────────────────────────────────────
//

// handleNodes returns all known nodes.
func (s *SchedulerService) handleNodes(w http.ResponseWriter, r *http.Request) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()

	var resp models.NodesResponse

	for _, n := range s.Nodes {
		resp.Nodes = append(resp.Nodes, models.NodeInfo{
			Name:       n.Name,
			CPU:        n.CPU,
			MemoryMB:   n.MemoryMB,
			UsedCPU:    n.UsedCPU,
			UsedMemory: n.UsedMemory,
			Status:     string(n.Status),
		})
	}

	writeJSON(w, resp)
}

//
// ─────────────────────────────────────────────────────────────
//   SCHEDULE HANDLER
// ─────────────────────────────────────────────────────────────
//

// handleSchedule selects a node for a container image.
func (s *SchedulerService) handleSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImageID string `json:"image_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", err.Error())
		return
	}

	s.Lock.RLock()
	defer s.Lock.RUnlock()

	// Convert models.Node → scheduler.Node
	var nodes []scheduler.Node
	for _, n := range s.Nodes {
		if n.Status == models.NodeHealthy {
			nodes = append(nodes, scheduler.Node{
				Name:   n.Name,
				CPU:    n.CPU,
				Memory: n.MemoryMB,
			})
		}
	}

	if len(nodes) == 0 {
		writeError(w, "no_nodes_available", "no healthy nodes available")
		return
	}

	s.Sched = scheduler.NewScheduler(nodes)

	selected, err := s.Sched.SelectNode(req.ImageID)
	if err != nil {
		writeError(w, "schedule_error", err.Error())
		return
	}

	resp := models.ScheduleDecision{
		ContainerImage: req.ImageID,
		SelectedNode:   selected.Name,
		Reason:         "least-loaded node selected",
	}

	s.Logger.Info("Scheduled image " + req.ImageID + " to node " + selected.Name)

	writeJSON(w, resp)
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
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   code,
		Details: details,
	})
}
