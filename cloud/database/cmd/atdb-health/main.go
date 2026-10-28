package main

import (
	"context"
	"net/http"
	"os"
	"time"
)

func main() {
	url := os.Getenv("DATABASE_HEALTH_URL")
	if url == "" {
		url = "http://127.0.0.1:8085/health"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		os.Exit(1)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		os.Exit(1)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
