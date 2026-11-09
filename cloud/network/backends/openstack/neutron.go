package openstack

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"atnetwork/core"
)

type NeutronConfig struct {
	AuthURL                     string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	ProjectID                   string
	Region                      string
	Interface                   string
	AllowInsecureHTTP           bool
}

type NeutronClient struct {
	config     NeutronConfig
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
	endpoint   string
}

func NewNeutronClient(config NeutronConfig) (*NeutronClient, error) {
	config.AuthURL = strings.TrimRight(strings.TrimSpace(config.AuthURL), "/")
	config.ApplicationCredentialID = strings.TrimSpace(config.ApplicationCredentialID)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.Region = strings.TrimSpace(config.Region)
	config.Interface = strings.TrimSpace(config.Interface)
	if config.Region == "" {
		config.Region = "RegionOne"
	}
	if config.Interface == "" {
		config.Interface = "public"
	}
	if config.AuthURL == "" || config.ApplicationCredentialID == "" ||
		strings.TrimSpace(config.ApplicationCredentialSecret) == "" || config.ProjectID == "" {
		return nil, errors.New("Neutron requires Keystone URL, application credentials, and project ID")
	}
	if err := validateEndpoint(config.AuthURL, config.AllowInsecureHTTP); err != nil {
		return nil, fmt.Errorf("Keystone URL: %w", err)
	}
	return &NeutronClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

type NeutronBackend struct {
	client        *NeutronClient
	accounts      CloudAccountSource
	baseConfig    NeutronConfig
	encryptionKey []byte
}

type providerNetwork struct {
	ID        string
	ProjectID string
	Status    string
}

type providerSubnet struct {
	ID     string
	Status string
}

type CloudAccountSource interface {
	GetCloudAccountForTenant(string) (core.CloudAccount, error)
}

func NewNeutronBackend(client *NeutronClient) *NeutronBackend {
	return &NeutronBackend{client: client}
}

func NewScopedNeutronBackend(
	accounts CloudAccountSource,
	config NeutronConfig,
	encryptionKey []byte,
) (*NeutronBackend, error) {
	if accounts == nil || len(encryptionKey) != 32 {
		return nil, errors.New("mapped OpenStack accounts and a 32-byte encryption key are required")
	}
	config.AuthURL = strings.TrimRight(strings.TrimSpace(config.AuthURL), "/")
	config.Region = strings.TrimSpace(config.Region)
	config.Interface = strings.TrimSpace(config.Interface)
	if config.Region == "" {
		config.Region = "RegionOne"
	}
	if config.Interface == "" {
		config.Interface = "public"
	}
	if err := validateEndpoint(config.AuthURL, config.AllowInsecureHTTP); err != nil {
		return nil, fmt.Errorf("Keystone URL: %w", err)
	}
	return &NeutronBackend{
		accounts: accounts, baseConfig: config,
		encryptionKey: append([]byte(nil), encryptionKey...),
	}, nil
}

func (b *NeutronBackend) clientForTenant(tenantID string) (*NeutronClient, error) {
	if b.accounts == nil {
		if b.client == nil {
			return nil, errors.New("Neutron client is not configured")
		}
		return b.client, nil
	}
	account, err := b.accounts.GetCloudAccountForTenant(tenantID)
	if err != nil {
		return nil, err
	}
	if account.ScopeID == "" || len(account.EncryptedSecret) == 0 || account.Provider != "" && account.Provider != "openstack" {
		return nil, errors.New("mapped OpenStack account is incomplete")
	}
	secret, err := decryptAccountSecret(b.encryptionKey, account.EncryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt mapped OpenStack credential: %w", err)
	}
	config := b.baseConfig
	config.ProjectID = account.ScopeID
	switch account.CredentialVersion {
	case 0:
		config.ApplicationCredentialID = account.LegacyCredentialID
		config.ApplicationCredentialSecret = string(secret)
	case 1:
		var credential struct {
			ProjectID        string `json:"project_id"`
			CredentialID     string `json:"credential_id"`
			CredentialSecret string `json:"credential_secret"`
			Region           string `json:"region"`
		}
		if err := json.Unmarshal(secret, &credential); err != nil {
			return nil, errors.New("invalid OpenStack credential payload")
		}
		if strings.TrimSpace(credential.ProjectID) != account.ScopeID || strings.TrimSpace(credential.Region) != account.Region {
			return nil, errors.New("OpenStack credential does not match the mapped account")
		}
		config.ApplicationCredentialID = credential.CredentialID
		config.ApplicationCredentialSecret = credential.CredentialSecret
	default:
		return nil, errors.New("unsupported cloud credential version")
	}
	if account.Region != "" {
		config.Region = account.Region
	}
	return NewNeutronClient(config)
}

func decryptAccountSecret(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("encrypted cloud credential is too short")
	}
	nonce, data := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, data, nil)
}

