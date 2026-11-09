package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	sdk "atonixcorp/cloud/devtools/sdk/go"
)

func ListCompute(w http.ResponseWriter, r *http.Request) {
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	items, err := client.ListCompute(r.Context())
	writeResult(w, items, err)
}

func CreateCompute(w http.ResponseWriter, r *http.Request) {
	var request sdk.CreateComputeRequest
	if err := decodeRequest(w, r, &request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	item, err := client.CreateCompute(r.Context(), request)
	writeResult(w, item, err)
}

func DeleteCompute(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "compute instance ID is required", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	if err := client.DeleteCompute(r.Context(), id); err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListNetwork(w http.ResponseWriter, r *http.Request) {
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	items, err := client.ListNetwork(r.Context())
	writeResult(w, items, err)
}

func CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var request sdk.CreateNetworkRequest
	if err := decodeRequest(w, r, &request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	item, err := client.CreateNetwork(r.Context(), request)
	writeResult(w, item, err)
}

func DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "network ID is required", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	if err := client.DeleteNetwork(r.Context(), url.PathEscape(id)); err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ListStorage(w http.ResponseWriter, r *http.Request) {
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	items, err := client.ListStorage(r.Context())
	writeResult(w, items, err)
}

func CreateStorage(w http.ResponseWriter, r *http.Request) {
	var request sdk.CreateVolumeRequest
	if err := decodeRequest(w, r, &request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	item, err := client.CreateStorage(r.Context(), request)
	writeResult(w, item, err)
}

func DeleteStorage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "volume ID is required", http.StatusBadRequest)
		return
	}
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	if err := client.DeleteStorage(r.Context(), url.PathEscape(id)); err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func Telemetry(w http.ResponseWriter, r *http.Request) {
	client, err := newSDKClient()
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	snapshot, err := client.Telemetry(r.Context())
	writeResult(w, snapshot, err)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func writeResult(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write DevTools portal response: %v", err)
	}
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	log.Printf("DevTools API request failed: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":  "upstream_error",
		"error": "cloud provider request failed",
	})
}
