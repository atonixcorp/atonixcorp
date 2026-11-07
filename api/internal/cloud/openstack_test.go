package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

func TestOpenStackConfigFromEnvRequiresExplicitHTTPOptIn(t *testing.T) {
	t.Setenv("OPENSTACK_AUTH_URL", "http://192.168.178.131/identity/v3")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_ID", "development-id")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET", "development-secret")
	t.Setenv("OPENSTACK_REGION", "")
	t.Setenv("OPENSTACK_INTERFACE", "")
	t.Setenv("OPENSTACK_ALLOW_INSECURE_HTTP", "false")

	if _, _, err := OpenStackConfigFromEnv(); err == nil {
		t.Fatal("OpenStackConfigFromEnv() accepted HTTP without the development opt-in")
	}

	t.Setenv("OPENSTACK_ALLOW_INSECURE_HTTP", "true")
	cfg, configured, err := OpenStackConfigFromEnv()
	if err != nil {
		t.Fatalf("OpenStackConfigFromEnv() error = %v", err)
	}
	if !configured || cfg.Region != "RegionOne" || cfg.Interface != "public" || !cfg.AllowInsecureHTTP {
		t.Fatalf("unexpected development configuration: configured=%v cfg=%+v", configured, cfg)
	}
}

func TestOpenStackConfigFromEnvRejectsUnknownInterface(t *testing.T) {
	t.Setenv("OPENSTACK_AUTH_URL", "https://keystone.example.test/v3")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_ID", "credential-id")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET", "credential-secret")
	t.Setenv("OPENSTACK_ALLOW_INSECURE_HTTP", "false")
	t.Setenv("OPENSTACK_INTERFACE", "private")

	if _, configured, err := OpenStackConfigFromEnv(); err == nil || configured {
		t.Fatalf("OpenStackConfigFromEnv() = configured %v, error %v; want invalid interface error",
			configured, err)
	}
}

func TestOpenStackConfigFromEnvRejectsIncompleteCredentials(t *testing.T) {
	t.Setenv("OPENSTACK_AUTH_URL", "https://keystone.example.test/v3")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_ID", "credential-id")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET", "")
	t.Setenv("OPENSTACK_ALLOW_INSECURE_HTTP", "false")

	if _, configured, err := OpenStackConfigFromEnv(); err == nil || configured {
		t.Fatalf("OpenStackConfigFromEnv() = configured %v, error %v; want incomplete config error", configured, err)
	}
}

func TestOpenStackConfigFromEnvAcceptsFactoryWithoutGlobalCredentials(t *testing.T) {
	t.Setenv("OPENSTACK_AUTH_URL", "https://keystone.example.test/v3")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_ID", "")
	t.Setenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET", "")
	t.Setenv("OPENSTACK_REGION", "")
	t.Setenv("OPENSTACK_INTERFACE", "")
	t.Setenv("OPENSTACK_ALLOW_INSECURE_HTTP", "false")
	cfg, configured, err := OpenStackConfigFromEnv()
	if err != nil {
		t.Fatalf("OpenStackConfigFromEnv(): %v", err)
	}
	if !configured || cfg.AuthURL == "" || cfg.ApplicationCredentialID != "" {
		t.Fatalf("provider factory config not loaded: configured=%v cfg=%+v", configured, cfg)
	}
}

func TestNewOpenStackProviderReturnsUnconnectedAccountFactory(t *testing.T) {
	provider, err := NewOpenStackProvider(context.Background(), OpenStackConfig{
		AuthURL: "https://keystone.example.test/v3", Region: "RegionOne", Interface: "public",
	})
	if err != nil {
		t.Fatalf("NewOpenStackProvider(): %v", err)
	}
	if provider.ComputeAvailable() || provider.Provider != nil || provider.Region() != "RegionOne" {
		t.Fatalf("unexpected account-scoped provider factory: %+v", provider)
	}
}

