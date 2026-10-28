package containers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
	platformcrypto "atonixcorp/api/internal/security/crypto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	ErrRuntimeUnavailable   = errors.New("Kubernetes runtime is not configured")
	ErrNotFound             = errors.New("container deployment not found")
	ErrConflict             = errors.New("idempotency key was already used for a different request")
	ErrNameConflict         = errors.New("deployment name is already in use in this project")
	ErrForbidden            = errors.New("project access denied")
	ErrInvalidRequest       = errors.New("invalid container deployment request")
	ErrOperationConflict    = errors.New("container deployment operation conflicts with current state")
	ErrSecretKeyUnavailable = errors.New("container secret encryption is not configured")
	ErrMetricsUnavailable   = errors.New("Kubernetes metrics API is not configured")
	ErrRegistryConflict     = errors.New("registry already exists in this project")
)

type Service struct {
	db          *gorm.DB
	provisioner Provisioner
	secretKey   []byte
}

func NewService(db *gorm.DB, provisioner Provisioner, secretKey []byte) *Service {
	return &Service{db: db, provisioner: provisioner, secretKey: secretKey}
}

func SecretKeyFromEnvironment(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, errors.New("CONTAINER_SECRET_KEY must be base64-encoded 32-byte key material")
	}
	return key, nil
}