func (b *NeutronBackend) ApplyNetwork(network core.Network) error {
	if network.External {
		return errors.New("external Neutron networks must be provisioned by an OpenStack administrator")
	}
	client, err := b.clientForTenant(network.TenantID)
	if err != nil {
		return err
	}
	existing, err := b.findNetwork(client, network.ID)
	if err != nil {
		return err
	}
	if existing.ID != "" {
		return nil
	}
	payload := map[string]any{"network": map[string]any{
		"name":           network.Name,
		"project_id":     client.config.ProjectID,
		"admin_state_up": true,
		"description":    resourceMarker("network", network.ID, network.TenantID, network.ProjectID),
	}}
	return client.request(http.MethodPost, "/networks", payload, http.StatusCreated, nil)
}

func (b *NeutronBackend) VerifyNetwork(network core.Network) error {
	client, err := b.clientForTenant(network.TenantID)
	if err != nil {
		return err
	}
	provider, err := b.findNetwork(client, network.ID)
	if err != nil {
		return err
	}
	if provider.ID == "" {
		return errors.New("Neutron network was not found after creation")
	}
	return nil
}

func (b *NeutronBackend) InspectNetwork(network core.Network) (core.Network, error) {
	client, err := b.clientForTenant(network.TenantID)
	if err != nil {
		return core.Network{}, err
	}
	provider, err := b.findNetwork(client, network.ID)
	if err != nil {
		return core.Network{}, err
	}
	if provider.ID == "" {
		return core.Network{}, errors.New("Neutron network was not found")
	}
	if provider.ProjectID != "" && provider.ProjectID != client.config.ProjectID {
		return core.Network{}, errors.New("Neutron returned a network from an unexpected project")
	}
	network.ProviderID = provider.ID
	network.Status = provider.Status
	subnet, err := b.findNetworkSubnet(client, provider.ID, network.CIDR)
	if err != nil {
		return core.Network{}, err
	}
	network.ProviderSubnetID = subnet.ID
	network.ProviderSubnetStatus = subnet.Status
	network.SubnetStatus = subnet.Status
	return network, nil
}

func (b *NeutronBackend) ApplySubnet(subnet core.Subnet) error {
	client, err := b.clientForTenant(subnet.TenantID)
	if err != nil {
		return err
	}
	networkID, err := b.findNetwork(client, subnet.NetworkID)
	if err != nil {
		return err
	}
	if networkID.ID == "" {
		return errors.New("parent Neutron network is not provisioned")
	}
	existing, err := b.findSubnet(client, networkID.ID, subnet)
	if err != nil {
		return err
	}
	if existing.ID != "" {
		return nil
	}
	_, network, err := net.ParseCIDR(subnet.CIDR)
	if err != nil {
		return fmt.Errorf("parse subnet CIDR: %w", err)
	}
	providerSubnet := map[string]any{
		"network_id": networkID.ID,
		"cidr":       network.String(),
		"ip_version": 4,
		"description": resourceMarker(
			"subnet", subnet.ID, "", "",
		),
	}
	if network.IP.To4() == nil {
		providerSubnet["ip_version"] = 6
	}
	if subnet.GatewayIP != "" {
		providerSubnet["gateway_ip"] = subnet.GatewayIP
	}
	return client.request(http.MethodPost, "/subnets", map[string]any{
		"subnet": providerSubnet,
	}, http.StatusCreated, nil)
}

func (b *NeutronBackend) VerifySubnet(subnet core.Subnet) error {
	client, err := b.clientForTenant(subnet.TenantID)
	if err != nil {
		return err
	}
	networkID, err := b.findNetwork(client, subnet.NetworkID)
	if err != nil {
		return err
	}
	if networkID.ID == "" {
		return errors.New("parent Neutron network is not provisioned")
	}
	provider, err := b.findSubnet(client, networkID.ID, subnet)
	if err != nil {
		return err
	}
	if provider.ID == "" {
		return errors.New("Neutron subnet was not found after creation")
	}
	return nil
}

