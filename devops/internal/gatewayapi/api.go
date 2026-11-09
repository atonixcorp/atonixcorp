package gatewayapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const authSessionTTL = 12 * time.Hour

type AuthUser struct {
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Identity    string   `json:"identity"`
	RBACRealm   string   `json:"rbac_realm"`
	Permissions []string `json:"permissions"`
}

type AuthSession struct {
	Token     string   `json:"token,omitempty"`
	User      AuthUser `json:"user"`
	ExpiresAt string   `json:"expires_at"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type signupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

var authSessions = struct {
	sync.RWMutex
	store map[string]sessionRecord
}{store: make(map[string]sessionRecord)}

type sessionRecord struct {
	User      AuthUser
	ExpiresAt time.Time
}

type Provider struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Repos     int    `json:"repos"`
	Latency   string `json:"latency"`
	Identity  string `json:"identity"`
	Connected bool   `json:"connected"`
}

type SecurityState struct {
	LDAPRegistered    bool `json:"ldap_registered"`
	RBACVerified      bool `json:"rbac_verified"`
	AttestationSigned bool `json:"attestation_signed"`
	Verified          bool `json:"verified"`
}

type Repository struct {
	ID            string        `json:"id"`
	ProviderID    string        `json:"provider_id"`
	Name          string        `json:"name"`
	Branch        string        `json:"branch"`
	DefaultBranch string        `json:"default_branch"`
	Commit        string        `json:"commit"`
	LastCommitAt  string        `json:"last_commit_at"`
	Reviewer      string        `json:"reviewer"`
	Summary       string        `json:"summary"`
	CloneURL      string        `json:"clone_url"`
	Identity      string        `json:"identity"`
	Security      SecurityState `json:"security"`
}

type CloneOperation struct {
	RepositoryID string `json:"repository_id"`
	Status       string `json:"status"`
	CloneURL     string `json:"clone_url"`
	Command      string `json:"command"`
	UPI          string `json:"upi"`
	UpdatedAt    string `json:"updated_at"`
}

type Review struct {
	ID                string   `json:"id"`
	RepositoryID      string   `json:"repository_id"`
	Title             string   `json:"title"`
	Status            string   `json:"status"`
	RequiredApprovals int      `json:"required_approvals"`
	Approvals         int      `json:"approvals"`
	Reviewers         []string `json:"reviewers"`
	LastUpdated       string   `json:"last_updated"`
}

type PipelineStage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type PipelineRun struct {
	ID        string `json:"id"`
	StartedAt string `json:"started_at"`
	Status    string `json:"status"`
	Trigger   string `json:"trigger"`
}

type Pipeline struct {
	ID           string          `json:"id"`
	RepositoryID string          `json:"repository_id"`
	Name         string          `json:"name"`
	Branch       string          `json:"branch"`
	LastRun      string          `json:"last_run"`
	Status       string          `json:"status"`
	Stages       []PipelineStage `json:"stages"`
	RunHistory   []PipelineRun   `json:"run_history"`
	LogChannel   string          `json:"log_channel"`
	UPI          string          `json:"upi"`
	Security     SecurityState   `json:"security"`
	UpdatedAt    string          `json:"updated_at"`
}

type Deployment struct {
	ID              string        `json:"id"`
	RepositoryID    string        `json:"repository_id"`
	ServiceName     string        `json:"service_name"`
	Version         string        `json:"version"`
	Environment     string        `json:"environment"`
	Status          string        `json:"status"`
	Cluster         string        `json:"cluster"`
	Artifact        string        `json:"artifact"`
	TargetCommit    string        `json:"target_commit"`
	PreviousVersion string        `json:"previous_version"`
	LogChannel      string        `json:"log_channel"`
	UPI             string        `json:"upi"`
	Security        SecurityState `json:"security"`
}

type Container struct {
	Name       string        `json:"name"`
	UPI        string        `json:"upi"`
	State      string        `json:"state"`
	Host       string        `json:"host"`
	Actions    []string      `json:"actions"`
	CPU        string        `json:"cpu"`
	Memory     string        `json:"memory"`
	Restarts   int           `json:"restarts"`
	MetricsURL string        `json:"metrics_url"`
	LogChannel string        `json:"log_channel"`
	Security   SecurityState `json:"security"`
}

type DashboardSecurity struct {
	RepositoryIdentity string        `json:"repository_identity"`
	UIProcessIdentity  string        `json:"ui_process_identity"`
	Directory          SecurityState `json:"directory"`
}

type Event struct {
	ID           string `json:"id"`
	Time         string `json:"time"`
	Component    string `json:"component"`
	Kind         string `json:"kind"`
	RepositoryID string `json:"repository_id"`
	Action       string `json:"action"`
	Result       string `json:"result"`
	UPI          string `json:"upi"`
	Summary      string `json:"summary"`
}

type CloudLayer struct {
	Name     string   `json:"name"`
	Platform string   `json:"platform"`
	Status   string   `json:"status"`
	Endpoint string   `json:"endpoint"`
	Identity string   `json:"identity"`
	Summary  string   `json:"summary"`
	Coverage []string `json:"coverage"`
}

type Cluster struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Provider           string `json:"provider"`
	Status             string `json:"status"`
	Version            string `json:"version"`
	ControlPlanes      int    `json:"control_planes"`
	Workers            int    `json:"workers"`
	GPUWorkers         int    `json:"gpu_workers"`
	RancherProject     string `json:"rancher_project"`
	RegistrationStatus string `json:"registration_status"`
	UpgradePolicy      string `json:"upgrade_policy"`
	APIEndpoint        string `json:"api_endpoint"`
}

type AutomationLane struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Entrypoint string `json:"entrypoint"`
	Target     string `json:"target"`
	LastRun    string `json:"last_run"`
}

type ObservabilitySurface struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
	Endpoint string `json:"endpoint"`
	Backing  string `json:"backing"`
}

type SelfManagementCapability struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Workflow string `json:"workflow"`
	Summary  string `json:"summary"`
}

type Overview struct {
	Providers       []Provider                 `json:"providers"`
	Repositories    []Repository               `json:"repositories"`
	CloneOperations []CloneOperation           `json:"clone_operations"`
	Reviews         []Review                   `json:"reviews"`
	Pipelines       []Pipeline                 `json:"pipelines"`
	Deployments     []Deployment               `json:"deployments"`
	Containers      []Container                `json:"containers"`
	Security        DashboardSecurity          `json:"security"`
	Events          []Event                    `json:"events"`
	UpdatedAt       string                     `json:"updated_at"`
	Metrics         []Metric                   `json:"metrics"`
	Activity        []string                   `json:"activity"`
	CloudLayers     []CloudLayer               `json:"cloud_layers"`
	Clusters        []Cluster                  `json:"clusters"`
	AutomationLanes []AutomationLane           `json:"automation_lanes"`
	Observability   []ObservabilitySurface     `json:"observability"`
	SelfManagement  []SelfManagementCapability `json:"self_management"`
}

type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Hint  string `json:"hint"`
}

func Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/public/home", handlePublicHome)
	mux.HandleFunc("/api/public/community", handlePublicCommunity)
	mux.HandleFunc("/api/public/audit", handlePublicHubAudit)
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/signup", handleSignup)
	mux.HandleFunc("/api/auth/signup-requests", requireAuth(handleSignupRequests))
	mux.HandleFunc("/api/auth/signup-requests/", requireAuth(handleSignupRequestReview))
	mux.HandleFunc("/api/auth/session", handleSession)
	mux.HandleFunc("/api/auth/logout", handleLogout)
	mux.HandleFunc("/api/header/navigation", requireAuth(handleHeaderNavigation))
	mux.HandleFunc("/api/header/audit", requireAuth(handleHeaderAudit))
	mux.HandleFunc("/api/organizations", requireAuth(handleOrganizations))
	mux.HandleFunc("/api/organizations/", requireAuth(handleOrganization))
	mux.HandleFunc("/api/community/posts", requireAuth(handleCommunityPosts))
	mux.HandleFunc("/git/", serveGitHTTP)

	mux.HandleFunc("/api/providers", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Providers)
	}))
	mux.HandleFunc("/api/repositories", requireAuth(handleRepositories))
	mux.HandleFunc("/api/repositories/import", requireAuth(handleRepositoryImport))
	mux.HandleFunc("/api/reviews", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Reviews)
	}))
	mux.HandleFunc("/api/pipelines", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Pipelines)
	}))
	mux.HandleFunc("/api/deployments", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Deployments)
	}))
	mux.HandleFunc("/api/containers", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Containers)
	}))
	mux.HandleFunc("/api/overview", requireAuth(func(w http.ResponseWriter, _ *http.Request, _ sessionRecord) {
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview)
	}))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request loginRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid login payload"))
		return
	}

	if err := ensureConfiguredLocalAdminAccount(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}

	user, err := authenticateUser(request.Username, request.Password)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}

	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("failed to create session token"))
		return
	}

	expiresAt := time.Now().UTC().Add(authSessionTTL)
	authSessions.Lock()
	authSessions.store[token] = sessionRecord{User: user, ExpiresAt: expiresAt}
	authSessions.Unlock()

	writeJSON(w, AuthSession{Token: token, User: user, ExpiresAt: expiresAt.Format(time.RFC3339)})
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request signupRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid signup payload"))
		return
	}

	username := strings.TrimSpace(request.Username)
	email := strings.TrimSpace(request.Email)
	password := strings.TrimSpace(request.Password)

	if username == "" {
		writeError(w, http.StatusBadRequest, errors.New("username is required"))
		return
	}
	if email == "" {
		writeError(w, http.StatusBadRequest, errors.New("email is required"))
		return
	}
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, errors.New("email must be valid"))
		return
	}
	if len(password) < 8 {
		writeError(w, http.StatusBadRequest, errors.New("password must be at least 8 characters"))
		return
	}

	requestID := uuid.NewString()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "pending_review"
	message := "Account request submitted for administrator review."
	if localAuthEnabled() {
		localStatus, err := createLocalAuthAccount(ctx, requestID, username, email, password)
		if err != nil {
			status := http.StatusInternalServerError
			if isConflictError(err) || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				status = http.StatusConflict
			}
			writeError(w, status, err)
			return
		}
		status = localStatus
		if status == "approved" {
			message = "Account created. You can sign in now."
		}
	}

	if err := createSignupRequestRecord(ctx, signupRequestRecord{
		ID:       requestID,
		Username: username,
		Email:    email,
		Status:   status,
	}); err != nil {
		status := http.StatusInternalServerError
		if isConflictError(err) || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}

	writeJSONStatus(w, http.StatusAccepted, signupResponse{
		RequestID: requestID,
		Status:    status,
		Message:   message,
	})
}

func handleSignupRequests(w http.ResponseWriter, r *http.Request, session sessionRecord) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isAdminSession(session) {
		writeError(w, http.StatusForbidden, errors.New("administrator access is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	requests, err := listSignupRequestRecords(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, requests)
}

func handleSignupRequestReview(w http.ResponseWriter, r *http.Request, session sessionRecord) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isAdminSession(session) {
		writeError(w, http.StatusForbidden, errors.New("administrator access is required"))
		return
	}

	requestID := strings.TrimPrefix(r.URL.Path, "/api/auth/signup-requests/")
	requestID = strings.TrimSuffix(requestID, "/review")
	requestID = strings.Trim(requestID, "/")
	if requestID == "" {
		writeError(w, http.StatusBadRequest, errors.New("signup request id is required"))
		return
	}

	var decision signupDecisionRequest
	if err := decodeJSON(r, &decision); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid signup review payload"))
		return
	}

	decision.Status = strings.TrimSpace(decision.Status)
	if decision.Status != "approved" && decision.Status != "rejected" {
		writeError(w, http.StatusBadRequest, errors.New("signup request status must be approved or rejected"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	updated, err := reviewSignupRequestRecord(ctx, requestID, session.User.Username, decision)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	if err := syncLocalAuthAccountReview(ctx, requestID, decision.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, updated)
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requireAuth(func(w http.ResponseWriter, _ *http.Request, session sessionRecord) {
		writeJSON(w, AuthSession{User: session.User, ExpiresAt: session.ExpiresAt.Format(time.RFC3339)})
	})(w, r)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token := bearerToken(r)
	if token != "" {
		authSessions.Lock()
		delete(authSessions.store, token)
		authSessions.Unlock()
	}

	w.WriteHeader(http.StatusNoContent)
}

func requireAuth(next func(w http.ResponseWriter, r *http.Request, session sessionRecord)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, errors.New("missing bearer token"))
			return
		}

		session, ok := lookupSession(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, errors.New("session expired or invalid"))
			return
		}

		next(w, r, session)
	}
}

func handleRepositories(w http.ResponseWriter, r *http.Request, _ sessionRecord) {
	switch r.Method {
	case http.MethodGet:
		overview, err := overviewData()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, overview.Repositories)
	case http.MethodPost:
		var req createRepositoryRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		response, err := createRepository(req)
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "repository name is required" {
				status = http.StatusBadRequest
			}
			if isConflictError(err) {
				status = http.StatusConflict
			}
			writeError(w, status, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, response)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleRepositoryImport(w http.ResponseWriter, r *http.Request, _ sessionRecord) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req importRepositoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := importRepository(req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "source repository URL is required" || err.Error() == "repository name is required" {
			status = http.StatusBadRequest
		}
		if isConflictError(err) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, response)
}

func lookupSession(token string) (sessionRecord, bool) {
	authSessions.RLock()
	session, ok := authSessions.store[token]
	authSessions.RUnlock()
	if !ok {
		return sessionRecord{}, false
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		authSessions.Lock()
		delete(authSessions.store, token)
		authSessions.Unlock()
		return sessionRecord{}, false
	}
	return session, true
}

func bearerToken(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return ""
	}
	prefix := "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
}

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isConflictError(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

func isAdminSession(session sessionRecord) bool {
	if session.User.Role == "platform-admin" {
		return true
	}
	for _, permission := range session.User.Permissions {
		if permission == "control-panel:admin" {
			return true
		}
	}
	return false
}
