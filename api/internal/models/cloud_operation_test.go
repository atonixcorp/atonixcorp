package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCloudOperationJSONHidesProviderIdentifiers(t *testing.T) {
	payload, err := json.Marshal(CloudOperation{
		ID: "atonix-operation", Provider: "openstack",
		ProviderOperationID: "provider-operation-id",
	})
	if err != nil {
		t.Fatalf("marshal operation: %v", err)
	}
	for _, privateValue := range []string{"openstack", "provider-operation-id", `"provider"`} {
		if strings.Contains(string(payload), privateValue) {
			t.Fatalf("operation response leaked %q: %s", privateValue, payload)
		}
	}
	if !strings.Contains(string(payload), `"id":"atonix-operation"`) {
		t.Fatalf("operation response omitted its AtonixCorp ID: %s", payload)
	}
}

func TestContainerAuditEventJSONHidesBackendErrorDetails(t *testing.T) {
	payload, err := json.Marshal(ContainerDeploymentEvent{
		ID: 1, ErrorDetails: "Kubernetes endpoint https://cluster.example failed",
	})
	if err != nil {
		t.Fatalf("marshal container event: %v", err)
	}
	if strings.Contains(string(payload), "Kubernetes endpoint") ||
		strings.Contains(string(payload), "cluster.example") {
		t.Fatalf("container event leaked backend error details: %s", payload)
	}
}
