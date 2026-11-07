package openstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplicationCredentialAuthDiscoversProjectEndpoints(t *testing.T) {
	var authBody map[string]any
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v3/auth/tokens" || request.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&authBody); err != nil {
			t.Errorf("decode Keystone request: %v", err)
		}
		writer.Header().Set("X-Subject-Token", "test-token")
		writer.WriteHeader(http.StatusCreated)
		body := `{"token":{"expires_at":"` +
			time.Now().Add(time.Hour).UTC().Format(time.RFC3339) +
			`","project":{"id":"project-id"},"catalog":[` +
			`{"type":"compute","endpoints":[{"region_id":"RegionOne","interface":"public","url":"` + server.URL + `/compute/v2.1/%(project_id)s"}]},` +
			`{"type":"volumev3","endpoints":[{"region_id":"RegionOne","interface":"public","url":"` + server.URL + `/volume/v3/%(project_id)s"}]},` +
			`{"type":"network","endpoints":[{"region_id":"RegionOne","interface":"public","url":"` + server.URL + `/network/v2.0"}]}` +
			`]}}`
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	client, err := NewClient(AuthConfig{
		AuthURL: server.URL + "/v3", ApplicationCredentialID: "credential-id",
		ApplicationCredentialSecret: "credential-secret", ProjectID: "project-id",
		Region: "RegionOne", AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	if _, err := client.ensureToken(context.Background()); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if client.ComputeEndpoint != server.URL+"/compute/v2.1/project-id" ||
		client.VolumeEndpoint != server.URL+"/volume/v3/project-id" ||
		client.NetworkEndpoint != server.URL+"/network/v2.0" {
		t.Fatalf("unexpected discovered endpoints: compute=%q volume=%q network=%q",
			client.ComputeEndpoint, client.VolumeEndpoint, client.NetworkEndpoint)
	}
	identity, ok := authBody["auth"].(map[string]any)["identity"].(map[string]any)
	if !ok || identity["methods"] == nil || identity["application_credential"] == nil {
		t.Fatalf("Keystone request did not use application-credential auth: %#v", authBody)
	}
	if _, hasScope := authBody["auth"].(map[string]any)["scope"]; hasScope {
		t.Fatalf("application-credential request must not add password-auth scope: %#v", authBody)
	}
}

func TestNewClientRequiresExplicitOptInForHTTP(t *testing.T) {
	_, err := NewClient(AuthConfig{
		AuthURL:                     "http://192.168.178.131/identity/v3",
		ApplicationCredentialID:     "credential-id",
		ApplicationCredentialSecret: "credential-secret",
		ProjectID:                   "project-id",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("NewClient() error = %v, want explicit HTTP rejection", err)
	}
}
