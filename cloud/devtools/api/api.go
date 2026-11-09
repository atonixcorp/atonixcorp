package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type DevToolsAPI struct {
	Version     string            `json:"version"`
	Build       string            `json:"build"`
	Timestamp   int64             `json:"timestamp"`
	Services    map[string]string `json:"services"`
	SDKs        []string          `json:"sdks"`
	CLICommands []string          `json:"cli_commands"`
	Portal      string            `json:"portal"`
}

func Info(w http.ResponseWriter, r *http.Request) {
	resp := DevToolsAPI{
		Version:   "v1.0.0",
		Build:     "stable",
		Timestamp: time.Now().Unix(),

		Services: map[string]string{
			"compute":       "/compute",
			"network":       "/network",
			"storage":       "/storage",
			"swift":         "/swift",
			"identity":      "/identity",
			"images":        "/images",
			"orchestration": "/orchestration",
			"projects":      "/projects",
			"telemetry":     "/telemetry",
		},

		SDKs: []string{
			"go",
			"python",
			"javascript",
			"java",
		},

		CLICommands: []string{
			"create compute",
			"delete compute",
			"list compute",
			"create network",
			"delete network",
			"list network",
			"create storage",
			"delete storage",
			"list storage",
			"create container",
			"delete container",
			"upload object",
			"delete object",
			"identity",
			"list images",
			"list events",
			"list metrics",
			"list projects",
			"telemetry",
		},

		Portal: "/portal",
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, resp)
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func NewRouter(apiToken string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", Health)
	mux.HandleFunc("GET /ready", Health)
	mux.HandleFunc("GET /info", Info)

	protected := func(handler http.HandlerFunc) http.Handler {
		return requireToken(apiToken, handler)
	}
	for _, pattern := range []string{
		"GET /projects/list",
		"GET /compute/list",
		"POST /compute/create",
		"DELETE /compute/delete/{id}",
		"GET /network/list",
		"POST /network/create",
		"DELETE /network/delete/{id}",
		"GET /storage/list",
		"POST /storage/create",
		"DELETE /storage/delete/{id}",
		"GET /telemetry",
		"GET /identity/info",
		"GET /images/list",
		"GET /orchestration/events",
		"GET /orchestration/metrics",
		"POST /swift/container/create",
		"DELETE /swift/container/delete/{name}",
		"GET /swift/container/list",
		"POST /swift/object/upload",
		"DELETE /swift/object/delete/{container}/{name}",
	} {
		mux.Handle(pattern, protected(providerUnavailable))
	}
	return mux
}

func requireToken(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if expected == "" || !strings.HasPrefix(header, prefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, prefix)), []byte(expected)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"code":  "unauthorized",
				"error": "authentication required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func providerUnavailable(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"code":  "provider_unconfigured",
		"error": "this resource operation requires a configured cloud provider",
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
