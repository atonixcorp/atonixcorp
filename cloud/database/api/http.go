package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/services"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "database-user-id"

type apiClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type APIServer struct {
	provisioning *services.ProvisioningService
	controller   *core.Controller
	store        core.Store
	jwtSecret    []byte
}

func NewAPIServer(controller *core.Controller, store core.Store, jwtSecret string) *APIServer {
	return &APIServer{
		provisioning: services.NewProvisioningService(controller),
		controller:   controller,
		store:        store,
		jwtSecret:    []byte(jwtSecret),
	}
}

func (s *APIServer) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Group(func(protected chi.Router) {
		protected.Use(s.authenticate)
		protected.Get("/api/v1/database-engines", s.ListEngines)
		protected.Get("/api/v1/database-plans", s.ListPlans)
		protected.Get("/api/v1/tenants/{tenant_id}/projects/{project_id}/databases", s.ListInstances)
		protected.Post("/api/v1/tenants/{tenant_id}/projects/{project_id}/databases", s.CreateInstance)
		protected.Get("/api/v1/tenants/{tenant_id}/projects/{project_id}/databases/{instance_id}", s.GetInstance)
		protected.Get("/api/v1/tenants/{tenant_id}/projects/{project_id}/operations/{operation_id}", s.GetOperation)
		protected.Get("/api/v1/tenants/{tenant_id}/projects/{project_id}/operations/{operation_id}/events", s.StreamOperation)
	})
	return r
}

func (s *APIServer) StreamOperation(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeProject(w, r) {
		return
	}
	operationID := chi.URLParam(r, "operation_id")
	operation, err := s.store.GetOperation(operationID)
	if err != nil || operation.TenantID != chi.URLParam(r, "tenant_id") ||
		operation.ProjectID != chi.URLParam(r, "project_id") {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "event streaming is unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	lastUpdate := time.Time{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	for {
		if !operation.UpdatedAt.Equal(lastUpdate) {
			payload, marshalErr := json.Marshal(operation)
			if marshalErr != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: operation\ndata: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
			lastUpdate = operation.UpdatedAt
			if operation.Status == "blocked" || operation.Status == "failed" ||
				operation.Status == "completed" || operation.Status == "cancelled" {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			operation, err = s.store.GetOperation(operationID)
			if err != nil || operation.TenantID != chi.URLParam(r, "tenant_id") ||
				operation.ProjectID != chi.URLParam(r, "project_id") {
				return
			}
		}
	}
}

func (s *APIServer) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.jwtSecret) == 0 {
			http.Error(w, "database API authentication is not configured", http.StatusServiceUnavailable)
			return
		}
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "bearer token required", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims := &apiClaims{}
		token, err := jwt.ParseWithClaims(
			tokenString, claims,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected JWT signing method")
				}
				return s.jwtSecret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)
		if err != nil || token == nil || !token.Valid || claims.UserID == 0 {
			http.Error(w, "invalid bearer token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *APIServer) authorizeProject(w http.ResponseWriter, r *http.Request) bool {
	userID, ok := r.Context().Value(userIDContextKey).(uint)
	if !ok {
		http.Error(w, "invalid authenticated user", http.StatusUnauthorized)
		return false
	}
	allowed, err := s.store.CanAccessProject(
		userID, chi.URLParam(r, "tenant_id"), chi.URLParam(r, "project_id"),
	)
	if err != nil {
		http.Error(w, "could not verify project access", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "project access denied", http.StatusForbidden)
		return false
	}
	return true
}
