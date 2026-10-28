package api

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"atonixcorp/cloud/compute/controlplane"
	"golang.org/x/crypto/ssh"
)

const (
	testAPIToken  = "test-api-token-0123456789-abcdefgh"
	testNodeToken = "test-node-token-0123456789-abcdefgh"
)

func newTestServer(t *testing.T) (*Server, *controlplane.FileStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "instances.json")
	store, err := controlplane.OpenFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddSSHKey(controlplane.SSHKey{
		ID: "key_test", Name: "test-key", Fingerprint: "SHA256:test", KeyType: "ssh-ed25519", Available: true,
	}); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(controlplane.DefaultCatalog(), store, testAPIToken, testNodeToken)
	if err != nil {
		t.Fatal(err)
	}
	return server, store, path
}

func TestCatalogPlanFilters(t *testing.T) {
	server, _, _ := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/compute/plans?region=jhb-1&type=general&compute_type=shared", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body)
	}
	var plans []controlplane.Plan
	if err := json.Unmarshal(response.Body.Bytes(), &plans); err != nil {
		t.Fatal(err)
	}
	if len(plans) != 4 {
		t.Fatalf("expected 4 filtered plans, got %d", len(plans))
	}
	for _, plan := range plans {
		if plan.Region != "jhb-1" || plan.ComputeType != "shared" || plan.Family != "general" {
			t.Fatalf("plan filter returned unrelated plan: %+v", plan)
		}
	}
}

func TestCreateInstancesPersistsQuantityAndAuthoritativePrices(t *testing.T) {
	server, _, path := newTestServer(t)
	requestBody := `{
		"name":"worker",
		"region":"jhb-1",
		"compute_type":"shared",
		"plan_id":"gp-2c-4g",
		"image_id":"ubuntu-24-04",
		"ssh_key_ids":["key_test"],
		"network_id":"vpc-default-jhb-1",
		"public_ipv4":true,
		"public_ipv6":true,
		"quantity":3,
		"volumes":[{"name":"data","type":"nvme_ssd","size_gb":100}],
		"tags":["production","worker"]
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances", strings.NewReader(requestBody))
	request.Header.Set("Authorization", "Bearer "+testAPIToken)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body)
	}
	var result struct {
		Status    string                  `json:"status"`
		Instances []controlplane.Instance `json:"instances"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "provisioning" || len(result.Instances) != 3 {
		t.Fatalf("unexpected deployment response: %+v", result)
	}
	for i, instance := range result.Instances {
		expectedName := "worker-" + []string{"01", "02", "03"}[i]
		if instance.Name != expectedName || instance.Hostname != expectedName {
			t.Errorf("expected generated name and hostname %q, got %q and %q", expectedName, instance.Name, instance.Hostname)
		}
		if instance.Status != "provisioning" || instance.Provisioning[0].Status != "queued" {
			t.Errorf("instance should remain explicitly provisioning: %+v", instance)
		}
		if instance.MonthlyPrice != 66.4 {
			t.Errorf("expected backend monthly price 66.4, got %v", instance.MonthlyPrice)
		}
	}

	reopened, err := controlplane.OpenFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 3 {
		t.Fatalf("expected 3 persisted instances, got %d", len(persisted))
	}
	keys, err := reopened.ListSSHKeys()
	if err != nil || len(keys) != 1 {
		t.Fatalf("expected SSH key to persist, got %d keys and error %v", len(keys), err)
	}
}

func TestCreateInstancesRequiresBearerTokenAndValidCatalogSelection(t *testing.T) {
	server, _, _ := newTestServer(t)
	body := `{"name":"api-01","region":"cpt-1","compute_type":"shared","plan_id":"gp-2c-4g","image_id":"ubuntu-24-04","ssh_key_ids":["key_test"]}`

	request := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances", strings.NewReader(body))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated request to return 401, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAPIToken)
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected unsupported network region/plan selection to return 422, got %d: %s", response.Code, response.Body)
	}
}

func TestCreateInstancesRequiresSSHAndDNSHostname(t *testing.T) {
	server, _, _ := newTestServer(t)
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing SSH key",
			body: `{"name":"api-01","region":"jhb-1","compute_type":"shared","plan_id":"gp-2c-4g","image_id":"ubuntu-24-04"}`,
		},
		{
			name: "invalid hostname",
			body: `{"name":"api-01","hostname":"api_node","region":"jhb-1","compute_type":"shared","plan_id":"gp-2c-4g","image_id":"ubuntu-24-04","ssh_key_ids":["key_test"]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer "+testAPIToken)
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, response.Code, response.Body)
			}
		})
	}
}

func TestNodeStatusRequiresOrderedStepsBeforeRunning(t *testing.T) {
	server, store, _ := newTestServer(t)
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances", bytes.NewBufferString(
		`{"name":"api-01","region":"jhb-1","compute_type":"shared","plan_id":"gp-2c-4g","image_id":"ubuntu-24-04","ssh_key_ids":["key_test"]}`,
	))
	createRequest.Header.Set("Authorization", "Bearer "+testAPIToken)
	createResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("create failed: %d %s", createResponse.Code, createResponse.Body)
	}
	var created struct {
		Instances []controlplane.Instance `json:"instances"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	instanceID := created.Instances[0].ID

	runningUpdate := `{"status":"running","step":"health_checks","step_status":"completed","node_id":"node-01"}`
	updateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances/"+instanceID+"/status", strings.NewReader(runningUpdate))
	updateRequest.Header.Set("Authorization", "Bearer "+testNodeToken)
	updateResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusConflict {
		t.Fatalf("expected out-of-order status update to fail, got %d: %s", updateResponse.Code, updateResponse.Body)
	}

	instance, err := store.Get(instanceID)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range instance.Provisioning {
		body, err := json.Marshal(map[string]string{
			"status":      "provisioning",
			"step":        step.Name,
			"step_status": "completed",
		})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances/"+instanceID+"/status", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+testNodeToken)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("could not complete step %q: %d %s", step.Name, response.Code, response.Body)
		}
	}

	updateRequest = httptest.NewRequest(http.MethodPost, "/api/v1/compute/instances/"+instanceID+"/status", strings.NewReader(runningUpdate))
	updateRequest.Header.Set("Authorization", "Bearer "+testNodeToken)
	updateResponse = httptest.NewRecorder()
	server.Handler().ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected valid completion to return 200, got %d: %s", updateResponse.Code, updateResponse.Body)
	}
	var updated controlplane.Instance
	if err := json.Unmarshal(updateResponse.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status != "running" || updated.NodeID != "node-01" {
		t.Fatalf("expected running instance assigned to node, got %+v", updated)
	}
}

func TestCreateSSHKeyValidatesAndPersistsPublicKey(t *testing.T) {
	server, store, _ := newTestServer(t)
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{
		"name":       "workstation",
		"public_key": string(ssh.MarshalAuthorizedKey(key)),
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/compute/ssh-keys", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAPIToken)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/compute/ssh-keys", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAPIToken)
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected duplicate key to return 409, got %d: %s", response.Code, response.Body)
	}

	keys, err := store.ListSSHKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[1].Fingerprint == "" {
		t.Fatalf("expected stored SSH key and fingerprint, got %+v", keys)
	}
}
