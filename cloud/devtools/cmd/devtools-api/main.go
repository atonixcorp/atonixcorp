package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	devtoolsapi "atonixcorp/cloud/devtools/api"
)

func main() {
	token := os.Getenv("ATCLOUD_API_TOKEN")
	if token == "" {
		log.Fatal("ATCLOUD_API_TOKEN is required")
	}
	addr := os.Getenv("ATCLOUD_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           devtoolsapi.NewRouter(token),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("DevTools API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
