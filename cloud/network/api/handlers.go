package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"atnetwork/core"
	"github.com/gorilla/mux"
)

type Handler struct {
	Controller *core.Controller
	Policy     core.Policy
}

func NewHandler(controller *core.Controller, policies ...core.Policy) *Handler {
	policy := core.NewPolicy()
	if len(policies) > 0 && policies[0] != nil {
		policy = policies[0]
	}
	return &Handler{Controller: controller, Policy: policy}
}

// -----------------------------
// NETWORK
// -----------------------------

func (h *Handler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req CreateNetworkRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	n := core.Network{
		ID:        core.GenerateID("net"),
		Name:      req.Name,
		CIDR:      req.CIDR,
		TenantID:  mux.Vars(r)["tenantID"],
		ProjectID: mux.Vars(r)["projectID"],
		External:  false,
	}

	if err := h.Policy.ValidateNetwork(n); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_NETWORK", err.Error())
		return
	}
	if err := h.Controller.CreateNetwork(n); err != nil {
		if errors.Is(err, core.ErrBackendUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "NETWORK_BACKEND_UNAVAILABLE", "Network provisioning is currently unavailable.")
			return
		}
		if errors.Is(err, core.ErrCIDROverlap) {
			writeError(w, http.StatusConflict, "CIDR_CONFLICT", "The requested address range overlaps an existing network.")
			return
		}
		log.Printf("create network: %v", err)
		writeError(w, http.StatusInternalServerError, "NETWORK_CREATE_FAILED", "Network provisioning could not be completed.")
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	tenantID, projectID := mux.Vars(r)["tenantID"], mux.Vars(r)["projectID"]
	networks, err := h.Controller.ListNetworksInProject(tenantID, projectID)
	if err != nil {
		log.Printf("list project networks: %v", err)
		writeError(w, http.StatusInternalServerError, "NETWORK_LIST_FAILED", "Networks could not be loaded.")
		return
	}
	if networks == nil {
		networks = []core.Network{}
	}
	writeJSON(w, http.StatusOK, networks)
}

func (h *Handler) Capabilities(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"network_provisioning": h.Controller.NetworkProvisioningEnabled(),
		"resources":            []string{"networks", "subnets"},
	})
}

// -----------------------------
// SUBNET
// -----------------------------

func (h *Handler) CreateSubnet(w http.ResponseWriter, r *http.Request) {
	var req CreateSubnetRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	s := core.Subnet{
		ID:        core.GenerateID("sub"),
		NetworkID: req.NetworkID,
		CIDR:      req.CIDR,
		GatewayIP: req.GatewayIP,
	}

	if err := h.Policy.ValidateSubnet(s); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SUBNET", err.Error())
		return
	}
	network, err := h.Controller.GetNetworkInProject(
		req.NetworkID, mux.Vars(r)["tenantID"], mux.Vars(r)["projectID"],
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "NETWORK_NOT_FOUND", "Network not found.")
		return
	}
	if err := h.Policy.ValidateSubnetInNetwork(s, network); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SUBNET", err.Error())
		return
	}
	if err := h.Controller.CreateSubnet(s); err != nil {
		if errors.Is(err, core.ErrBackendUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "NETWORK_BACKEND_UNAVAILABLE", "Network provisioning is currently unavailable.")
			return
		}
		if errors.Is(err, core.ErrCIDROverlap) {
			writeError(w, http.StatusConflict, "CIDR_CONFLICT", "The requested subnet overlaps an existing subnet.")
			return
		}
		log.Printf("create subnet: %v", err)
		writeError(w, http.StatusInternalServerError, "SUBNET_CREATE_FAILED", "Subnet provisioning could not be completed.")
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

// -----------------------------
// ROUTER
// -----------------------------

func (h *Handler) CreateRouter(w http.ResponseWriter, r *http.Request) {
	var req CreateRouterRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	rtr := core.Router{
		ID:                core.GenerateID("rtr"),
		Name:              req.Name,
		TenantID:          req.TenantID,
		ExternalNetworkID: req.ExternalNetworkID,
	}

	if err := h.Controller.CreateRouter(rtr); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(rtr)
}

func (h *Handler) AttachRouterInterface(w http.ResponseWriter, r *http.Request) {
	var req AttachRouterInterfaceRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	iface := core.RouterInterface{
		ID:        core.GenerateID("rif"),
		RouterID:  req.RouterID,
		SubnetID:  req.SubnetID,
		IPAddress: req.IPAddress,
	}

	if err := h.Controller.AttachRouterInterface(iface); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(iface)
}

// -----------------------------
// PORT
// -----------------------------

func (h *Handler) CreatePort(w http.ResponseWriter, r *http.Request) {
	var req CreatePortRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	p := core.Port{
		ID:        core.GenerateID("prt"),
		NetworkID: req.NetworkID,
		DeviceID:  req.DeviceID,
		MAC:       req.MAC,
	}

	if err := h.Controller.CreatePort(p); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(p)
}

// -----------------------------
// SECURITY GROUP
// -----------------------------

func (h *Handler) CreateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateSecurityGroupRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	sg := core.SecurityGroup{
		ID:       core.GenerateID("sg"),
		Name:     req.Name,
		TenantID: req.TenantID,
	}

	if err := h.Controller.CreateSecurityGroup(sg); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(sg)
}

// -----------------------------
// FLOATING IP
// -----------------------------

func (h *Handler) CreateFloatingIP(w http.ResponseWriter, r *http.Request) {
	var req CreateFloatingIPRequest
	if err := decodeRequest(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid.")
		return
	}

	fip := core.FloatingIP{
		ID:             core.GenerateID("fip"),
		TenantID:       req.TenantID,
		ExternalIP:     req.ExternalIP,
		InternalPortID: req.InternalPortID,
		InternalIP:     req.InternalIP,
	}

	if err := h.Controller.CreateFloatingIP(fip); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(fip)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("encode networking API response: %v", err)
		http.Error(w, "could not encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(payload, '\n')); err != nil {
		log.Printf("write networking API response: %v", err)
		return
	}
}
