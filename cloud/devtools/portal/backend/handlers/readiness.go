package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func Ready(w http.ResponseWriter, r *http.Request) {
	endpoint, err := url.Parse(strings.TrimRight(os.Getenv("ATCLOUD_ENDPOINT"), "/"))
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		http.Error(w, "upstream endpoint is not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String()+"/health", nil)
	if err != nil {
		http.Error(w, "upstream health check could not be created", http.StatusServiceUnavailable)
		return
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		log.Printf("DevTools upstream readiness check failed: %v", err)
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Printf("DevTools upstream readiness returned status %d", response.StatusCode)
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ready"}); err != nil {
		log.Printf("write DevTools readiness response: %v", fmt.Errorf("encode response: %w", err))
	}
}