func (b *NeutronBackend) findNetwork(client *NeutronClient, localID string) (providerNetwork, error) {
	query := url.Values{"project_id": []string{client.config.ProjectID}, "limit": []string{"1000"}}
	var response struct {
		Networks []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			ProjectID   string `json:"project_id"`
			Status      string `json:"status"`
		} `json:"networks"`
	}
	if err := client.request(
		http.MethodGet, "/networks?"+query.Encode(), nil, http.StatusOK, &response,
	); err != nil {
		return providerNetwork{}, err
	}
	marker := resourceMarkerPrefix("network", localID)
	var found providerNetwork
	for _, network := range response.Networks {
		if network.Description != marker && !strings.HasPrefix(network.Description, marker+":") {
			continue
		}
		if found.ID != "" {
			return providerNetwork{}, errors.New("multiple Neutron networks match the AtonixCorp resource marker")
		}
		found = providerNetwork{
			ID: network.ID, ProjectID: network.ProjectID, Status: network.Status,
		}
	}
	return found, nil
}

func (b *NeutronBackend) findSubnet(client *NeutronClient, networkID string, subnet core.Subnet) (providerSubnet, error) {
	query := url.Values{
		"network_id": []string{networkID},
		"cidr":       []string{subnet.CIDR},
		"limit":      []string{"1000"},
	}
	var response struct {
		Subnets []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"subnets"`
	}
	if err := client.request(
		http.MethodGet, "/subnets?"+query.Encode(), nil, http.StatusOK, &response,
	); err != nil {
		return providerSubnet{}, err
	}
	marker := resourceMarkerPrefix("subnet", "")
	var found providerSubnet
	for _, item := range response.Subnets {
		if !strings.HasPrefix(item.Description, marker) {
			continue
		}
		if found.ID != "" {
			return providerSubnet{}, errors.New("multiple Neutron subnets match the AtonixCorp resource marker")
		}
		found = providerSubnet{ID: item.ID, Status: item.Status}
	}
	return found, nil
}

func (b *NeutronBackend) findNetworkSubnet(
	client *NeutronClient,
	networkID string,
	cidr string,
) (providerSubnet, error) {
	query := url.Values{
		"network_id": []string{networkID},
		"cidr":       []string{cidr},
		"limit":      []string{"1000"},
	}
	var response struct {
		Subnets []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"subnets"`
	}
	if err := client.request(
		http.MethodGet, "/subnets?"+query.Encode(), nil, http.StatusOK, &response,
	); err != nil {
		return providerSubnet{}, err
	}
	var found providerSubnet
	for _, item := range response.Subnets {
		if !strings.HasPrefix(item.Description, "atonixcorp:subnet:") {
			continue
		}
		if found.ID != "" {
			return providerSubnet{}, errors.New("multiple AtonixCorp subnets match the network CIDR")
		}
		found = providerSubnet{ID: item.ID, Status: item.Status}
	}
	return found, nil
}

func resourceMarkerPrefix(kind, id string) string {
	return "atonixcorp:" + kind + ":" + id
}

func resourceMarker(kind, id, tenantID, projectID string) string {
	marker := resourceMarkerPrefix(kind, id)
	if tenantID != "" && projectID != "" {
		marker += ":tenant:" + tenantID + ":project:" + projectID
	}
	return marker
}

func (c *NeutronClient) request(
	method, path string, payload any, expectedStatus int, result any,
) error {
	if err := c.ensureAuthenticated(); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil && payload != nil {
		return err
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, endpoint := c.currentSession()
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(body)
		}
		request, err := http.NewRequestWithContext(
			context.Background(), method, endpoint+path, reader,
		)
		if err != nil {
			return err
		}
		request.Header.Set("X-Auth-Token", token)
		if payload != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := c.httpClient.Do(request)
		if err != nil {
			return fmt.Errorf("Neutron request failed: %w", err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		closeErr := response.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read Neutron response: %w", readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close Neutron response: %w", closeErr)
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 {
			c.invalidateToken(token)
			if err := c.ensureAuthenticated(); err != nil {
				return err
			}
			continue
		}
		if response.StatusCode != expectedStatus {
			return fmt.Errorf(
				"Neutron %s %s returned unexpected HTTP status %d",
				method, request.URL.Path, response.StatusCode,
			)
		}
		if result != nil && len(responseBody) > 0 {
			if err := json.Unmarshal(responseBody, result); err != nil {
				return fmt.Errorf("decode Neutron response: %w", err)
			}
		}
		return nil
	}
	return errors.New("Neutron authentication retry was exhausted")
}

