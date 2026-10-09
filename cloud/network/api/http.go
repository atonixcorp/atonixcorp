package api

import (
	"fmt"
	"net/http"

	"atnetwork/core"

	"github.com/gorilla/mux"
)

func StartHTTP(controller *core.Controller, policy core.Policy) error {
	h := NewHandler(controller)

	r := mux.NewRouter()

	// NETWORK
	r.HandleFunc("/v1/networks", h.CreateNetwork).Methods("POST")

	// SUBNET
	r.HandleFunc("/v1/subnets", h.CreateSubnet).Methods("POST")

	// ROUTER
	r.HandleFunc("/v1/routers", h.CreateRouter).Methods("POST")
	r.HandleFunc("/v1/routers/attach-interface", h.AttachRouterInterface).Methods("POST")

	// PORT
	r.HandleFunc("/v1/ports", h.CreatePort).Methods("POST")

	// SECURITY GROUP
	r.HandleFunc("/v1/security-groups", h.CreateSecurityGroup).Methods("POST")

	// FLOATING IP
	r.HandleFunc("/v1/floatingips", h.CreateFloatingIP).Methods("POST")

	fmt.Println("ATN API listening on :8080")
	return http.ListenAndServe(":8080", r)
}
