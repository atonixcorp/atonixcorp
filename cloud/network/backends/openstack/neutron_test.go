package openstack

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"atnetwork/core"
)

func TestNeutronUsesApplicationCredentialAndIdempotentNetworkMarker(t *testing.T) {
	var server *httptest.Server
	var networkDescription string
	var subnetDescription string
	var subnetCreateCalls int
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/auth/tokens":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode Keystone request: %v", err)
			}
			auth, _ := body["auth"].(map[string]any)
			identity, _ := auth["identity"].(map[string]any)
			if identity["application_credential"] == nil {
				t.Errorf("Keystone request did not use an application credential: %#v", body)
			}
			if _, ok := auth["scope"]; ok {
				t.Errorf("application-credential auth unexpectedly scoped a password: %#v", body)
			}
			writer.Header().Set("X-Subject-Token", "test-token")
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			catalog := []any{map[string]any{
				"type": "network",
				"endpoints": []any{map[string]any{
					"region_id": "RegionOne", "interface": "public",
					"url": server.URL + "/networking",
				}},
			}}
			token := map[string]any{
				"expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
				"project":    map[string]string{"id": "shared-project"},
				"catalog":    catalog,
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"token": token})
		case request.URL.Path == "/networking/v2.0/networks" && request.Method == http.MethodGet:
			if request.Header.Get("X-Auth-Token") != "test-token" {
				t.Errorf("Neutron request missing auth token")
			}
			networks := []map[string]string{}
			if networkDescription != "" {
				network := map[string]string{
					"id": "provider-network", "description": networkDescription,
					"project_id": "shared-project", "status": "ACTIVE",
				}
				networks = append(networks, network)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"networks": networks})
		case request.URL.Path == "/networking/v2.0/subnets" && request.Method == http.MethodGet:
			subnets := []map[string]string{}
			if subnetDescription != "" {
				subnets = append(subnets, map[string]string{
					"id": "provider-subnet", "description": subnetDescription, "status": "ACTIVE",
				})
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"subnets": subnets})
		case request.URL.Path == "/networking/v2.0/networks" && request.Method == http.MethodPost:
			var payload struct {
				Network struct {
					ProjectID   string `json:"project_id"`
					Description string `json:"description"`
				} `json:"network"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode Neutron create request: %v", err)
			}
			if payload.Network.ProjectID != "shared-project" {
				t.Errorf("Neutron project_id = %q, want shared project", payload.Network.ProjectID)
			}
			networkDescription = payload.Network.Description
			writer.WriteHeader(http.StatusCreated)
		case request.URL.Path == "/networking/v2.0/subnets" && request.Method == http.MethodPost:
			subnetCreateCalls++
			var payload struct {
				Subnet struct {
					NetworkID   string `json:"network_id"`
					Description string `json:"description"`
					CIDR        string `json:"cidr"`
				} `json:"subnet"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode Neutron subnet create request: %v", err)
			}
			if payload.Subnet.NetworkID != "provider-network" || payload.Subnet.CIDR != "10.5.0.0/24" {
				t.Errorf("Neutron subnet request = %+v", payload.Subnet)
			}
			subnetDescription = payload.Subnet.Description
			writer.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewNeutronClient(NeutronConfig{
		AuthURL: server.URL + "/v3", ApplicationCredentialID: "credential-id",
		ApplicationCredentialSecret: "credential-secret", ProjectID: "shared-project",
		Region: "RegionOne", AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewNeutronClient(): %v", err)
	}
	backend := NewNeutronBackend(client)
	network := core.Network{
		ID: "local-network", Name: "app-network", TenantID: "7", ProjectID: "9",
		CIDR: "10.5.0.0/24",
	}
	if err := backend.ApplyNetwork(network); err != nil {
		t.Fatalf("ApplyNetwork(): %v", err)
	}
	if err := backend.VerifyNetwork(network); err != nil {
		t.Fatalf("VerifyNetwork(): %v", err)
	}
	if !strings.HasPrefix(networkDescription, "atonixcorp:network:local-network:tenant:7:project:9") {
		t.Fatalf("network lacks AtonixCorp ownership marker: %q", networkDescription)
	}
	subnet := core.Subnet{
		ID: "local-subnet", NetworkID: network.ID, CIDR: "10.5.0.0/24",
		TenantID: network.TenantID, ProjectID: network.ProjectID,
	}
	if err := backend.ApplySubnet(subnet); err != nil {
		t.Fatalf("ApplySubnet(): %v", err)
	}
	if err := backend.VerifySubnet(subnet); err != nil {
		t.Fatalf("VerifySubnet(): %v", err)
	}
	retrySubnet := subnet
	retrySubnet.ID = "recovered-local-subnet"
	if err := backend.ApplySubnet(retrySubnet); err != nil {
		t.Fatalf("ApplySubnet() for provider subnet recovery: %v", err)
	}
	if err := backend.VerifySubnet(retrySubnet); err != nil {
		t.Fatalf("VerifySubnet() for provider subnet recovery: %v", err)
	}
	if subnetCreateCalls != 1 {
		t.Fatalf("Neutron created %d subnets, want one", subnetCreateCalls)
	}
	inspected, err := backend.InspectNetwork(network)
	if err != nil {
		t.Fatalf("InspectNetwork(): %v", err)
	}
	if inspected.ProviderID != "provider-network" || inspected.Status != "ACTIVE" ||
		inspected.ProviderSubnetID != "provider-subnet" ||
		inspected.ProviderSubnetStatus != "ACTIVE" {
		t.Fatalf("unexpected provider network details: %+v", inspected)
	}
}