func (c *NeutronClient) ensureAuthenticated() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.expiresAt) > time.Minute {
		return nil
	}
	payload := map[string]any{"auth": map[string]any{
		"identity": map[string]any{
			"methods": []string{"application_credential"},
			"application_credential": map[string]string{
				"id":     c.config.ApplicationCredentialID,
				"secret": c.config.ApplicationCredentialSecret,
			},
		},
	}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(
		http.MethodPost, c.config.AuthURL+"/auth/tokens", bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("Keystone authentication failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("Keystone authentication returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Token struct {
			ExpiresAt string `json:"expires_at"`
			Project   struct {
				ID string `json:"id"`
			} `json:"project"`
			Catalog []struct {
				Type      string `json:"type"`
				Endpoints []struct {
					Region    string `json:"region"`
					RegionID  string `json:"region_id"`
					Interface string `json:"interface"`
					URL       string `json:"url"`
				} `json:"endpoints"`
			} `json:"catalog"`
		} `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return fmt.Errorf("decode Keystone response: %w", err)
	}
	token := response.Header.Get("X-Subject-Token")
	expiresAt, err := time.Parse(time.RFC3339, result.Token.ExpiresAt)
	if token == "" || err != nil || !expiresAt.After(time.Now()) {
		return errors.New("Keystone returned an invalid token or expiry")
	}
	if result.Token.Project.ID != c.config.ProjectID {
		return errors.New("OpenStack application credential is not scoped to the configured project")
	}
	var endpoint string
	for _, service := range result.Token.Catalog {
		if service.Type != "network" {
			continue
		}
		for _, candidate := range service.Endpoints {
			region := candidate.RegionID
			if region == "" {
				region = candidate.Region
			}
			if region != c.config.Region || candidate.Interface != c.config.Interface {
				continue
			}
			endpoint = strings.NewReplacer(
				"%(project_id)s", c.config.ProjectID,
				"%(tenant_id)s", c.config.ProjectID,
				"{project_id}", c.config.ProjectID,
				"{tenant_id}", c.config.ProjectID,
			).Replace(candidate.URL)
			break
		}
	}
	if endpoint == "" {
		return errors.New("Keystone catalog has no matching Neutron endpoint")
	}
	if err := validateEndpoint(endpoint, c.config.AllowInsecureHTTP); err != nil {
		return fmt.Errorf("Keystone catalog Neutron endpoint: %w", err)
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(endpoint, "/v2.0") {
		endpoint += "/v2.0"
	}
	c.token, c.expiresAt, c.endpoint = token, expiresAt, endpoint
	return nil
}

func (c *NeutronClient) currentSession() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token, c.endpoint
}

func (c *NeutronClient) invalidateToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == token {
		c.token = ""
	}
}

func validateEndpoint(value string, allowHTTP bool) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("valid OpenStack endpoint URL is required")
	}
	if parsed.Scheme == "https" || parsed.Scheme == "http" && allowHTTP {
		return nil
	}
	return errors.New("OpenStack endpoints must use HTTPS unless insecure HTTP is explicitly enabled")
}

func (b *NeutronBackend) ApplyRouter(router core.Router) error {
	return errors.New("Neutron router operations are not enabled by this backend")
}

func (b *NeutronBackend) AttachInterface(core.Router, core.Subnet, core.RouterInterface) error {
	return errors.New("Neutron router-interface operations are not enabled by this backend")
}

func (b *NeutronBackend) ApplyPort(core.Port) error {
	return errors.New("Neutron port operations are not enabled by this backend")
}

func (b *NeutronBackend) ApplySecurityGroup(core.SecurityGroup) error {
	return errors.New("Neutron security-group operations are not enabled by this backend")
}

func (b *NeutronBackend) ListNetworks() ([]core.Network, error) {
	return nil, errors.New("Neutron network import is not enabled")
}

func (b *NeutronBackend) ListSubnets() ([]core.Subnet, error) {
	return nil, errors.New("Neutron subnet import is not enabled")
}

func (b *NeutronBackend) ListRouters() ([]core.Router, error) {
	return nil, errors.New("Neutron router import is not enabled")
}

func (b *NeutronBackend) ListPorts() ([]core.Port, error) {
	return nil, errors.New("Neutron port import is not enabled")
}

func (b *NeutronBackend) ListSecurityGroups() ([]core.SecurityGroup, error) {
	return nil, errors.New("Neutron security-group import is not enabled")
}
