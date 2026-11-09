package gatewayapi

import (
	"context"
	"strings"
	"testing"
)

func TestEnsureDefaultLocalAdminRequiresExplicitCredentials(t *testing.T) {
	t.Setenv("ORCASTACK_AUTH_LOCAL_ENABLED", "true")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_USERNAME", "")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_EMAIL", "")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_PASSWORD", "")

	err := ensureConfiguredLocalAdminAccount(context.Background())
	if err == nil || !strings.Contains(err.Error(), "must be configured") {
		t.Fatalf("missing local admin credentials error = %v", err)
	}
}

func TestEnsureDefaultLocalAdminRejectsWeakPassword(t *testing.T) {
	t.Setenv("ORCASTACK_AUTH_LOCAL_ENABLED", "true")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_USERNAME", "administrator")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_EMAIL", "admin@example.invalid")
	t.Setenv("ORCASTACK_AUTH_LOCAL_ADMIN_PASSWORD", "short")

	err := ensureConfiguredLocalAdminAccount(context.Background())
	if err == nil || !strings.Contains(err.Error(), "between 12 and 72 bytes") {
		t.Fatalf("weak local admin password error = %v", err)
	}
}