type testCloudAccountSource struct {
	account core.CloudAccount
}

func (s testCloudAccountSource) GetCloudAccountForTenant(tenantID string) (core.CloudAccount, error) {
	if tenantID != "7" {
		return core.CloudAccount{}, errors.New("unexpected tenant")
	}
	return s.account, nil
}

func TestScopedNeutronUsesMappedAccount(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{3}, gcm.NonceSize())
	encryptedSecret := gcm.Seal(nonce, nonce, []byte("mapped-secret"), nil)
	var server *httptest.Server
	var networkDescription string
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v3/auth/tokens":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode Keystone request: %v", err)
			}
			auth, _ := payload["auth"].(map[string]any)
			identity, _ := auth["identity"].(map[string]any)
			applicationCredential, _ := identity["application_credential"].(map[string]any)
			if applicationCredential["id"] != "mapped-credential" ||
				applicationCredential["secret"] != "mapped-secret" {
				t.Errorf("Neutron did not use the mapped application credential")
			}
			writer.Header().Set("X-Subject-Token", "mapped-token")
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(writer).Encode(map[string]any{"token": map[string]any{
				"expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
				"project":    map[string]string{"id": "mapped-project"},
				"catalog": []any{map[string]any{
					"type": "network",
					"endpoints": []any{map[string]any{
						"region_id": "RegionOne", "interface": "public",
						"url": server.URL + "/network/v2.0",
					}},
				}},
			}})
		case request.URL.Path == "/network/v2.0/networks" && request.Method == http.MethodGet:
			if request.URL.Query().Get("project_id") != "mapped-project" {
				t.Errorf("network listing used project %q", request.URL.Query().Get("project_id"))
			}
			networks := []map[string]string{}
			if networkDescription != "" {
				networks = append(networks, map[string]string{
					"id": "mapped-provider-network", "description": networkDescription,
					"project_id": "mapped-project", "status": "ACTIVE",
				})
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"networks": networks})
		case request.URL.Path == "/network/v2.0/networks" && request.Method == http.MethodPost:
			var payload struct {
				Network struct {
					ProjectID   string `json:"project_id"`
					Description string `json:"description"`
				} `json:"network"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode Neutron create request: %v", err)
			}
			if payload.Network.ProjectID != "mapped-project" {
				t.Errorf("Neutron create used project %q", payload.Network.ProjectID)
			}
			networkDescription = payload.Network.Description
			writer.WriteHeader(http.StatusCreated)
		case request.URL.Path == "/network/v2.0/subnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"subnets": []any{}})
		default:
			t.Errorf("unexpected request %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend, err := NewScopedNeutronBackend(testCloudAccountSource{account: core.CloudAccount{
		ProjectID: "mapped-project", CredentialID: "mapped-credential",
		EncryptedSecret: encryptedSecret, Region: "RegionOne",
	}}, NeutronConfig{
		AuthURL: server.URL + "/v3", Region: "RegionOne", Interface: "public",
		AllowInsecureHTTP: true,
	}, key)
	if err != nil {
		t.Fatalf("NewScopedNeutronBackend(): %v", err)
	}
	network := core.Network{
		ID: "local-network", Name: "mapped-network", CIDR: "10.8.0.0/24",
		TenantID: "7", ProjectID: "9",
	}
	if err := backend.ApplyNetwork(network); err != nil {
		t.Fatalf("ApplyNetwork(): %v", err)
	}
	if err := backend.VerifyNetwork(network); err != nil {
		t.Fatalf("VerifyNetwork(): %v", err)
	}
	inspected, err := backend.InspectNetwork(network)
	if err != nil {
		t.Fatalf("InspectNetwork(): %v", err)
	}
	if inspected.ProviderID != "mapped-provider-network" || inspected.Status != "ACTIVE" {
		t.Fatalf("unexpected mapped-project network details: %+v", inspected)
	}
}

func TestNewNeutronClientRequiresExplicitHTTPOptIn(t *testing.T) {
	_, err := NewNeutronClient(NeutronConfig{
		AuthURL:                     "http://192.168.178.131/identity/v3",
		ApplicationCredentialID:     "credential-id",
		ApplicationCredentialSecret: "credential-secret",
		ProjectID:                   "shared-project",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("NewNeutronClient() error = %v, want explicit HTTP rejection", err)
	}
}
