package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"atonixcorp/cloud/compute/controlplane"
	"golang.org/x/crypto/ssh"
)

var (
	namePattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	tagPattern      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,62}$`)
)

type Server struct {
	catalog   controlplane.Catalog
	store     controlplane.Store
	apiToken  string
	nodeToken string
}

type internalError struct {
	err error
}

func (e internalError) Error() string {
	return e.err.Error()
}

func (e internalError) Unwrap() error {
	return e.err
}

func NewServer(catalog controlplane.Catalog, store controlplane.Store, apiToken, nodeToken string) (*Server, error) {
	if store == nil {
		return nil, errors.New("compute instance store is required")
	}
	if len(apiToken) < 32 {
		return nil, errors.New("compute API token must contain at least 32 characters")
	}
	if len(nodeToken) < 32 {
		return nil, errors.New("compute node token must contain at least 32 characters")
	}
	if subtle.ConstantTimeCompare([]byte(apiToken), []byte(nodeToken)) == 1 {
		return nil, errors.New("compute API and node tokens must be different")
	}
	return &Server{catalog: catalog, store: store, apiToken: apiToken, nodeToken: nodeToken}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/compute/types", s.computeTypes)
	mux.HandleFunc("GET /api/v1/compute/regions", s.regions)
	mux.HandleFunc("GET /api/v1/compute/images", s.images)
	mux.HandleFunc("GET /api/v1/compute/plans", s.plans)
	mux.HandleFunc("POST /api/v1/compute/estimates", s.estimate)
	mux.HandleFunc("GET /api/v1/compute/networks", s.networks)
	mux.Handle("GET /api/v1/compute/ssh-keys", s.requireToken(s.apiToken, http.HandlerFunc(s.sshKeys)))
	mux.Handle("POST /api/v1/compute/ssh-keys", s.requireToken(s.apiToken, http.HandlerFunc(s.createSSHKey)))
	mux.HandleFunc("GET /api/v1/compute/firewalls", s.firewalls)
	mux.Handle("GET /api/v1/compute/instances", s.requireToken(s.apiToken, http.HandlerFunc(s.listInstances)))
	mux.Handle("POST /api/v1/compute/instances", s.requireToken(s.apiToken, http.HandlerFunc(s.createInstances)))
	mux.Handle("GET /api/v1/compute/instances/{instanceID}", s.requireToken(s.apiToken, http.HandlerFunc(s.getInstance)))
	mux.HandleFunc("POST /api/v1/compute/instances/{instanceID}/status", s.updateInstanceStatus)
	return withJSONHeaders(mux)
}

func (s *Server) requireToken(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "valid bearer credentials are required")
			return
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "valid bearer credentials are required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) computeTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalog.ComputeTypes)
}

func (s *Server) regions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalog.Regions)
}

func (s *Server) images(w http.ResponseWriter, _ *http.Request) {
	images := make([]controlplane.Image, 0, len(s.catalog.Images))
	for _, image := range s.catalog.Images {
		if image.Available {
			images = append(images, image)
		}
	}
	writeJSON(w, http.StatusOK, images)
}

func (s *Server) plans(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	family := r.URL.Query().Get("type")
	if family == "" {
		family = r.URL.Query().Get("family")
	}
	computeType := r.URL.Query().Get("compute_type")
	plans := make([]controlplane.Plan, 0, len(s.catalog.Plans))
	for _, plan := range s.catalog.Plans {
		if !plan.Available ||
			(region != "" && plan.Region != region) ||
			(computeType != "" && plan.ComputeType != computeType) ||
			(family != "" && plan.Family != family) {
			continue
		}
		plans = append(plans, plan)
	}
	writeJSON(w, http.StatusOK, plans)
}

func (s *Server) estimate(w http.ResponseWriter, r *http.Request) {
	var request controlplane.EstimateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.Quantity == 0 {
		request.Quantity = 1
	}
	if request.Quantity < 1 || request.Quantity > 50 {
		writeError(w, http.StatusUnprocessableEntity, "quantity must be between 1 and 50")
		return
	}
	if _, exists := findBy(s.catalog.Regions, func(item controlplane.Region) bool {
		return item.ID == request.Region && item.Available
	}); !exists {
		writeError(w, http.StatusUnprocessableEntity, "selected region is not available")
		return
	}
	plan := s.findPlan(request.PlanID)
	if plan.ID == "" || plan.Region != request.Region || plan.ComputeType != request.ComputeType {
		writeError(w, http.StatusUnprocessableEntity, "selected plan is not available for the chosen region and compute type")
		return
	}
	if len(request.Volumes) > 16 {
		writeError(w, http.StatusUnprocessableEntity, "at most 16 additional volumes are allowed")
		return
	}
	volumeNames := make(map[string]bool, len(request.Volumes))
	for _, volume := range request.Volumes {
		if !namePattern.MatchString(volume.Name) || volume.Type != "nvme_ssd" || volume.SizeGB < 1 || volume.SizeGB > 10000 {
			writeError(w, http.StatusUnprocessableEntity, "additional volumes require a valid name, supported type, and size between 1 and 10000 GB")
			return
		}
		if volumeNames[volume.Name] {
			writeError(w, http.StatusUnprocessableEntity, "volume names must be unique")
			return
		}
		volumeNames[volume.Name] = true
	}

	volumeMonthly := float64(volumeCapacity(request.Volumes)) * s.catalog.VolumeMonthlyPricePerGB
	writeJSON(w, http.StatusOK, controlplane.Estimate{
		HourlyPrice:   roundPrice((plan.HourlyPrice + volumeMonthly/730) * float64(request.Quantity)),
		MonthlyPrice:  roundPrice((plan.MonthlyPrice + volumeMonthly) * float64(request.Quantity)),
		Quantity:      request.Quantity,
		Currency:      plan.Currency,
		PricingSource: "compute_catalog",
		Authoritative: false,
	})
}

func (s *Server) networks(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	networks := make([]controlplane.Network, 0, len(s.catalog.Networks))
	for _, network := range s.catalog.Networks {
		if network.Available && (region == "" || network.Region == region) {
			networks = append(networks, network)
		}
	}
	writeJSON(w, http.StatusOK, networks)
}

func (s *Server) sshKeys(w http.ResponseWriter, _ *http.Request) {
	keys, err := s.store.ListSSHKeys()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list SSH keys")
		log.Printf("list compute SSH keys: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) createSSHKey(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !namePattern.MatchString(request.Name) {
		writeError(w, http.StatusUnprocessableEntity, "name must be 1-63 characters and use letters, numbers, dots, hyphens, or underscores")
		return
	}
	if len(request.PublicKey) > 16384 {
		writeError(w, http.StatusUnprocessableEntity, "public_key is too large")
		return
	}
	key, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(request.PublicKey))
	if err != nil || len(strings.TrimSpace(string(rest))) > 0 {
		writeError(w, http.StatusUnprocessableEntity, "public_key must contain exactly one valid SSH public key")
		return
	}
	id, err := controlplane.NewID("key")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create SSH key")
		log.Printf("create SSH key ID: %v", err)
		return
	}
	sshKey := controlplane.SSHKey{
		ID:          id,
		Name:        request.Name,
		PublicKey:   string(ssh.MarshalAuthorizedKey(key)),
		Fingerprint: ssh.FingerprintSHA256(key),
		KeyType:     key.Type(),
		Available:   true,
	}
	if err := s.store.AddSSHKey(sshKey); err != nil {
		if errors.Is(err, controlplane.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "this SSH key has already been added")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not save SSH key")
		log.Printf("save compute SSH key: %v", err)
		return
	}
	writeJSON(w, http.StatusCreated, sshKey)
}

func (s *Server) firewalls(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, availableFirewalls(s.catalog.Firewalls))
}

func (s *Server) listInstances(w http.ResponseWriter, _ *http.Request) {
	instances, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list compute instances")
		log.Printf("list compute instances: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, instances)
}

func (s *Server) getInstance(w http.ResponseWriter, r *http.Request) {
	instance, err := s.store.Get(r.PathValue("instanceID"))
	if errors.Is(err, controlplane.ErrNotFound) {
		writeError(w, http.StatusNotFound, "compute instance not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not retrieve compute instance")
		log.Printf("get compute instance: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, instance)
}

func (s *Server) createInstances(w http.ResponseWriter, r *http.Request) {
	var request controlplane.InstanceRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.validateRequest(&request); err != nil {
		var internal internalError
		if errors.As(err, &internal) {
			writeError(w, http.StatusInternalServerError, "could not validate compute deployment")
			log.Printf("validate compute deployment: %v", err)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	plan := s.findPlan(request.PlanID)
	now := time.Now().UTC()
	deploymentID, err := controlplane.NewID("dep")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create deployment")
		log.Printf("create deployment ID: %v", err)
		return
	}

	instances := make([]controlplane.Instance, 0, request.Quantity)
	for i := 1; i <= request.Quantity; i++ {
		instanceID, err := controlplane.NewID("ins")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create deployment")
			log.Printf("create instance ID: %v", err)
			return
		}
		name := request.Name
		hostname := request.Hostname
		if request.Quantity > 1 {
			name = fmt.Sprintf("%s-%02d", request.Name, i)
			if hostname == "" {
				hostname = request.Name
			}
			hostname = fmt.Sprintf("%s-%02d", hostname, i)
		} else if hostname == "" {
			hostname = name
		}
		volumeMonthlyPrice := float64(volumeCapacity(request.Volumes)) * s.catalog.VolumeMonthlyPricePerGB
		monthlyPrice := (plan.MonthlyPrice + volumeMonthlyPrice) * float64(request.Quantity)
		instances = append(instances, controlplane.Instance{
			ID:           instanceID,
			DeploymentID: deploymentID,
			Name:         name,
			Hostname:     hostname,
			Label:        request.Label,
			Region:       request.Region,
			ComputeType:  request.ComputeType,
			PlanID:       plan.ID,
			ImageID:      request.ImageID,
			AuthMethod:   request.AuthMethod,
			NetworkID:    request.NetworkID,
			SSHKeyIDs:    append([]string{}, request.SSHKeyIDs...),
			FirewallID:   request.FirewallID,
			PublicIPv4:   request.PublicIPv4,
			PublicIPv6:   request.PublicIPv6,
			PrivateNet:   request.PrivateNet,
			HostnameDNS:  request.HostnameDNS,
			UserData:     request.UserData,
			Tags:         append([]string{}, request.Tags...),
			Volumes:      append([]controlplane.VolumeRequest{}, request.Volumes...),
			Status:       "provisioning",
			Provisioning: initialProvisioningSteps(),
			HourlyPrice:  (plan.HourlyPrice + volumeMonthlyPrice/730),
			MonthlyPrice: monthlyPrice / float64(request.Quantity),
			Currency:     plan.Currency,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	if err := s.store.Create(instances); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save compute deployment")
		log.Printf("save compute deployment: %v", err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"deployment_id": deploymentID,
		"status":        "provisioning",
		"instances":     instances,
	})
}

func (s *Server) validateRequest(request *controlplane.InstanceRequest) error {
	if !namePattern.MatchString(request.Name) {
		return errors.New("name must be 1-63 characters and use letters, numbers, dots, hyphens, or underscores")
	}
	if request.Hostname == "" {
		request.Hostname = strings.ReplaceAll(request.Name, "_", "-")
	}
	if request.Hostname != "" && !hostnamePattern.MatchString(request.Hostname) {
		return errors.New("hostname must be a valid DNS hostname")
	}
	if request.Label == "" {
		request.Label = request.Name
	}
	if strings.TrimSpace(request.Label) == "" || len(request.Label) > 128 {
		return errors.New("label must contain 1-128 non-whitespace characters")
	}
	if request.Quantity == 0 {
		request.Quantity = 1
	}
	if request.Quantity < 1 || request.Quantity > 50 {
		return errors.New("quantity must be between 1 and 50")
	}
	if request.Quantity > 1 && (len(request.Name)+3 > 63 || len(request.Hostname)+3 > 63) {
		return errors.New("name or hostname is too long to add quantity suffixes")
	}
	if request.AuthMethod == "" {
		request.AuthMethod = "ssh_key"
	}
	if request.AuthMethod != "ssh_key" {
		return errors.New("only SSH key authentication is currently supported")
	}
	if len(request.SSHKeyIDs) == 0 {
		return errors.New("at least one SSH key is required")
	}

	region, exists := findBy(s.catalog.Regions, func(item controlplane.Region) bool {
		return item.ID == request.Region
	})
	if !exists || !region.Available {
		return errors.New("selected region is not available")
	}
	computeType, exists := findBy(s.catalog.ComputeTypes, func(item controlplane.ComputeType) bool {
		return item.ID == request.ComputeType
	})
	if !exists {
		return errors.New("selected compute type is not available")
	}
	plan := s.findPlan(request.PlanID)
	if plan.ID == "" || plan.Region != request.Region || plan.ComputeType != computeType.ID {
		return errors.New("selected plan is not available for the chosen region and compute type")
	}
	if _, exists := findBy(s.catalog.Images, func(item controlplane.Image) bool {
		return item.ID == request.ImageID && item.Available
	}); !exists {
		return errors.New("selected image is not available")
	}

	if request.NetworkID != "" {
		if _, exists := findBy(s.catalog.Networks, func(item controlplane.Network) bool {
			return item.ID == request.NetworkID && item.Region == request.Region && item.Available
		}); !exists {
			return errors.New("selected network is not available in the chosen region")
		}
	}
	sshKeys, err := s.store.ListSSHKeys()
	if err != nil {
		return internalError{err: fmt.Errorf("load SSH key inventory: %w", err)}
	}
	seenKeys := make(map[string]bool, len(request.SSHKeyIDs))
	for _, keyID := range request.SSHKeyIDs {
		if seenKeys[keyID] {
			return fmt.Errorf("duplicate SSH key ID %q", keyID)
		}
		seenKeys[keyID] = true
		if _, exists := findBy(sshKeys, func(item controlplane.SSHKey) bool {
			return item.ID == keyID && item.Available
		}); !exists {
			return fmt.Errorf("SSH key %q is not available", keyID)
		}
	}
	if request.FirewallID != "" {
		if _, exists := findBy(s.catalog.Firewalls, func(item controlplane.Firewall) bool {
			return item.ID == request.FirewallID && item.Available
		}); !exists {
			return errors.New("selected firewall is not available")
		}
	}
	if len(request.UserData) > 65536 {
		return errors.New("user_data must be at most 65536 bytes")
	}
	if len(request.Tags) > 20 {
		return errors.New("at most 20 tags are allowed")
	}
	seenTags := make(map[string]bool, len(request.Tags))
	for _, tag := range request.Tags {
		if !tagPattern.MatchString(tag) {
			return fmt.Errorf("invalid tag %q", tag)
		}
		if seenTags[tag] {
			return fmt.Errorf("duplicate tag %q", tag)
		}
		seenTags[tag] = true
	}
	if len(request.Volumes) > 16 {
		return errors.New("at most 16 additional volumes are allowed")
	}
	volumeNames := make(map[string]bool, len(request.Volumes))
	for _, volume := range request.Volumes {
		if volume.Name == "" || !namePattern.MatchString(volume.Name) {
			return errors.New("each additional volume needs a valid name")
		}
		if volume.Type != "nvme_ssd" {
			return fmt.Errorf("volume type %q is not supported", volume.Type)
		}
		if volume.SizeGB < 1 || volume.SizeGB > 10000 {
			return errors.New("volume size must be between 1 and 10000 GB")
		}
		if volumeNames[volume.Name] {
			return fmt.Errorf("duplicate volume name %q", volume.Name)
		}
		volumeNames[volume.Name] = true
	}
	return nil
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	return strings.TrimPrefix(value, prefix), true
}

func (s *Server) updateInstanceStatus(w http.ResponseWriter, r *http.Request) {
	provided, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "valid compute node credentials are required")
		return
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(s.nodeToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "valid compute node credentials are required")
		return
	}

	var update struct {
		Status     string `json:"status"`
		Step       string `json:"step"`
		StepStatus string `json:"step_status"`
		Message    string `json:"message"`
		NodeID     string `json:"node_id"`
		PublicIP   string `json:"public_ip"`
		PrivateIP  string `json:"private_ip"`
	}
	if err := decodeJSON(w, r, &update); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if update.Status != "provisioning" && update.Status != "running" && update.Status != "failed" {
		writeError(w, http.StatusUnprocessableEntity, "status must be provisioning, running, or failed")
		return
	}
	if update.StepStatus != "in_progress" && update.StepStatus != "completed" && update.StepStatus != "failed" {
		writeError(w, http.StatusUnprocessableEntity, "step_status must be in_progress, completed, or failed")
		return
	}
	if update.Status == "failed" && update.StepStatus != "failed" {
		writeError(w, http.StatusUnprocessableEntity, "failed instances require a failed provisioning step")
		return
	}
	if update.Status == "running" && (update.NodeID == "" || update.Step != "health_checks" || update.StepStatus != "completed") {
		writeError(w, http.StatusUnprocessableEntity, "running instances require a completed health_checks step and a node_id")
		return
	}

	instance, err := s.store.Get(r.PathValue("instanceID"))
	if errors.Is(err, controlplane.ErrNotFound) {
		writeError(w, http.StatusNotFound, "compute instance not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not retrieve compute instance")
		log.Printf("get instance for status update: %v", err)
		return
	}
	if instance.Status != "provisioning" {
		writeError(w, http.StatusConflict, "only provisioning instances can receive provisioning updates")
		return
	}
	stepIndex := -1
	for i := range instance.Provisioning {
		if instance.Provisioning[i].Name == update.Step {
			stepIndex = i
			break
		}
	}
	if stepIndex < 0 {
		writeError(w, http.StatusUnprocessableEntity, "unknown provisioning step")
		return
	}
	if update.Status == "running" && stepIndex != len(instance.Provisioning)-1 {
		writeError(w, http.StatusUnprocessableEntity, "health_checks must be the final provisioning step")
		return
	}

	now := time.Now().UTC()
	for i := 0; i < stepIndex; i++ {
		if instance.Provisioning[i].Status != "completed" {
			writeError(w, http.StatusConflict, "provisioning steps must be completed in order")
			return
		}
	}
	if update.StepStatus == "failed" {
		instance.Status = "failed"
	} else if update.Step == "health_checks" && update.Status == "running" {
		instance.Status = "running"
		instance.NodeID = update.NodeID
		instance.PublicIP = update.PublicIP
		instance.PrivateIP = update.PrivateIP
	} else if instance.Provisioning[stepIndex].Status == "completed" && update.StepStatus != "completed" {
		writeError(w, http.StatusConflict, "provisioning step is already completed")
		return
	}
	instance.Provisioning[stepIndex].Status = update.StepStatus
	instance.Provisioning[stepIndex].UpdatedAt = &now
	instance.Provisioning[stepIndex].Message = update.Message
	instance.UpdatedAt = now
	if err := s.store.Update(instance); err != nil {
		if errors.Is(err, controlplane.ErrNotFound) {
			writeError(w, http.StatusNotFound, "compute instance not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update compute instance")
		log.Printf("update compute instance status: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, instance)
}

func (s *Server) findPlan(id string) controlplane.Plan {
	plan, _ := findBy(s.catalog.Plans, func(item controlplane.Plan) bool {
		return item.ID == id && item.Available
	})
	return plan
}

func initialProvisioningSteps() []controlplane.ProvisioningStep {
	names := []string{
		"creating_instance",
		"allocating_compute",
		"attaching_storage",
		"configuring_network",
		"installing_image",
		"applying_ssh_keys",
		"applying_firewall",
		"running_initialization",
		"health_checks",
	}
	steps := make([]controlplane.ProvisioningStep, 0, len(names))
	for _, name := range names {
		steps = append(steps, controlplane.ProvisioningStep{Name: name, Status: "pending"})
	}
	steps[0].Status = "queued"
	return steps
}

func volumeCapacity(volumes []controlplane.VolumeRequest) int {
	total := 0
	for _, volume := range volumes {
		total += volume.SizeGB
	}
	return total
}

func roundPrice(value float64) float64 {
	return math.Round(value*100) / 100
}

func availableFirewalls(firewalls []controlplane.Firewall) []controlplane.Firewall {
	result := make([]controlplane.Firewall, 0, len(firewalls))
	for _, firewall := range firewalls {
		if firewall.Available {
			result = append(result, firewall)
		}
	}
	return result
}

func findBy[T any](items []T, predicate func(T) bool) (T, bool) {
	for _, item := range items {
		if predicate(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != nil {
		if !errors.Is(err, io.EOF) {
			return errors.New("request body must contain a single JSON value")
		}
	} else {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write compute API response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
