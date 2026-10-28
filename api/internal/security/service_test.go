package security

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidatePolicyRequestAcceptsTypedIdentityRules(t *testing.T) {
	request := CreatePolicyRequest{
		Name: "Require reviewed identity",
		Type: "identity",
		Rules: []PolicyRule{
			{Field: "identity.role", Operator: "not_in", Value: "guest,anonymous", Effect: "require_review"},
			{Field: "identity.mfa_required", Operator: "equals", Value: "false", Effect: "alert"},
		},
	}
	if err := ValidatePolicyRequest(&request); err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
}

func TestValidatePolicyRequestRejectsUnknownFieldsAndOperators(t *testing.T) {
	tests := []CreatePolicyRequest{
		{
			Name:  "Unrestricted expression",
			Type:  "network",
			Rules: []PolicyRule{{Field: "network.raw_query", Operator: "equals", Value: "*", Effect: "allow"}},
		},
		{
			Name:  "Wrong typed operator",
			Type:  "workload",
			Rules: []PolicyRule{{Field: "workload.image_signed", Operator: "greater_than", Value: "0", Effect: "deny"}},
		},
		{
			Name:  "Unsupported type",
			Type:  "vendor_expression",
			Rules: []PolicyRule{{Field: "x", Operator: "equals", Value: "x", Effect: "deny"}},
		},
	}
	for _, test := range tests {
		if err := ValidatePolicyRequest(&test); !errors.Is(err, ErrInvalid) {
			t.Errorf("expected ErrInvalid for request type %q, got %v", test.Type, err)
		}
	}
}

func TestValidatePolicyRequestBoundsInputs(t *testing.T) {
	request := CreatePolicyRequest{
		Name:  strings.Repeat("a", 101),
		Type:  "identity",
		Rules: []PolicyRule{{Field: "identity.role", Operator: "equals", Value: "operator", Effect: "alert"}},
	}
	if err := ValidatePolicyRequest(&request); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected overlong name to be rejected, got %v", err)
	}

	request.Name = "Policy"
	request.Rules = make([]PolicyRule, maxPolicyRules+1)
	if err := ValidatePolicyRequest(&request); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected too many rules to be rejected, got %v", err)
	}
}

func TestPolicyRolePermissionsDefaultToDeny(t *testing.T) {
	for _, roles := range [][2]string{
		{"viewer", "viewer"},
		{"owner", "viewer"},
		{"viewer", "deployer"},
		{"member", "owner"},
	} {
		if canManagePolicies(roles[0], roles[1]) {
			t.Errorf("unexpected policy write access for roles %v", roles)
		}
	}
	if !canManagePolicies("owner", "deployer") {
		t.Fatal("tenant owner and project deployer should create drafts")
	}
	if !canManagePolicies("admin", "admin") {
		t.Fatal("tenant and project admins should create drafts")
	}
}

func TestAuditReadRequiresElevatedScopedRole(t *testing.T) {
	if canReadSecurityAudit("owner", "viewer") {
		t.Fatal("viewer must not read security audit records")
	}
	if !canReadSecurityAudit("admin", "security_auditor") {
		t.Fatal("security auditor in an admin tenant should read scoped audit records")
	}
	if canReadSecurityAudit("viewer", "security_auditor") {
		t.Fatal("project auditor without elevated tenant membership must be denied")
	}
}

func TestParseCursor(t *testing.T) {
	if cursor, err := ParseCursor(""); err != nil || cursor != 0 {
		t.Fatalf("empty cursor: got %d, %v", cursor, err)
	}
	if cursor, err := ParseCursor("42"); err != nil || cursor != 42 {
		t.Fatalf("valid cursor: got %d, %v", cursor, err)
	}
	for _, invalid := range []string{"0", "-1", "abc", "18446744073709551616"} {
		if _, err := ParseCursor(invalid); !errors.Is(err, ErrInvalid) {
			t.Errorf("expected invalid cursor %q to be rejected, got %v", invalid, err)
		}
	}
}

func TestAuditQueryReasonContainsOnlyBoundedFilterIntent(t *testing.T) {
	reason := auditQueryReason(50, 123, "identity")
	if reason != "LIMIT_50_AFTER_123_TYPE_IDENTITY" {
		t.Fatalf("unexpected normalized audit intent: %q", reason)
	}
	if strings.Contains(reason, "SELECT") || time.Now().IsZero() {
		t.Fatalf("unexpected raw query detail in audit intent: %q", reason)
	}
}
