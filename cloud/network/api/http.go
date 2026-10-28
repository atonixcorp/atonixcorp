package api

import (
	"fmt"
	"net/http"

	"atnetwork/core"

	"github.com/gorilla/mux"
)

func StartHTTP(controller *core.Controller, policy core.Policy, authorizer projectAuthorizer, jwtSecret string) error {
	if authorizer == nil {
		return fmt.Errorf("networking project authorizer is required")
	}
	if jwtSecret == "" {
		return fmt.Errorf("networking JWT secret is required")
	}
	fmt.Println("ATN API listening on :8080")
	return http.ListenAndServe(":8080", NewRouter(controller, policy, authorizer, jwtSecret))
}

func NewRouter(controller *core.Controller, policy core.Policy, authorizer projectAuthorizer, jwtSecret string) http.Handler {
	h := NewHandler(controller, policy)

	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)
	r.Handle("/api/v1/networking/capabilities", authenticate(jwtSecret, http.HandlerFunc(h.Capabilities))).
		Methods(http.MethodGet)
	scoped := r.PathPrefix("/api/v1/networking/tenants/{tenantID}/projects/{projectID}").Subrouter()
	scoped.Use(func(next http.Handler) http.Handler {
		return authenticate(jwtSecret, authorizeProject(authorizer, next))
	})
	scoped.HandleFunc("/networks", h.ListNetworks).Methods(http.MethodGet)
	scoped.HandleFunc("/networks", h.CreateNetwork).Methods(http.MethodPost)
	scoped.HandleFunc("/networks/{networkID}/subnets", h.CreateSubnet).Methods(http.MethodPost)
	return r
}
