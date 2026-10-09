package api

import (
	"encoding/json"
	"net/http"

	"atnetwork/core"
)

type Handler struct {
	Controller *core.Controller
}

func NewHandler(controller *core.Controller) *Handler {
	return &Handler{Controller: controller}
}

// -----------------------------
// NETWORK
// -----------------------------

func (h *Handler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req CreateNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	n := core.Network{
		ID:       core.GenerateID("net"),
		Name:     req.Name,
		CIDR:     req.CIDR,
		TenantID: req.TenantID,
		External: req.External,
	}

	if err := h.Controller.CreateNetwork(n); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(n)
}

// -----------------------------
// SUBNET
// -----------------------------

func (h *Handler) CreateSubnet(w http.ResponseWriter, r *http.Request) {
	var req CreateSubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	s := core.Subnet{
		ID:        core.GenerateID("sub"),
		NetworkID: req.NetworkID,
		CIDR:      req.CIDR,
		GatewayIP: req.GatewayIP,
	}

	if err := h.Controller.CreateSubnet(s); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(s)
}

// -----------------------------
// ROUTER
// -----------------------------

func (h *Handler) CreateRouter(w http.ResponseWriter, r *http.Request) {
	var req CreateRouterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
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