func TestVerifyCredentialProjectChecksKeystoneTokenScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/v3/auth/tokens" {
			t.Errorf("unexpected Keystone scope verification request %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("X-Auth-Token") != "token" ||
			request.Header.Get("X-Subject-Token") != "token" {
			t.Errorf("scope verification missing token headers")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"token":{"project":{"id":"project-123"}}}`))
	}))
	defer server.Close()

	cfg := OpenStackConfig{AuthURL: server.URL + "/v3", ProjectID: "project-123"}
	if err := verifyCredentialProject(context.Background(), cfg, "token"); err != nil {
		t.Fatalf("verifyCredentialProject(): %v", err)
	}
	cfg.ProjectID = "another-project"
	if err := verifyCredentialProject(context.Background(), cfg, "token"); err == nil ||
		!strings.Contains(err.Error(), "not scoped") {
		t.Fatalf("verifyCredentialProject() error = %v, want scope mismatch", err)
	}
}

func TestVerifyCredentialProjectRejectsUnscopedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"token": map[string]any{}})
	}))
	defer server.Close()
	err := verifyCredentialProject(context.Background(), OpenStackConfig{
		AuthURL: server.URL + "/v3", ProjectID: "project-123",
	}, "token")
	if err == nil || !strings.Contains(err.Error(), "not scoped") {
		t.Fatalf("verifyCredentialProject() error = %v, want unscoped token rejection", err)
	}
}

func TestOpenStackCapabilitiesUseRegionalPublicCatalogEndpoints(t *testing.T) {
	catalog := []openStackCatalogService{
		{Type: "compute", Endpoints: []openStackCatalogEndpoint{
			{RegionID: "RegionOne", Interface: "public", URL: "https://nova.example.test"},
		}},
		{Type: "network", Endpoints: []openStackCatalogEndpoint{
			{Region: "RegionTwo", Interface: "public", URL: "https://neutron.example.test"},
		}},
		{Type: "image", Endpoints: []openStackCatalogEndpoint{
			{Region: "RegionOne", Interface: "internal", URL: "https://glance.example.test"},
		}},
		{Type: "volumev3", Endpoints: []openStackCatalogEndpoint{
			{Region: "RegionOne", Interface: "public", URL: "https://cinder.example.test"},
		}},
		{Type: "object-store", Endpoints: []openStackCatalogEndpoint{
			{Region: "RegionOne", Interface: "public", URL: "https://swift.example.test"},
		}},
	}

	capabilities := openStackCapabilities(catalog, "RegionOne", "public")
	if !capabilities.Compute || !capabilities.BlockStorage || !capabilities.ObjectStorage {
		t.Fatalf("available services were not discovered: %+v", capabilities)
	}
	if capabilities.Network || capabilities.Image {
		t.Fatalf("services outside the configured region/interface were enabled: %+v", capabilities)
	}
}

func TestCatalogPageURLAddsOpenStackMarkerAndLimit(t *testing.T) {
	got := catalogPageURL("https://provider.example/v2/images?visibility=public", "image-500")
	if !strings.HasPrefix(got, "https://provider.example/v2/images?") ||
		!strings.Contains(got, "visibility=public") ||
		!strings.Contains(got, "limit=500") ||
		!strings.Contains(got, "marker=image-500") {
		t.Fatalf("catalogPageURL() = %q, want query preserved with limit and marker", got)
	}
}

func TestSecurityGroupRequestsAreScopedAndUseSafeTestRule(t *testing.T) {
	var groupCreated, ruleCreated bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Auth-Token") != "provider-token" {
			t.Errorf("request missing provider token")
		}
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v2.0/security-groups":
			if request.URL.Query().Get("project_id") != "openstack-project" {
				t.Errorf("security group list project_id = %q", request.URL.Query().Get("project_id"))
			}
			_, _ = writer.Write([]byte(`{"security_groups":[
				{"id":"owned","name":"owned","project_id":"openstack-project","security_group_rules":[
					{"id":"rule-owned","direction":"ingress","ethertype":"IPv4","protocol":"tcp","port_range_min":443,"port_range_max":443,"remote_ip_prefix":"192.0.2.0/24"}
				]},
				{"id":"foreign","name":"foreign","project_id":"another-project","security_group_rules":[]}
			]}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2.0/security-groups":
			var body struct {
				Group map[string]any `json:"security_group"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode group request: %v", err)
			}
			if body.Group["project_id"] != "openstack-project" || body.Group["name"] != "atonixcorp-test" {
				t.Errorf("unexpected security group create body: %+v", body.Group)
			}
			groupCreated = true
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"security_group":{"id":"created","name":"atonixcorp-test","description":"test","project_id":"openstack-project","security_group_rules":[]}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2.0/security-group-rules":
			var body struct {
				Rule map[string]any `json:"security_group_rule"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode security rule request: %v", err)
			}
			for key, expected := range map[string]any{
				"security_group_id": "created", "project_id": "openstack-project",
				"direction": "ingress", "ethertype": "IPv4", "protocol": "tcp",
				"port_range_min": float64(443), "port_range_max": float64(443),
				"remote_ip_prefix": "192.0.2.0/24",
			} {
				if body.Rule[key] != expected {
					t.Errorf("security rule %s = %v, want %v", key, body.Rule[key], expected)
				}
			}
			ruleCreated = true
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"security_group_rule":{"id":"created-rule","direction":"ingress","ethertype":"IPv4","protocol":"tcp","port_range_min":443,"port_range_max":443,"remote_ip_prefix":"192.0.2.0/24"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := &OpenStackProvider{
		Provider: &gophercloud.ProviderClient{
			TokenID: "provider-token", HTTPClient: *server.Client(),
		},
		Network:   &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "provider-token", HTTPClient: *server.Client()}, Endpoint: server.URL + "/v2.0/"},
		projectID: "openstack-project",
	}
	groups, err := provider.ListSecurityGroups(context.Background())
	if err != nil {
		t.Fatalf("ListSecurityGroups(): %v", err)
	}
	if len(groups) != 1 || groups[0].ID != "owned" || len(groups[0].Rules) != 1 {
		t.Fatalf("ListSecurityGroups() = %+v, want only mapped project groups", groups)
	}
	group, err := provider.CreateSecurityGroup(context.Background(), "atonixcorp-test")
	if err != nil {
		t.Fatalf("CreateSecurityGroup(): %v", err)
	}
	if group.ID != "created" || group.ProjectID != "openstack-project" ||
		len(group.Rules) != 1 || group.Rules[0].RemoteIPPrefix != "192.0.2.0/24" {
		t.Fatalf("CreateSecurityGroup() = %+v, want provider details and safe rule", group)
	}
	if !groupCreated || !ruleCreated {
		t.Fatalf("expected group and rule requests, got group=%v rule=%v", groupCreated, ruleCreated)
	}
}

