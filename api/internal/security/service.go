package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
	"gorm.io/gorm"
)

var (
	ErrForbidden             = errors.New("security scope access denied")
	ErrNotFound              = errors.New("security resource not found")
	ErrInvalid               = errors.New("invalid security request")
	ErrTooLarge              = errors.New("security request body is too large")
	ErrConflict              = errors.New("security request conflicts with current state")
	ErrSimulationUnavailable = errors.New("security policy simulation is not configured")
)

const (
	maxPolicyRules = 40
	maxPageSize    = 100
)

var policyNamePattern = regexp.MustCompile(`^[\pL\pN][\pL\pN ._()\-]{0,98}[\pL\pN)]$|^[\pL\pN]$`)
var idempotencyKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)

type PolicyRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
}

type CreatePolicyRequest struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Description string       `json:"description"`
	Rules       []PolicyRule `json:"rules"`
}

type Scope struct {
	UserID    uint
	TenantID  uint
	ProjectID uint
}

type RequestContext struct {
	RequestID     string
	CorrelationID string
}

type PolicyResponse struct {
	ID            uint         `json:"id"`
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	Description   string       `json:"description"`
	Rules         []PolicyRule `json:"rules"`
	State         string       `json:"state"`
	Version       uint         `json:"version"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	DataClass     string       `json:"data_classification"`
	MaskingStatus string       `json:"masking_status"`
	SourceQuality string       `json:"source_quality"`
}

type PolicyPage struct {
	Items        []PolicyResponse `json:"items"`
	NextCursor   string           `json:"next_cursor,omitempty"`
	Limit        int              `json:"limit"`
	Capabilities `json:"capabilities"`
}

type Capabilities struct {
	PolicyRead     bool `json:"security.policy.read"`
	PolicyCreate   bool `json:"security.policy.create"`
	PolicySimulate bool `json:"security.policy.simulate"`
	PolicyApprove  bool `json:"security.policy.approve"`
	PolicyDeploy   bool `json:"security.policy.deploy"`
	PolicyRollback bool `json:"security.policy.rollback"`
	AuditRead      bool `json:"security.audit.read"`
}

type SecurityOverview struct {
	PolicyDrafts       int64      `json:"policy_drafts"`
	PolicySimulated    int64      `json:"policy_simulated"`
	TelemetryState     string     `json:"telemetry_state"`
	TelemetryFreshness *time.Time `json:"telemetry_freshness"`
	DataClass          string     `json:"data_classification"`
	MaskingStatus      string     `json:"masking_status"`
	SourceQuality      string     `json:"source_quality"`
	Capabilities       `json:"capabilities"`
}

type AuditResponse struct {
	ID            uint      `json:"id"`
	ActorID       uint      `json:"actor_id"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Result        string    `json:"result"`
	Reason        string    `json:"reason"`
	RequestID     string    `json:"request_id"`
	CorrelationID string    `json:"correlation_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditPage struct {
	Items      []AuditResponse `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Limit      int             `json:"limit"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func ValidatePolicyRequest(request *CreatePolicyRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)
	request.Description = strings.TrimSpace(request.Description)
	if len(request.Name) > 100 || !policyNamePattern.MatchString(request.Name) {
		return fmt.Errorf("%w: policy name must be 1-100 characters and use letters, numbers, spaces, or . _ ( ) -", ErrInvalid)
	}
	if len(request.Description) > 1000 {
		return fmt.Errorf("%w: policy description is too long", ErrInvalid)
	}
	fields, ok := policyFields[request.Type]
	if !ok {
		return fmt.Errorf("%w: unsupported policy type", ErrInvalid)
	}
	if len(request.Rules) == 0 || len(request.Rules) > maxPolicyRules {
		return fmt.Errorf("%w: include between 1 and %d policy rules", ErrInvalid, maxPolicyRules)
	}
	for index := range request.Rules {
		rule := &request.Rules[index]
		rule.Field = strings.TrimSpace(rule.Field)
		rule.Operator = strings.TrimSpace(rule.Operator)
		rule.Value = strings.TrimSpace(rule.Value)
		rule.Effect = strings.TrimSpace(rule.Effect)
		operators, fieldExists := fields[rule.Field]
		if !fieldExists {
			return fmt.Errorf("%w: rule %d uses a field unsupported for this policy type", ErrInvalid, index+1)
		}
		if !operators[rule.Operator] || len(rule.Value) == 0 || len(rule.Value) > 256 {
			return fmt.Errorf("%w: rule %d has an unsupported operator or value", ErrInvalid, index+1)
		}
		if !allowedEffects[rule.Effect] {
			return fmt.Errorf("%w: rule %d has an unsupported effect", ErrInvalid, index+1)
		}
	}
	return nil
}

var policyFields = map[string]map[string]map[string]bool{
	"identity": {
		"identity.role":         {"equals": true, "not_equals": true, "in": true, "not_in": true},
		"identity.mfa_required": {"equals": true, "not_equals": true},
	},
	"network": {
		"network.source_cidr": {"equals": true, "not_equals": true, "in": true, "not_in": true},
		"network.destination_port": {
			"equals": true, "not_equals": true, "in": true, "not_in": true,
			"greater_than": true, "greater_than_or_equal": true,
			"less_than": true, "less_than_or_equal": true,
		},
	},
	"workload": {
		"workload.image_digest": {"equals": true, "not_equals": true, "in": true, "not_in": true},
		"workload.image_signed": {"equals": true, "not_equals": true},
	},
	"data": {
		"data.encryption_enabled": {"equals": true, "not_equals": true},
		"data.classification":     {"equals": true, "not_equals": true, "in": true, "not_in": true},
	},
	"threat_detection": {
		"event.severity": {"equals": true, "not_equals": true, "in": true, "not_in": true},
		"event.action":   {"equals": true, "not_equals": true, "in": true, "not_in": true},
	},
	"vulnerability": {
		"vulnerability.cvss": {
			"equals": true, "not_equals": true, "greater_than": true,
			"greater_than_or_equal": true, "less_than": true, "less_than_or_equal": true,
		},
		"vulnerability.fix_available": {"equals": true, "not_equals": true},
	},
	"compliance": {
		"compliance.control":   {"equals": true, "not_equals": true, "in": true, "not_in": true},
		"compliance.framework": {"equals": true, "not_equals": true},
	},
}

var allowedEffects = map[string]bool{
	"alert": true, "require_review": true, "allow": true, "deny": true,
}

func (s *Service) CreateDraft(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
	idempotencyKey string,
	request CreatePolicyRequest,
) (PolicyResponse, error) {
	if scope.UserID == 0 || scope.TenantID == 0 || scope.ProjectID == 0 {
		return PolicyResponse{}, ErrForbidden
	}
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 ||
		!idempotencyKeyPattern.MatchString(idempotencyKey) {
		return PolicyResponse{}, fmt.Errorf("%w: a valid Idempotency-Key header is required", ErrInvalid)
	}
	hashInput, err := json.Marshal(request)
	if err != nil {
		return PolicyResponse{}, err
	}
	hash := sha256.Sum256(hashInput)
	requestHash := hex.EncodeToString(hash[:])
	var result PolicyResponse
	var denied bool
	var conflict bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeProject(tx, scope, true)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.create", "security_policy", "", "denied", "SCOPE_OR_PERMISSION_DENIED")
		}

		var existing models.SecurityPolicy
		lookupErr := tx.Where(
			"tenant_id = ? AND project_id = ? AND idempotency_key = ?",
			scope.TenantID, scope.ProjectID, idempotencyKey,
		).First(&existing).Error
		if lookupErr == nil {
			if existing.RequestHash != requestHash {
				conflict = true
				return s.writeAudit(tx, scope, requestContext, "security.policy.create", "security_policy", strconv.FormatUint(uint64(existing.ID), 10), "denied", "IDEMPOTENCY_KEY_REUSED")
			}
			result, err = policyResponse(existing)
			if err != nil {
				return err
			}
			return s.writeAudit(tx, scope, requestContext, "security.policy.create", "security_policy", strconv.FormatUint(uint64(existing.ID), 10), "replayed", "IDEMPOTENT_REPLAY")
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}

		rulesJSON, err := json.Marshal(request.Rules)
		if err != nil {
			return err
		}
		policy := models.SecurityPolicy{
			TenantID: scope.TenantID, ProjectID: scope.ProjectID, OwnerID: scope.UserID,
			Name: request.Name, Type: request.Type, Description: request.Description,
			RulesJSON: rulesJSON, State: "Draft", Version: 1,
			IdempotencyKey: idempotencyKey, RequestHash: requestHash,
		}
		if err := tx.Create(&policy).Error; err != nil {
			return err
		}
		result, err = policyResponse(policy)
		if err != nil {
			return err
		}
		return s.writeAudit(tx, scope, requestContext, "security.policy.create", "security_policy", strconv.FormatUint(uint64(policy.ID), 10), "success", "DRAFT_CREATED")
	})
	if err != nil {
		return PolicyResponse{}, err
	}
	if denied {
		return PolicyResponse{}, ErrForbidden
	}
	if conflict {
		return PolicyResponse{}, ErrConflict
	}
	return result, nil
}

func (s *Service) ListPolicies(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
	limit int,
	after uint,
	policyType string,
) (PolicyPage, error) {
	if limit < 1 || limit > maxPageSize {
		return PolicyPage{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalid, maxPageSize)
	}
	if policyType != "" && !policyFieldsExist(policyType) {
		return PolicyPage{}, fmt.Errorf("%w: unsupported policy type filter", ErrInvalid)
	}
	var result PolicyPage
	var denied bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeProject(tx, scope, false)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.search", "security_policy", "", "denied", "SCOPE_OR_PERMISSION_DENIED")
		}
		query := tx.Where("tenant_id = ? AND project_id = ?", scope.TenantID, scope.ProjectID)
		if after > 0 {
			query = query.Where("id > ?", after)
		}
		if policyType != "" {
			query = query.Where("type = ?", policyType)
		}
		var policies []models.SecurityPolicy
		if err := query.Order("id ASC").Limit(limit + 1).Find(&policies).Error; err != nil {
			return err
		}
		if len(policies) > limit {
			result.NextCursor = strconv.FormatUint(uint64(policies[limit-1].ID), 10)
			policies = policies[:limit]
		}
		result.Items = make([]PolicyResponse, 0, len(policies))
		for _, policy := range policies {
			view, err := policyResponse(policy)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, view)
		}
		result.Limit = limit
		capabilities, capabilityErr := capabilitiesFor(scope, tx)
		if capabilityErr != nil {
			return capabilityErr
		}
		result.Capabilities = capabilities
		return s.writeAudit(tx, scope, requestContext, "security.policy.search", "security_policy", "", "success", auditQueryReason(limit, after, policyType))
	})
	if err != nil {
		return PolicyPage{}, err
	}
	if denied {
		return PolicyPage{}, ErrForbidden
	}
	return result, nil
}

func (s *Service) GetPolicy(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
	policyID uint,
) (PolicyResponse, error) {
	var result PolicyResponse
	var denied bool
	var missing bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeProject(tx, scope, false)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.read", "security_policy", strconv.FormatUint(uint64(policyID), 10), "denied", "SCOPE_OR_PERMISSION_DENIED")
		}
		var policy models.SecurityPolicy
		err := tx.Where(
			"id = ? AND tenant_id = ? AND project_id = ?",
			policyID, scope.TenantID, scope.ProjectID,
		).First(&policy).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			missing = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.read", "security_policy", strconv.FormatUint(uint64(policyID), 10), "not_found", "RESOURCE_NOT_FOUND")
		}
		if err != nil {
			return err
		}
		result, err = policyResponse(policy)
		if err != nil {
			return err
		}
		return s.writeAudit(tx, scope, requestContext, "security.policy.read", "security_policy", strconv.FormatUint(uint64(policyID), 10), "success", "POLICY_READ")
	})
	if err != nil {
		return PolicyResponse{}, err
	}
	if denied {
		return PolicyResponse{}, ErrForbidden
	}
	if missing {
		return PolicyResponse{}, ErrNotFound
	}
	return result, nil
}

func (s *Service) SimulatePolicy(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
	policyID uint,
) error {
	var denied bool
	var missing bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeProject(tx, scope, true)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.simulate", "security_policy", strconv.FormatUint(uint64(policyID), 10), "denied", "SCOPE_OR_PERMISSION_DENIED")
		}
		var policy models.SecurityPolicy
		err := tx.Select("id").Where(
			"id = ? AND tenant_id = ? AND project_id = ?",
			policyID, scope.TenantID, scope.ProjectID,
		).First(&policy).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			missing = true
			return s.writeAudit(tx, scope, requestContext, "security.policy.simulate", "security_policy", strconv.FormatUint(uint64(policyID), 10), "not_found", "RESOURCE_NOT_FOUND")
		}
		if err != nil {
			return err
		}
		return s.writeAudit(tx, scope, requestContext, "security.policy.simulate", "security_policy", strconv.FormatUint(uint64(policyID), 10), "unavailable", "SIMULATION_ENGINE_NOT_CONFIGURED")
	})
	if err != nil {
		return err
	}
	if denied {
		return ErrForbidden
	}
	if missing {
		return ErrNotFound
	}
	return ErrSimulationUnavailable
}

func (s *Service) Overview(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
) (SecurityOverview, error) {
	var result SecurityOverview
	var denied bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeProject(tx, scope, false)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.overview.read", "security_overview", "", "denied", "SCOPE_OR_PERMISSION_DENIED")
		}
		base := tx.Model(&models.SecurityPolicy{}).Where(
			"tenant_id = ? AND project_id = ?", scope.TenantID, scope.ProjectID,
		)
		if err := base.Where("state = ?", "Draft").Count(&result.PolicyDrafts).Error; err != nil {
			return err
		}
		result.TelemetryState = "unavailable"
		result.TelemetryFreshness = nil
		result.DataClass = "internal"
		result.MaskingStatus = "not_masked"
		result.SourceQuality = "authoritative_policy_records_no_live_telemetry"
		capabilities, capabilityErr := capabilitiesFor(scope, tx)
		if capabilityErr != nil {
			return capabilityErr
		}
		result.Capabilities = capabilities
		return s.writeAudit(tx, scope, requestContext, "security.overview.read", "security_overview", "", "success", "LIVE_TELEMETRY_UNAVAILABLE")
	})
	if err != nil {
		return SecurityOverview{}, err
	}
	if denied {
		return SecurityOverview{}, ErrForbidden
	}
	return result, nil
}

func (s *Service) ListAudit(
	ctx context.Context,
	scope Scope,
	requestContext RequestContext,
	limit int,
	after uint,
	sinceHours int,
) (AuditPage, error) {
	if limit < 1 || limit > maxPageSize || sinceHours < 1 || sinceHours > 24*30 {
		return AuditPage{}, fmt.Errorf("%w: limit or audit time range is outside allowed bounds", ErrInvalid)
	}
	var result AuditPage
	var denied bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		allowed, authErr := authorizeAudit(tx, scope)
		if authErr != nil {
			return authErr
		}
		if !allowed {
			denied = true
			return s.writeAudit(tx, scope, requestContext, "security.audit.search", "security_audit", "", "denied", "SCOPE_OR_PERMISSION_DENIED")
		}
		since := time.Now().UTC().Add(-time.Duration(sinceHours) * time.Hour)
		query := tx.Where(
			"tenant_id = ? AND project_id = ? AND created_at >= ?",
			scope.TenantID, scope.ProjectID, since,
		)
		if after > 0 {
			query = query.Where("id < ?", after)
		}
		var events []models.SecurityAuditEvent
		if err := query.Order("id DESC").Limit(limit + 1).Find(&events).Error; err != nil {
			return err
		}
		if len(events) > limit {
			result.NextCursor = strconv.FormatUint(uint64(events[limit-1].ID), 10)
			events = events[:limit]
		}
		result.Items = make([]AuditResponse, 0, len(events))
		for _, event := range events {
			result.Items = append(result.Items, auditResponse(event))
		}
		result.Limit = limit
		return s.writeAudit(tx, scope, requestContext, "security.audit.search", "security_audit", "", "success", fmt.Sprintf("LIMIT_%d_SINCE_HOURS_%d", limit, sinceHours))
	})
	if err != nil {
		return AuditPage{}, err
	}
	if denied {
		return AuditPage{}, ErrForbidden
	}
	return result, nil
}

func (s *Service) writeAudit(
	tx *gorm.DB,
	scope Scope,
	requestContext RequestContext,
	action, resourceType, resourceID, result, reason string,
) error {
	return tx.Create(&models.SecurityAuditEvent{
		TenantID: scope.TenantID, ProjectID: scope.ProjectID, ActorID: scope.UserID,
		Action: action, ResourceType: resourceType, ResourceID: resourceID,
		Result: result, Reason: reason,
		RequestID: requestContext.RequestID, CorrelationID: requestContext.CorrelationID,
		CreatedAt: time.Now().UTC(),
	}).Error
}

func authorizeProject(tx *gorm.DB, scope Scope, write bool) (bool, error) {
	var membership struct {
		TenantRole  string
		ProjectRole string
	}
	err := tx.Table("projects").
		Select("tenant_memberships.role AS tenant_role, project_memberships.role AS project_role").
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = projects.tenant_id AND tenant_memberships.user_id = ?", scope.UserID).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id AND project_memberships.user_id = ?", scope.UserID).
		Where("projects.id = ? AND projects.tenant_id = ?", scope.ProjectID, scope.TenantID).
		Take(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !write {
		return true, nil
	}
	return canManagePolicies(membership.TenantRole, membership.ProjectRole), nil
}

func authorizeAudit(tx *gorm.DB, scope Scope) (bool, error) {
	var membership struct {
		TenantRole  string
		ProjectRole string
	}
	err := tx.Table("projects").
		Select("tenant_memberships.role AS tenant_role, project_memberships.role AS project_role").
		Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = projects.tenant_id AND tenant_memberships.user_id = ?", scope.UserID).
		Joins("JOIN project_memberships ON project_memberships.project_id = projects.id AND project_memberships.user_id = ?", scope.UserID).
		Where("projects.id = ? AND projects.tenant_id = ?", scope.ProjectID, scope.TenantID).
		Take(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return canReadSecurityAudit(membership.TenantRole, membership.ProjectRole), nil
}

func canManagePolicies(tenantRole, projectRole string) bool {
	return (tenantRole == "owner" || tenantRole == "admin") &&
		(projectRole == "owner" || projectRole == "admin" || projectRole == "deployer")
}

func canReadSecurityAudit(tenantRole, projectRole string) bool {
	return (tenantRole == "owner" || tenantRole == "admin") &&
		(projectRole == "owner" || projectRole == "admin" || projectRole == "security_auditor")
}

func capabilitiesFor(scope Scope, db *gorm.DB) (Capabilities, error) {
	readAllowed, err := authorizeProject(db, scope, false)
	if err != nil {
		return Capabilities{}, err
	}
	policyWriteAllowed, err := authorizeProject(db, scope, true)
	if err != nil {
		return Capabilities{}, err
	}
	auditReadAllowed, err := authorizeAudit(db, scope)
	if err != nil {
		return Capabilities{}, err
	}
	return Capabilities{
		PolicyRead: readAllowed, PolicyCreate: policyWriteAllowed, PolicySimulate: policyWriteAllowed,
		PolicyApprove: false, PolicyDeploy: false, PolicyRollback: false, AuditRead: auditReadAllowed,
	}, nil
}

func policyResponse(policy models.SecurityPolicy) (PolicyResponse, error) {
	var rules []PolicyRule
	if err := json.Unmarshal(policy.RulesJSON, &rules); err != nil {
		return PolicyResponse{}, err
	}
	if rules == nil {
		rules = []PolicyRule{}
	}
	return PolicyResponse{
		ID: policy.ID, Name: policy.Name, Type: policy.Type, Description: policy.Description,
		Rules: rules, State: policy.State, Version: policy.Version,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
		DataClass: "internal", MaskingStatus: "not_masked", SourceQuality: "authoritative",
	}, nil
}

func auditResponse(event models.SecurityAuditEvent) AuditResponse {
	return AuditResponse{
		ID: event.ID, ActorID: event.ActorID, Action: event.Action,
		ResourceType: event.ResourceType, ResourceID: event.ResourceID,
		Result: event.Result, Reason: event.Reason, RequestID: event.RequestID,
		CorrelationID: event.CorrelationID, CreatedAt: event.CreatedAt,
	}
}

func policyFieldsExist(policyType string) bool {
	_, exists := policyFields[policyType]
	return exists
}

func auditQueryReason(limit int, after uint, policyType string) string {
	return fmt.Sprintf("LIMIT_%d_AFTER_%d_TYPE_%s", limit, after, strings.ToUpper(policyType))
}

func ParseCursor(value string) (uint, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("%w: invalid cursor", ErrInvalid)
	}
	return uint(parsed), nil
}
