package containers

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func validSpec() DeploymentSpec {
	return DeploymentSpec{
		Name: "web", Environment: "development", Image: "registry.example/app:1.0",
		Replicas: 2, CPURequest: "100m", CPULimit: "1",
		MemoryRequest: "128Mi", MemoryLimit: "1Gi",
		Ports: []Port{{Container: 8080}},
		EnvironmentVars: []EnvironmentVariable{
			{Name: "MODE", Value: "test"},
			{Name: "API_TOKEN", Value: "never-return-this", Secret: true},
		},
		Volumes:        []Volume{{Name: "data", SizeGiB: 10, MountPath: "/var/lib/app"}},
		ReadinessProbe: &Probe{Path: "/ready", Port: 8080},
	}
}

func TestValidateSpecNormalizesAndRejectsUnsafeInput(t *testing.T) {
	spec := validSpec()
	spec.Name = " Web "
	if err := ValidateSpec(&spec); err != nil {
		t.Fatal(err)
	}
	if spec.Name != "web" || spec.Ports[0].Protocol != string(corev1.ProtocolTCP) ||
		spec.Volumes[0].AccessMode != string(corev1.ReadWriteOnce) {
		t.Fatalf("defaults were not applied: %+v", spec)
	}

	spec = validSpec()
	spec.Environment = "production"
	if err := ValidateSpec(&spec); err == nil || !strings.Contains(err.Error(), "immutable sha256 digest") {
		t.Fatalf("production tag must be rejected, got %v", err)
	}

	spec = validSpec()
	spec.Volumes[0].MountPath = "/proc/secret"
	if err := ValidateSpec(&spec); err == nil {
		t.Fatal("expected protected system mount path to be rejected")
	}
}

func TestHideSecretValuesKeepsOnlyRedactedMetadata(t *testing.T) {
	spec, secrets := hideSecretValues(validSpec())
	if len(secrets) != 1 || secrets["API_TOKEN"] != "never-return-this" {
		t.Fatalf("secret value was not captured: %#v", secrets)
	}
	for _, variable := range spec.EnvironmentVars {
		if variable.Secret && variable.Value != "" {
			t.Fatal("secret value remained in serializable deployment spec")
		}
	}
}

func TestPreserveSecretValuesDuringConfigurationUpdate(t *testing.T) {
	spec := validSpec()
	spec.EnvironmentVars[1].Value = ""
	preserveSecretValues(&spec, validSpec(), map[string]string{"API_TOKEN": "previous-secret"})
	if spec.EnvironmentVars[1].Value != "previous-secret" {
		t.Fatal("omitted secret input must retain the currently encrypted value")
	}

	spec.EnvironmentVars[1].Value = "rotated-secret"
	preserveSecretValues(&spec, validSpec(), map[string]string{"API_TOKEN": "previous-secret"})
	if spec.EnvironmentVars[1].Value != "rotated-secret" {
		t.Fatal("explicit secret input must replace the current value")
	}
}

func TestNamespaceNameIsTenantAndProjectScoped(t *testing.T) {
	first := NamespaceName(9, 4)
	second := NamespaceName(9, 5)
	if first == second || first != "atonix-t9-p4" {
		t.Fatalf("unexpected namespace isolation names: %q, %q", first, second)
	}
}
