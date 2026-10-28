package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atonixcorp/api/internal/models"
)

func TestPrometheusAdapterQueriesScopedNormalizedSeries(t *testing.T) {
	const serviceToken = "server-side-test-token"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/query" {
			t.Errorf("unexpected provider path %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer "+serviceToken {
			t.Errorf("configured service credential was not forwarded")
		}
		query := request.URL.Query().Get("query")
		for _, required := range []string{
			"atonix_resource_cpu_utilization_ratio",
			`tenant_id="1"`,
			`project_id="2"`,
			`resource_type="storage.volume"`,
			`resource_id="volume-123"`,
		} {
			if !strings.Contains(query, required) {
				t.Errorf("PromQL query is missing scope label %q: %s", required, query)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []any{
					map[string]any{
						"metric": map[string]string{
							"__name__":  "atonix_resource_cpu_utilization_ratio",
							"tenant_id": "1", "project_id": "2",
							"resource_type": "storage.volume", "resource_id": "volume-123",
						},
						"value": []any{float64(1793200000), "0.61"},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &prometheusAdapter{
		baseURL: server.URL,
		token:   serviceToken,
		client:  server.Client(),
	}
	metrics, err := adapter.Metrics(context.Background(), models.ObservabilityResource{
		TenantID: 1, ProjectID: 2, ResourceType: "storage.volume", ResourceID: "volume-123",
	})
	if err != nil {
		t.Fatalf("query Prometheus-compatible adapter: %v", err)
	}
	if len(metrics) != 1 || metrics[0].Name != "atonix_resource_cpu_utilization_ratio" ||
		metrics[0].Value != 61 || metrics[0].Unit != "%" {
		t.Fatalf("unexpected normalized metric result: %+v", metrics)
	}
}

func TestPrometheusAdapterReadinessReportsConnectedOnlyOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/-/ready" {
			t.Errorf("unexpected readiness path %q", request.URL.Path)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	adapter := &prometheusAdapter{baseURL: server.URL, client: server.Client()}
	state, err := adapter.Availability(context.Background())
	if err != nil || state != "connected" {
		t.Fatalf("healthy provider should be reported connected, got %q, %v", state, err)
	}
}

func TestPrometheusAdapterRejectsURLCredentials(t *testing.T) {
	t.Setenv("OBSERVABILITY_PROMETHEUS_URL", "https://monitor:secret@example.invalid")
	if _, err := newPrometheusAdapterFromEnvironment(); err == nil {
		t.Fatal("Prometheus URL with embedded credentials must be rejected")
	}
}