type RegistryRequest struct {
	TenantID  uint   `json:"tenant_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type RegistryResponse struct {
	ID                uint   `json:"id"`
	TenantID          uint   `json:"tenant_id"`
	ProjectID         uint   `json:"project_id"`
	Name              string `json:"name"`
	URL               string `json:"url"`
	CredentialsStored bool   `json:"credentials_stored"`
	CreatedAt         string `json:"created_at"`
}

func (s *Service) CreateRegistry(ctx context.Context, userID uint, request RegistryRequest) (RegistryResponse, error) {
	if userID == 0 || request.TenantID == 0 || request.ProjectID == 0 {
		return RegistryResponse{}, ErrForbidden
	}
	if err := s.authorizeProject(userID, request.TenantID, request.ProjectID, true); err != nil {
		return RegistryResponse{}, err
	}
	if err := normalizeRegistryRequest(&request); err != nil {
		return RegistryResponse{}, err
	}
	var encrypted []byte
	var err error
	if request.Password != "" {
		if len(s.secretKey) != 32 {
			return RegistryResponse{}, ErrSecretKeyUnavailable
		}
		encrypted, err = platformcrypto.EncryptAESGCM(s.secretKey, []byte(request.Password))
		if err != nil {
			return RegistryResponse{}, errors.New("could not encrypt registry credentials")
		}
	}
	registry := models.ContainerRegistry{
		TenantID: request.TenantID, ProjectID: request.ProjectID, Name: request.Name,
		URL: request.URL, Username: request.Username, EncryptedPassword: encrypted, OwnerID: userID,
	}
	if err := s.db.WithContext(ctx).Create(&registry).Error; err != nil {
		var existing models.ContainerRegistry
		if lookupErr := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND project_id = ? AND name = ?", request.TenantID, request.ProjectID, request.Name,
		).First(&existing).Error; lookupErr == nil {
			return RegistryResponse{}, ErrRegistryConflict
		}
		return RegistryResponse{}, err
	}
	return registryResponse(registry), nil
}

func normalizeRegistryRequest(request *RegistryRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	if len(request.Name) < 2 || len(request.Name) > 100 {
		return fmt.Errorf("%w: registry name must be between 2 and 100 characters", ErrInvalidRequest)
	}
	parsed, err := url.Parse(strings.TrimSpace(request.URL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || len(request.URL) > 512 {
		return fmt.Errorf("%w: registry URL must be an HTTPS origin without embedded credentials or query values", ErrInvalidRequest)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	request.URL = strings.TrimRight(parsed.String(), "/")
	if (request.Username == "") != (request.Password == "") {
		return fmt.Errorf("%w: registry username and password must both be provided", ErrInvalidRequest)
	}
	return nil
}

func (s *Service) ListRegistries(ctx context.Context, userID, tenantID, projectID uint) ([]RegistryResponse, error) {
	if err := s.authorizeProject(userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	var registries []models.ContainerRegistry
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ?", tenantID, projectID).
		Order("created_at DESC").Limit(100).Find(&registries).Error; err != nil {
		return nil, err
	}
	result := make([]RegistryResponse, 0, len(registries))
	for _, registry := range registries {
		result = append(result, registryResponse(registry))
	}
	return result, nil
}

func registryResponse(registry models.ContainerRegistry) RegistryResponse {
	return RegistryResponse{
		ID: registry.ID, TenantID: registry.TenantID, ProjectID: registry.ProjectID,
		Name: registry.Name, URL: registry.URL, CredentialsStored: len(registry.EncryptedPassword) > 0,
		CreatedAt: registry.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) Create(
	ctx context.Context,
	userID, tenantID, projectID uint,
	idempotencyKey, correlationID string,
	spec DeploymentSpec,
) (DeploymentResponse, error) {
	if s.provisioner == nil {
		return DeploymentResponse{}, ErrRuntimeUnavailable
	}
	if userID == 0 || tenantID == 0 || projectID == 0 {
		return DeploymentResponse{}, ErrForbidden
	}
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return DeploymentResponse{}, fmt.Errorf("%w: an idempotency key between 8 and 128 characters is required", ErrInvalidRequest)
	}
	if err := s.authorizeProject(userID, tenantID, projectID, true); err != nil {
		return DeploymentResponse{}, err
	}
	if err := ValidateSpec(&spec); err != nil {
		return DeploymentResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	spec, secretValues := hideSecretValues(spec)
	if len(secretValues) > 0 && len(s.secretKey) != 32 {
		return DeploymentResponse{}, ErrSecretKeyUnavailable
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return DeploymentResponse{}, err
	}
	secretJSON, err := json.Marshal(secretValues)
	if err != nil {
		return DeploymentResponse{}, err
	}
	var encryptedSecrets []byte
	if len(secretValues) > 0 {
		encryptedSecrets, err = platformcrypto.EncryptAESGCM(s.secretKey, secretJSON)
		if err != nil {
			return DeploymentResponse{}, errors.New("could not encrypt deployment secrets")
		}
	}
	hashInput := append(append([]byte(nil), specJSON...), secretJSON...)
	requestHash := fmt.Sprintf("%x", sha256.Sum256(hashInput))
	var existing models.ContainerDeployment
	err = s.db.WithContext(ctx).Where(
		"tenant_id = ? AND idempotency_key = ?", tenantID, idempotencyKey,
	).First(&existing).Error
	if err == nil {
		if existing.OwnerID == userID && existing.ProjectID == projectID && existing.RequestHash == requestHash {
			return responseFor(existing), nil
		}
		return DeploymentResponse{}, ErrConflict
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return DeploymentResponse{}, err
	}
	var named models.ContainerDeployment
	if err := s.db.WithContext(ctx).Where(
		"tenant_id = ? AND project_id = ? AND name = ? AND deleted_at IS NULL",
		tenantID, projectID, spec.Name,
	).First(&named).Error; err == nil {
		return DeploymentResponse{}, ErrNameConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return DeploymentResponse{}, err
	}

	deploymentID, err := randomID()
	if err != nil {
		return DeploymentResponse{}, err
	}
	jobID, err := randomID()
	if err != nil {
		return DeploymentResponse{}, err
	}
	if correlationID == "" {
		correlationID, err = randomID()
		if err != nil {
			return DeploymentResponse{}, err
		}
	}
	record := models.ContainerDeployment{
		ID: deploymentID, TenantID: tenantID, ProjectID: projectID, OwnerID: userID,
		Name: spec.Name, State: "PENDING_VALIDATION", DesiredReplicas: spec.Replicas, CurrentRevision: 1,
		SpecJSON: specJSON, EncryptedSecrets: encryptedSecrets, IdempotencyKey: idempotencyKey,
		RequestHash: requestHash, CorrelationID: correlationID, ProvisioningJobID: jobID,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		if err := createRevision(tx, record, 1, userID, requestHash); err != nil {
			return err
		}
		if err := createEvent(tx, record, "", "PENDING_VALIDATION", "deployment request accepted", userID, "", ""); err != nil {
			return err
		}
		if err := createEvent(tx, record, "PENDING_VALIDATION", "VALIDATED", "request and resource policy validated", userID, "", ""); err != nil {
			return err
		}
		record.State = "QUEUED"
		if err := tx.Model(&record).Update("state", record.State).Error; err != nil {
			return err
		}
		return createEvent(tx, record, "VALIDATED", "QUEUED", "provisioning job queued", userID, "", "")
	})
	if err != nil {
		var existing models.ContainerDeployment
		findErr := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND idempotency_key = ?", tenantID, idempotencyKey,
		).First(&existing).Error
		if findErr == nil {
			if existing.OwnerID == userID && existing.ProjectID == projectID && existing.RequestHash == requestHash {
				return responseFor(existing), nil
			}
			return DeploymentResponse{}, ErrConflict
		}
		return DeploymentResponse{}, err
	}
	return responseFor(record), nil
}

func (s *Service) List(ctx context.Context, userID, tenantID, projectID uint) ([]DeploymentResponse, error) {
	if err := s.authorizeProject(userID, tenantID, projectID, false); err != nil {
		return nil, err
	}
	var records []models.ContainerDeployment
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND project_id = ? AND deleted_at IS NULL", tenantID, projectID).
		Order("created_at DESC").Limit(500).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]DeploymentResponse, 0, len(records))
	for _, record := range records {
		result = append(result, responseFor(record))
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, userID uint, id string) (DeploymentResponse, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return DeploymentResponse{}, err
	}
	response := responseFor(record)
	if s.provisioner != nil && record.State != "DELETED" && record.State != "ROLLED_BACK" {
		namespace := NamespaceName(record.TenantID, record.ProjectID)
		status, statusErr := s.provisioner.Status(ctx, namespace, record.Name)
		if statusErr != nil {
			return DeploymentResponse{}, statusErr
		}
		response.ReadyReplicas = status.ReadyReplicas
	}
	return response, nil
}

func (s *Service) Revisions(ctx context.Context, userID uint, id string) ([]RevisionResponse, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return nil, err
	}
	var revisions []models.ContainerDeploymentRevision
	if err := s.db.WithContext(ctx).
		Where("deployment_id = ? AND tenant_id = ? AND project_id = ?", record.ID, record.TenantID, record.ProjectID).
		Order("revision DESC").Limit(100).Find(&revisions).Error; err != nil {
		return nil, err
	}
	result := make([]RevisionResponse, 0, len(revisions))
	for _, revision := range revisions {
		var spec DeploymentSpec
		if err := json.Unmarshal(revision.SpecJSON, &spec); err != nil {
			return nil, fmt.Errorf("decode deployment revision %d: %w", revision.Revision, err)
		}
		result = append(result, RevisionResponse{
			Revision: revision.Revision, SpecHash: revision.SpecHash, ActorID: revision.ActorID,
			CreatedAt: revision.CreatedAt.UTC().Format(time.RFC3339), Spec: spec,
		})
	}
	return result, nil
}

func (s *Service) Metrics(ctx context.Context, userID uint, id string) (DeploymentMetrics, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return DeploymentMetrics{}, err
	}
	if s.provisioner == nil {
		return DeploymentMetrics{}, ErrRuntimeUnavailable
	}
	return s.provisioner.Metrics(ctx, NamespaceName(record.TenantID, record.ProjectID), record.ID)
}

func (s *Service) Update(
	ctx context.Context,
	userID uint,
	id string,
	expectedRevision uint,
	spec DeploymentSpec,
) (DeploymentResponse, error) {
	record, err := s.findAuthorized(ctx, userID, id, true)
	if err != nil {
		return DeploymentResponse{}, err
	}
	if s.provisioner == nil {
		return DeploymentResponse{}, ErrRuntimeUnavailable
	}
	if record.CurrentRevision != expectedRevision {
		return DeploymentResponse{}, fmt.Errorf("%w: deployment revision changed; reload before updating", ErrOperationConflict)
	}
	if record.State != "RUNNING" && record.State != "DEGRADED" {
		return DeploymentResponse{}, fmt.Errorf("%w: only running deployments can be updated", ErrOperationConflict)
	}
	spec.Name = record.Name
	if err := ValidateSpec(&spec); err != nil {
		return DeploymentResponse{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	previousSecrets, err := s.decryptSecrets(record.EncryptedSecrets)
	if err != nil {
		return DeploymentResponse{}, err
	}
	previousSpec := responseFor(record).Spec
	preserveSecretValues(&spec, previousSpec, previousSecrets)
	spec, secretValues := hideSecretValues(spec)
	if len(secretValues) > 0 && len(s.secretKey) != 32 {
		return DeploymentResponse{}, ErrSecretKeyUnavailable
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return DeploymentResponse{}, err
	}
	secretJSON, err := json.Marshal(secretValues)
	if err != nil {
		return DeploymentResponse{}, err
	}
	var encryptedSecrets []byte
	if len(secretValues) > 0 {
		encryptedSecrets, err = platformcrypto.EncryptAESGCM(s.secretKey, secretJSON)
		if err != nil {
			return DeploymentResponse{}, errors.New("could not encrypt deployment secrets")
		}
	}
	hash := sha256.Sum256(append(append([]byte(nil), specJSON...), secretJSON...))
	specHash := hex.EncodeToString(hash[:])
	namespace := NamespaceName(record.TenantID, record.ProjectID)
	updateResult := s.db.WithContext(ctx).Model(&record).
		Where("current_revision = ? AND state IN ?", expectedRevision, []string{"RUNNING", "DEGRADED"}).
		Update("state", "UPDATING")
	if updateResult.Error != nil {
		return DeploymentResponse{}, updateResult.Error
	}
	if updateResult.RowsAffected != 1 {
		return DeploymentResponse{}, ErrOperationConflict
	}
	if err := s.appendEvent(record, record.State, "UPDATING", "deployment update requested", userID, "", ""); err != nil {
		return DeploymentResponse{}, err
	}

	oldSpec := responseFor(record).Spec
	oldSecrets := previousSecrets
	stage := func(string) error { return nil }
	endpoint, applyErr := s.provisioner.Create(ctx, namespace, record.TenantID, record.ProjectID, record.ID, spec, secretValues, stage)
	if applyErr == nil {
		applyErr = s.provisioner.WaitReady(ctx, namespace, record.Name, spec.Replicas)
	}
	if applyErr != nil {
		_, restoreErr := s.provisioner.Create(ctx, namespace, record.TenantID, record.ProjectID, record.ID, oldSpec, oldSecrets, stage)
		if restoreErr == nil {
			restoreErr = s.provisioner.WaitReady(ctx, namespace, record.Name, record.DesiredReplicas)
		}
		state := "RUNNING"
		code := "UPDATE_FAILED"
		details := trimError(applyErr.Error())
		if restoreErr != nil {
			state, code = "DEGRADED", "UPDATE_ROLLBACK_FAILED"
			details = trimError(restoreErr.Error())
		}
		_ = s.db.WithContext(ctx).Model(&record).Updates(map[string]any{"state": state, "error_code": code, "error_details": details}).Error
		_ = s.appendEvent(record, "UPDATING", state, "deployment update did not complete", userID, code, details)
		return DeploymentResponse{}, applyErr
	}

	nextRevision := expectedRevision + 1
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&record).Where("current_revision = ? AND state = ?", expectedRevision, "UPDATING").
			Updates(map[string]any{
				"spec_json": specJSON, "encrypted_secrets": encryptedSecrets,
				"request_hash": specHash, "current_revision": nextRevision,
				"desired_replicas": spec.Replicas, "endpoint": endpoint,
				"state": "RUNNING", "error_code": "", "error_details": "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOperationConflict
		}
		record.SpecJSON, record.EncryptedSecrets, record.RequestHash = specJSON, encryptedSecrets, specHash
		record.CurrentRevision, record.DesiredReplicas, record.Endpoint, record.State = nextRevision, spec.Replicas, endpoint, "RUNNING"
		if err := createRevision(tx, record, nextRevision, userID, specHash); err != nil {
			return err
		}
		return createEvent(tx, record, "UPDATING", "RUNNING", "deployment updated and readiness verified", userID, "", "")
	}); err != nil {
		return DeploymentResponse{}, err
	}
	return responseFor(record), nil
}

func (s *Service) Rollback(ctx context.Context, userID uint, id string, targetRevision uint) (DeploymentResponse, error) {
	record, err := s.findAuthorized(ctx, userID, id, true)
	if err != nil {
		return DeploymentResponse{}, err
	}
	if targetRevision == 0 || targetRevision >= record.CurrentRevision {
		return DeploymentResponse{}, fmt.Errorf("%w: rollback target must be an earlier revision", ErrInvalidRequest)
	}
	var revision models.ContainerDeploymentRevision
	err = s.db.WithContext(ctx).Where(
		"deployment_id = ? AND tenant_id = ? AND project_id = ? AND revision = ?",
		record.ID, record.TenantID, record.ProjectID, targetRevision,
	).First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DeploymentResponse{}, ErrNotFound
	}
	if err != nil {
		return DeploymentResponse{}, err
	}
	var spec DeploymentSpec
	if err := json.Unmarshal(revision.SpecJSON, &spec); err != nil {
		return DeploymentResponse{}, err
	}
	if s.provisioner == nil {
		return DeploymentResponse{}, ErrRuntimeUnavailable
	}
	if record.State != "RUNNING" && record.State != "DEGRADED" {
		return DeploymentResponse{}, fmt.Errorf("%w: only running deployments can be rolled back", ErrOperationConflict)
	}
	secrets, err := s.decryptSecrets(revision.EncryptedSecrets)
	if err != nil {
		return DeploymentResponse{}, err
	}
	namespace := NamespaceName(record.TenantID, record.ProjectID)
	updateResult := s.db.WithContext(ctx).Model(&record).Where(
		"current_revision = ? AND state IN ?", record.CurrentRevision, []string{"RUNNING", "DEGRADED"},
	).Update("state", "UPDATING")
	if updateResult.Error != nil {
		return DeploymentResponse{}, updateResult.Error
	}
	if updateResult.RowsAffected != 1 {
		return DeploymentResponse{}, ErrOperationConflict
	}
	if err := s.appendEvent(record, record.State, "UPDATING", fmt.Sprintf("rollback to revision %d requested", targetRevision), userID, "", ""); err != nil {
		return DeploymentResponse{}, err
	}
	endpoint, err := s.provisioner.Create(ctx, namespace, record.TenantID, record.ProjectID, record.ID, spec, secrets, func(string) error { return nil })
	if err == nil {
		err = s.provisioner.WaitReady(ctx, namespace, record.Name, spec.Replicas)
	}
	if err != nil {
		details := trimError(err.Error())
		_ = s.db.WithContext(ctx).Model(&record).Updates(map[string]any{
			"state": "DEGRADED", "error_code": "ROLLBACK_FAILED", "error_details": details,
		}).Error
		_ = s.appendEvent(record, "UPDATING", "DEGRADED", "rollback could not be verified", userID, "ROLLBACK_FAILED", details)
		return DeploymentResponse{}, err
	}
	nextRevision := record.CurrentRevision + 1
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&record).Where("current_revision = ? AND state = ?", record.CurrentRevision, "UPDATING").
			Updates(map[string]any{
				"spec_json": revision.SpecJSON, "encrypted_secrets": revision.EncryptedSecrets,
				"request_hash": revision.SpecHash, "current_revision": nextRevision,
				"desired_replicas": spec.Replicas, "endpoint": endpoint,
				"state": "RUNNING", "error_code": "", "error_details": "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOperationConflict
		}
		record.SpecJSON, record.EncryptedSecrets, record.RequestHash = revision.SpecJSON, revision.EncryptedSecrets, revision.SpecHash
		record.CurrentRevision, record.DesiredReplicas, record.Endpoint, record.State = nextRevision, spec.Replicas, endpoint, "RUNNING"
		if err := createRevision(tx, record, nextRevision, userID, revision.SpecHash); err != nil {
			return err
		}
		return createEvent(tx, record, "UPDATING", "RUNNING", fmt.Sprintf("rolled back to revision %d", targetRevision), userID, "", "")
	}); err != nil {
		return DeploymentResponse{}, err
	}
	return responseFor(record), nil
}

func (s *Service) RunReconciler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.reconcileOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("reconcile container deployments: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) reconcileOnce(ctx context.Context) error {
	if s.provisioner == nil {
		return ErrRuntimeUnavailable
	}
	var records []models.ContainerDeployment
	if err := s.db.WithContext(ctx).Where("state IN ?", []string{"RUNNING", "DEGRADED", "UNKNOWN"}).
		Order("updated_at ASC").Limit(100).Find(&records).Error; err != nil {
		return err
	}
	for _, record := range records {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		status, err := s.provisioner.Status(ctx, NamespaceName(record.TenantID, record.ProjectID), record.Name)
		if err != nil {
			if apierrors.IsNotFound(err) && record.State != "UNKNOWN" {
				if transitionErr := s.transition(record.ID, "UNKNOWN", "runtime deployment is missing", 0, "RUNTIME_RESOURCE_MISSING", ""); transitionErr != nil {
					return transitionErr
				}
				continue
			}
			log.Printf("observe deployment %s: %v", record.ID, err)
			continue
		}
		next := "RUNNING"
		if status.AvailableReplicas < record.DesiredReplicas {
			next = "DEGRADED"
		}
		if next != record.State {
			if err := s.transition(record.ID, next, "runtime health reconciled", 0, "", ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) Events(ctx context.Context, userID uint, id string) ([]models.ContainerDeploymentEvent, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return nil, err
	}
	var events []models.ContainerDeploymentEvent
	err = s.db.WithContext(ctx).Where(
		"deployment_id = ? AND tenant_id = ? AND project_id = ?", record.ID, record.TenantID, record.ProjectID,
	).Order("created_at ASC").Limit(1000).Find(&events).Error
	return events, err
}

func (s *Service) Logs(ctx context.Context, userID uint, id string) (string, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return "", err
	}
	if s.provisioner == nil {
		return "", ErrRuntimeUnavailable
	}
	return s.provisioner.Logs(ctx, NamespaceName(record.TenantID, record.ProjectID), record.ID)
}

func (s *Service) RuntimeEvents(ctx context.Context, userID uint, id string) ([]corev1.Event, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return nil, err
	}
	if s.provisioner == nil {
		return nil, ErrRuntimeUnavailable
	}
	return s.provisioner.Events(ctx, NamespaceName(record.TenantID, record.ProjectID), record.Name)
}

func (s *Service) Instances(ctx context.Context, userID uint, id string) ([]corev1.Pod, error) {
	record, err := s.findAuthorized(ctx, userID, id, false)
	if err != nil {
		return nil, err
	}
	if s.provisioner == nil {
		return nil, ErrRuntimeUnavailable
	}
	return s.provisioner.Instances(ctx, NamespaceName(record.TenantID, record.ProjectID), record.ID)
}

func (s *Service) Operate(ctx context.Context, userID uint, id, operation string, replicas int32) (DeploymentResponse, error) {
	record, err := s.findAuthorized(ctx, userID, id, true)
	if err != nil {
		return DeploymentResponse{}, err
	}
	if s.provisioner == nil {
		return DeploymentResponse{}, ErrRuntimeUnavailable
	}
	if record.State == "DELETED" || record.State == "DELETING" {
		return DeploymentResponse{}, fmt.Errorf("%w: deployment is already deleted or deleting", ErrOperationConflict)
	}
	namespace := NamespaceName(record.TenantID, record.ProjectID)
	var newState, reason, intermediateState string
	var waitFor int32
	var scaledSpec *DeploymentSpec
	switch operation {
	case "start":
		if record.State != "STOPPED" {
			return DeploymentResponse{}, fmt.Errorf("%w: only stopped deployments can be started", ErrOperationConflict)
		}
		if err := s.provisioner.Scale(ctx, namespace, record.Name, record.DesiredReplicas); err != nil {
			return DeploymentResponse{}, err
		}
		intermediateState, newState, reason = "SCALING", "RUNNING", "deployment started"
		waitFor = record.DesiredReplicas
	case "stop":
		if record.State != "RUNNING" && record.State != "DEGRADED" {
			return DeploymentResponse{}, fmt.Errorf("%w: only running or degraded deployments can be stopped", ErrOperationConflict)
		}
		if err := s.provisioner.Scale(ctx, namespace, record.Name, 0); err != nil {
			return DeploymentResponse{}, err
		}
		intermediateState, newState, reason = "STOPPING", "STOPPED", "deployment stopped"
		waitFor = 0
	case "restart":
		if err := s.provisioner.Restart(ctx, namespace, record.Name); err != nil {
			return DeploymentResponse{}, err
		}
		intermediateState, newState, reason = "RESTARTING", "RUNNING", "deployment restarted"
		waitFor = record.DesiredReplicas
	case "scale":
		if replicas < 1 || replicas > 50 {
			return DeploymentResponse{}, fmt.Errorf("%w: replicas must be between 1 and 50", ErrInvalidRequest)
		}
		if err := s.provisioner.Scale(ctx, namespace, record.Name, replicas); err != nil {
			return DeploymentResponse{}, err
		}
		record.DesiredReplicas = replicas
		var spec DeploymentSpec
		if err := json.Unmarshal(record.SpecJSON, &spec); err != nil {
			return DeploymentResponse{}, err
		}
		spec.Replicas = replicas
		scaledSpec = &spec
		intermediateState, newState, reason = "SCALING", "RUNNING", "replica count updated"
		waitFor = replicas
	case "delete":
		newState, reason = "DELETING", "deployment deletion requested"
		if err := s.transition(record.ID, newState, reason, userID, "", ""); err != nil {
			return DeploymentResponse{}, err
		}
		var spec DeploymentSpec
		if err := json.Unmarshal(record.SpecJSON, &spec); err != nil {
			return DeploymentResponse{}, err
		}
		if err := s.provisioner.Delete(ctx, namespace, record.Name, record.ID, spec.RetainVolumes); err != nil {
			return DeploymentResponse{}, err
		}
		now := time.Now().UTC()
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&record).Updates(map[string]any{
				"state": "DELETED", "deleted_at": now,
			}).Error; err != nil {
				return err
			}
			record.State, record.DeletedAt, record.UpdatedAt = "DELETED", &now, now
			return createEvent(tx, record, "DELETING", "DELETED", "Kubernetes resources released", userID, "", "")
		}); err != nil {
			return DeploymentResponse{}, err
		}
		return responseFor(record), nil
	case "redeploy", "rollback":
		return DeploymentResponse{}, fmt.Errorf("%w: %s requires revision history, which is not yet implemented", ErrInvalidRequest, operation)
	default:
		return DeploymentResponse{}, fmt.Errorf("%w: unsupported deployment operation", ErrInvalidRequest)
	}
	previousState := record.State
	if err := s.db.WithContext(ctx).Model(&record).Updates(map[string]any{
		"state": intermediateState, "desired_replicas": record.DesiredReplicas,
	}).Error; err != nil {
		return DeploymentResponse{}, err
	}
	record.State = intermediateState
	if err := s.appendEvent(record, previousState, intermediateState, reason, userID, "", ""); err != nil {
		return DeploymentResponse{}, err
	}
	if err := s.provisioner.WaitReplicas(ctx, namespace, record.Name, waitFor); err != nil {
		record.State = "DEGRADED"
		_ = s.db.WithContext(ctx).Model(&record).Update("state", record.State).Error
		_ = s.appendEvent(record, intermediateState, "DEGRADED", "requested lifecycle operation did not converge", userID, "OPERATION_TIMEOUT", trimError(err.Error()))
		return DeploymentResponse{}, err
	}
	if scaledSpec != nil {
		specJSON, err := json.Marshal(scaledSpec)
		if err != nil {
			return DeploymentResponse{}, err
		}
		hash := sha256.Sum256(specJSON)
		specHash := hex.EncodeToString(hash[:])
		nextRevision := record.CurrentRevision + 1
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&record).Where("current_revision = ?", record.CurrentRevision).Updates(map[string]any{
				"state": newState, "desired_replicas": record.DesiredReplicas,
				"spec_json": specJSON, "current_revision": nextRevision,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrOperationConflict
			}
			record.SpecJSON, record.CurrentRevision = specJSON, nextRevision
			record.RequestHash = specHash
			if err := createRevision(tx, record, nextRevision, userID, specHash); err != nil {
				return err
			}
			return createEvent(tx, record, intermediateState, newState, reason, userID, "", "")
		}); err != nil {
			return DeploymentResponse{}, err
		}
	} else if err := s.db.WithContext(ctx).Model(&record).Updates(map[string]any{
		"state": newState, "desired_replicas": record.DesiredReplicas,
	}).Error; err != nil {
		return DeploymentResponse{}, err
	}
	record.State, record.UpdatedAt = newState, time.Now().UTC()
	if scaledSpec == nil {
		if err := s.appendEvent(record, intermediateState, newState, reason, userID, "", ""); err != nil {
			return DeploymentResponse{}, err
		}
	}
	return responseFor(record), nil
}

func (s *Service) RunWorker(ctx context.Context) {
	for {
		processed, err := s.processNext(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("process container deployment job: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if !processed {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}

func (s *Service) processNext(ctx context.Context) (bool, error) {
	var record models.ContainerDeployment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("state = ?", "QUEUED").Order("created_at ASC").First(&record).Error; err != nil {
			return err
		}
		record.State = "PROVISIONING"
		if err := tx.Model(&record).Update("state", record.State).Error; err != nil {
			return err
		}
		return createEvent(tx, record, "QUEUED", "PROVISIONING", "provisioning worker claimed job", 0, "", "")
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var spec DeploymentSpec
	if err := json.Unmarshal(record.SpecJSON, &spec); err != nil {
		_ = s.failAndRollback(record, "INVALID_STORED_SPEC", "deployment specification could not be decoded")
		return true, err
	}
	secrets, err := s.decryptSecrets(record.EncryptedSecrets)
	if err != nil {
		_ = s.failAndRollback(record, "SECRET_DECRYPTION_FAILED", "deployment secrets could not be decrypted")
		return true, err
	}

	provisionContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var endpoint string
	var provisionErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			wait := time.Duration(1<<(attempt-2)) * time.Second
			select {
			case <-provisionContext.Done():
				provisionErr = provisionContext.Err()
			case <-time.After(wait):
			}
		}
		if provisionErr != nil {
			break
		}
		stage := func(next string) error {
			return s.transition(record.ID, next, strings.ToLower(strings.ReplaceAll(next, "_", " ")), 0, "", "")
		}
		endpoint, provisionErr = s.provisioner.Create(
			provisionContext,
			NamespaceName(record.TenantID, record.ProjectID),
			record.TenantID,
			record.ProjectID,
			record.ID,
			spec,
			secrets,
			stage,
		)
		if provisionErr == nil {
			if err := s.transition(record.ID, "VERIFYING", "waiting for desired replicas to become ready", 0, "", ""); err != nil {
				provisionErr = err
			} else {
				provisionErr = s.provisioner.WaitReady(
					provisionContext, NamespaceName(record.TenantID, record.ProjectID), spec.Name, spec.Replicas,
				)
			}
		}
		if provisionErr == nil {
			break
		}
	}
	if provisionErr != nil {
		_ = s.failAndRollback(record, "PROVISIONING_FAILED", provisionErr.Error())
		return true, provisionErr
	}

	if err := s.db.WithContext(ctx).Model(&record).Updates(map[string]any{
		"state": "RUNNING", "endpoint": endpoint, "error_code": "", "error_details": "",
	}).Error; err != nil {
		return true, err
	}
	record.State, record.Endpoint = "RUNNING", endpoint
	return true, s.appendEvent(record, "VERIFYING", "RUNNING", "all desired replicas passed readiness checks", 0, "", "")
}

func (s *Service) failAndRollback(record models.ContainerDeployment, code, details string) error {
	details = trimError(details)
	if err := s.db.Model(&record).Updates(map[string]any{
		"state": "FAILED", "error_code": code, "error_details": details,
	}).Error; err != nil {
		return err
	}
	record.State, record.ErrorCode, record.ErrorDetails = "FAILED", code, details
	if err := s.appendEvent(record, "PROVISIONING", "FAILED", "provisioning failed", 0, code, details); err != nil {
		return err
	}
	if err := s.transition(record.ID, "ROLLING_BACK", "removing partially created resources", 0, code, details); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.provisioner.Delete(ctx, NamespaceName(record.TenantID, record.ProjectID), record.Name, record.ID, false); err != nil {
		return s.transition(record.ID, "FAILED", "automatic rollback could not complete", 0, "ROLLBACK_FAILED", trimError(err.Error()))
	}
	return s.transition(record.ID, "ROLLED_BACK", "partially created resources removed", 0, code, details)
}

func (s *Service) transition(id, next, reason string, actorID uint, errorCode, errorDetails string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var record models.ContainerDeployment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, "id = ?", id).Error; err != nil {
			return err
		}
		previous := record.State
		record.State = next
		record.ErrorCode, record.ErrorDetails = errorCode, errorDetails
		if err := tx.Model(&record).Updates(map[string]any{
			"state": next, "error_code": errorCode, "error_details": errorDetails,
		}).Error; err != nil {
			return err
		}
		return createEvent(tx, record, previous, next, reason, actorID, errorCode, errorDetails)
	})
}

func (s *Service) appendEvent(record models.ContainerDeployment, previous, next, reason string, actorID uint, errorCode, errorDetails string) error {
	return s.db.Create(&models.ContainerDeploymentEvent{
		DeploymentID: record.ID, TenantID: record.TenantID, ProjectID: record.ProjectID,
		PreviousState: previous, NewState: next, Reason: reason, ActorID: actorID,
		CorrelationID: record.CorrelationID, ProvisioningJobID: record.ProvisioningJobID,
		ErrorCode: errorCode, ErrorDetails: errorDetails,
	}).Error
}

func createEvent(
	tx *gorm.DB, record models.ContainerDeployment, previous, next, reason string,
	actorID uint, errorCode, errorDetails string,
) error {
	return tx.Create(&models.ContainerDeploymentEvent{
		DeploymentID: record.ID, TenantID: record.TenantID, ProjectID: record.ProjectID,
		PreviousState: previous, NewState: next, Reason: reason, ActorID: actorID,
		CorrelationID: record.CorrelationID, ProvisioningJobID: record.ProvisioningJobID,
		ErrorCode: errorCode, ErrorDetails: errorDetails,
	}).Error
}

func createRevision(
	tx *gorm.DB,
	record models.ContainerDeployment,
	revision, actorID uint,
	specHash string,
) error {
	return tx.Create(&models.ContainerDeploymentRevision{
		DeploymentID: record.ID, TenantID: record.TenantID, ProjectID: record.ProjectID,
		Revision: revision, SpecJSON: record.SpecJSON, EncryptedSecrets: record.EncryptedSecrets,
		SpecHash: specHash, ActorID: actorID,
	}).Error
}

func (s *Service) authorizeProject(userID, tenantID, projectID uint, write bool) error {
	var tenantMembership models.TenantMembership
	if err := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&tenantMembership).Error; err != nil {
		return ErrForbidden
	}
	var project models.Project
	if err := s.db.Where("id = ? AND tenant_id = ?", projectID, tenantID).First(&project).Error; err != nil {
		return ErrForbidden
	}
	var membership models.ProjectMembership
	if err := s.db.Where("project_id = ? AND user_id = ?", projectID, userID).First(&membership).Error; err != nil {
		return ErrForbidden
	}
	if write && membership.Role != "owner" && membership.Role != "admin" && membership.Role != "deployer" {
		return ErrForbidden
	}
	return nil
}

func (s *Service) findAuthorized(ctx context.Context, userID uint, id string, write bool) (models.ContainerDeployment, error) {
	var record models.ContainerDeployment
	err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ContainerDeployment{}, ErrNotFound
	}
	if err != nil {
		return models.ContainerDeployment{}, err
	}
	if err := s.authorizeProject(userID, record.TenantID, record.ProjectID, write); err != nil {
		return models.ContainerDeployment{}, ErrNotFound
	}
	return record, nil
}

func (s *Service) decryptSecrets(ciphertext []byte) (map[string]string, error) {
	if len(ciphertext) == 0 {
		return map[string]string{}, nil
	}
	if len(s.secretKey) != 32 {
		return nil, errors.New("secret key is not configured")
	}
	plaintext, err := platformcrypto.DecryptAESGCM(s.secretKey, ciphertext)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	if err := json.Unmarshal(plaintext, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func responseFor(record models.ContainerDeployment) DeploymentResponse {
	var spec DeploymentSpec
	_ = json.Unmarshal(record.SpecJSON, &spec)
	spec.Replicas = record.DesiredReplicas
	return DeploymentResponse{
		ID: record.ID, TenantID: record.TenantID, ProjectID: record.ProjectID, Name: record.Name,
		State: record.State, CurrentRevision: record.CurrentRevision, DesiredReplicas: record.DesiredReplicas, Spec: spec,
		Endpoint: record.Endpoint, CorrelationID: record.CorrelationID,
		ProvisioningJobID: record.ProvisioningJobID, ErrorCode: record.ErrorCode,
		ErrorDetails: record.ErrorDetails, CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func NamespaceName(tenantID, projectID uint) string {
	return fmt.Sprintf("atonix-t%d-p%d", tenantID, projectID)
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func trimError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2000 {
		return value[:2000]
	}
	return value
}