func TestSecurityGroupCreationRemovesGroupWhenSafeRuleCreationFails(t *testing.T) {
	var groupCreated, groupDeleted bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v2.0/security-groups":
			groupCreated = true
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"security_group":{"id":"partial","name":"test","project_id":"openstack-project"}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2.0/security-group-rules":
			http.Error(writer, `{"NeutronError":{"message":"rule rejected"}}`, http.StatusBadRequest)
		case request.Method == http.MethodDelete && request.URL.Path == "/v2.0/security-groups/partial":
			groupDeleted = true
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	providerClient := &gophercloud.ProviderClient{
		TokenID: "provider-token", HTTPClient: *server.Client(),
	}
	provider := &OpenStackProvider{
		Network:   &gophercloud.ServiceClient{ProviderClient: providerClient, Endpoint: server.URL + "/v2.0/"},
		projectID: "openstack-project",
	}
	if _, err := provider.CreateSecurityGroup(context.Background(), "test"); !errors.Is(err, ErrProviderRequest) {
		t.Fatalf("CreateSecurityGroup() error = %v, want provider error", err)
	}
	if !groupCreated || !groupDeleted {
		t.Fatalf("partial security group cleanup: created=%v deleted=%v", groupCreated, groupDeleted)
	}
}

func TestProviderInstanceSeparatesFixedAndFloatingNovaAddresses(t *testing.T) {
	var server servers.Server
	err := json.Unmarshal([]byte(`{
		"id":"provider-instance-1",
		"status":"ACTIVE",
		"accessIPv4":"192.0.2.50",
		"addresses":{"private":[
			{"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"},
			{"addr":"192.0.2.50","OS-EXT-IPS:type":"floating"}
		]}
	}`), &server)
	if err != nil {
		t.Fatalf("decode Nova server fixture: %v", err)
	}
	instance := providerInstance(&server)
	if instance.ID != "provider-instance-1" || instance.PrivateIP != "10.0.0.8" ||
		instance.PublicIP != "192.0.2.50" || instance.Status != "ACTIVE" {
		t.Fatalf("providerInstance() = %+v, want Nova ID, status, fixed and floating addresses", instance)
	}
}
