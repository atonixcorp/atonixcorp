package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/models"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

type OpenStackConfig struct {
	AuthURL                     string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	ProjectID                   string
	AllowProjectProvisioning    bool
	Region                      string
	Interface                   string
	AllowInsecureHTTP           bool
}

type OpenStackCapabilities = ComputeCapabilities

type OpenStackProvider struct {
	Provider                 *gophercloud.ProviderClient
	Compute                  *gophercloud.ServiceClient
	Network                  *gophercloud.ServiceClient
	config                   OpenStackConfig
	region                   string
	projectID                string
	allowProjectProvisioning bool
	capabilities             ComputeCapabilities
}

type openStackImage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	MinDisk   int    `json:"min_disk"`
	MinMemory int    `json:"min_ram"`
}

type openStackFlavor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	VCPUs     int    `json:"vcpus"`
	RAM       int    `json:"ram"`
	Disk      int    `json:"disk"`
	Ephemeral int    `json:"OS-FLV-EXT-DATA:ephemeral"`
	Swap      string `json:"swap"`
}

type openStackNetwork struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	AdminStateUp bool   `json:"admin_state_up"`
	Shared       bool   `json:"shared"`
	External     bool   `json:"router:external"`
}

func catalogPageURL(endpoint, marker string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsed.Query()
	query.Set("limit", "500")
	if marker != "" {
		query.Set("marker", marker)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func providerRequestError(operation string, err error) error {
	for _, status := range []int{
		http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusConflict, http.StatusRequestEntityTooLarge, http.StatusTooManyRequests,
		http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusInternalServerError,
	} {
		if gophercloud.ResponseCodeIs(err, status) {
			return fmt.Errorf("%w: %s returned HTTP %d", ErrProviderRequest, operation, status)
		}
	}
	return fmt.Errorf("%w: %s provider request failed", ErrProviderRequest, operation)
}

func catalogRequestError(service string, err error) error {
	return providerRequestError(service+" catalog request", err)
}

func OpenStackConfigFromEnv() (OpenStackConfig, bool, error) {
	cfg := OpenStackConfig{
		AuthURL:                     strings.TrimSpace(os.Getenv("OPENSTACK_AUTH_URL")),
		ApplicationCredentialID:     strings.TrimSpace(os.Getenv("OPENSTACK_APPLICATION_CREDENTIAL_ID")),
		ApplicationCredentialSecret: os.Getenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET"),
		Region:                      strings.TrimSpace(os.Getenv("OPENSTACK_REGION")),
		Interface:                   strings.TrimSpace(os.Getenv("OPENSTACK_INTERFACE")),
		AllowInsecureHTTP:           strings.EqualFold(strings.TrimSpace(os.Getenv("OPENSTACK_ALLOW_INSECURE_HTTP")), "true"),
	}
	if cfg.AuthURL == "" && cfg.ApplicationCredentialID == "" &&
		cfg.ApplicationCredentialSecret == "" {
		return OpenStackConfig{}, false, nil
	}
	if cfg.AuthURL == "" {
		return OpenStackConfig{}, false, errors.New("OPENSTACK_AUTH_URL is required")
	}
	if (cfg.ApplicationCredentialID == "") != (strings.TrimSpace(cfg.ApplicationCredentialSecret) == "") {
		return OpenStackConfig{}, false, errors.New(
			"OPENSTACK_APPLICATION_CREDENTIAL_ID and OPENSTACK_APPLICATION_CREDENTIAL_SECRET must be set together",
		)
	}
	if err := validateAuthURL(cfg.AuthURL, cfg.AllowInsecureHTTP); err != nil {
		return OpenStackConfig{}, false, err
	}
	if cfg.Region == "" {
		cfg.Region = "RegionOne"
	}
	if cfg.Interface == "" {
		cfg.Interface = "public"
	}
	if !validOpenStackInterface(cfg.Interface) {
		return OpenStackConfig{}, false, errors.New(
			"OPENSTACK_INTERFACE must be public, internal, or admin",
		)
	}
	return cfg, true, nil
}

func NewOpenStackProvider(ctx context.Context, cfg OpenStackConfig) (*OpenStackProvider, error) {
	if cfg.AuthURL == "" || cfg.Region == "" {
		return nil, errors.New("Keystone URL and region are required")
	}
	if (cfg.ApplicationCredentialID == "") != (strings.TrimSpace(cfg.ApplicationCredentialSecret) == "") {
		return nil, errors.New("both OpenStack application credential values are required")
	}
	if cfg.AllowProjectProvisioning && cfg.ProjectID == "" {
		return nil, errors.New("organization-scoped provisioning requires a Keystone project ID")
	}
	if cfg.Interface == "" {
		cfg.Interface = "public"
	}
	if !validOpenStackInterface(cfg.Interface) {
		return nil, errors.New("OPENSTACK_INTERFACE must be public, internal, or admin")
	}
	if err := validateAuthURL(cfg.AuthURL, cfg.AllowInsecureHTTP); err != nil {
		return nil, err
	}
	if cfg.ApplicationCredentialID == "" {
		return &OpenStackProvider{config: cfg, region: cfg.Region}, nil
	}
	provider, err := openstack.AuthenticatedClient(ctx, gophercloud.AuthOptions{
		IdentityEndpoint:            cfg.AuthURL,
		ApplicationCredentialID:     cfg.ApplicationCredentialID,
		ApplicationCredentialSecret: cfg.ApplicationCredentialSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("authenticate with Keystone: %w", err)
	}
	tokenInfo, err := readOpenStackToken(ctx, cfg, provider.TokenID)
	if err != nil {
		return nil, err
	}
	if cfg.ProjectID != "" {
		if tokenInfo.ProjectID == "" || tokenInfo.ProjectID != cfg.ProjectID {
			return nil, errors.New("OpenStack application credential is not scoped to the mapped Keystone project")
		}
	}
	capabilities := openStackCapabilities(tokenInfo.Catalog, cfg.Region, cfg.Interface)
	var compute *gophercloud.ServiceClient
	if capabilities.Compute {
		compute, err = openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region: cfg.Region, Availability: openStackAvailability(cfg.Interface),
		})
		if err != nil {
			return nil, fmt.Errorf("discover Nova endpoint: %w", err)
		}
	}
	var network *gophercloud.ServiceClient
	if capabilities.Network {
		network, err = openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
			Region: cfg.Region, Availability: openStackAvailability(cfg.Interface),
		})
		if err != nil {
			return nil, fmt.Errorf("discover Neutron endpoint: %w", err)
		}
	}
	return &OpenStackProvider{
		Provider: provider, Compute: compute, Network: network, config: cfg, region: cfg.Region,
		projectID:                cfg.ProjectID,
		allowProjectProvisioning: cfg.AllowProjectProvisioning,
		capabilities:             capabilities,
	}, nil
}

type OpenStackAccountCredentials struct {
	ProjectID        string `json:"project_id"`
	CredentialID     string `json:"credential_id"`
	CredentialSecret string `json:"credential_secret"`
	Region           string `json:"region"`
}

func (account OpenStackAccountCredentials) ProviderName() string { return "openstack" }
func (account OpenStackAccountCredentials) ScopeID() string {
	return strings.TrimSpace(account.ProjectID)
}
func (account OpenStackAccountCredentials) RegionName() string {
	return strings.TrimSpace(account.Region)
}
func (account OpenStackAccountCredentials) Validate() error {
	if account.ScopeID() == "" || strings.TrimSpace(account.CredentialID) == "" || strings.TrimSpace(account.CredentialSecret) == "" ||
		len(account.ScopeID()) > 255 || len(account.CredentialID) > 255 || len(account.RegionName()) > 64 {
		return ErrInvalid
	}
	return nil
}
func (account OpenStackAccountCredentials) Encode() ([]byte, error) {
	if err := account.Validate(); err != nil {
		return nil, err
	}
	account.ProjectID = account.ScopeID()
	account.Region = account.RegionName()
	account.CredentialID = strings.TrimSpace(account.CredentialID)
	return json.Marshal(account)
}

func (p *OpenStackProvider) DecodeStoredAccount(record models.CloudAccount, plaintext []byte) (CloudAccountCredentials, error) {
	if record.Provider != "" && record.Provider != p.Name() {
		return nil, ErrInvalid
	}
	var account OpenStackAccountCredentials
	switch record.CredentialVersion {
	case 0:
		account = OpenStackAccountCredentials{ProjectID: record.ScopeID, CredentialID: record.CredentialID, CredentialSecret: string(plaintext), Region: record.Region}
	case 1:
		if err := json.Unmarshal(plaintext, &account); err != nil {
			return nil, ErrInvalid
		}
	default:
		return nil, ErrInvalid
	}
	if err := account.Validate(); err != nil {
		return nil, err
	}
	if account.ScopeID() != record.ScopeID || account.RegionName() != record.Region {
		return nil, ErrInvalid
	}
	return account, nil
}

func (p *OpenStackProvider) ForAccount(
	ctx context.Context, account CloudAccountCredentials,
) (Provider, error) {
	credential, ok := account.(OpenStackAccountCredentials)
	if !ok {
		return nil, ErrInvalid
	}
	if err := credential.Validate(); err != nil {
		return nil, err
	}
	config := p.config
	config.ApplicationCredentialID = strings.TrimSpace(credential.CredentialID)
	config.ApplicationCredentialSecret = credential.CredentialSecret
	config.ProjectID = credential.ScopeID()
	config.AllowProjectProvisioning = true
	if credential.RegionName() != "" {
		config.Region = credential.RegionName()
	}
	return NewOpenStackProvider(ctx, config)
}

type openStackTokenInfo struct {
	ProjectID string
	Catalog   []openStackCatalogService
}

type openStackCatalogService struct {
	Type      string                     `json:"type"`
	Endpoints []openStackCatalogEndpoint `json:"endpoints"`
}

type openStackCatalogEndpoint struct {
	Region    string `json:"region"`
	RegionID  string `json:"region_id"`
	Interface string `json:"interface"`
	URL       string `json:"url"`
}

func readOpenStackToken(
	ctx context.Context, cfg OpenStackConfig, token string,
) (openStackTokenInfo, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, strings.TrimRight(cfg.AuthURL, "/")+"/auth/tokens", nil,
	)
	if err != nil {
		return openStackTokenInfo{}, fmt.Errorf("inspect OpenStack service catalog: %w", err)
	}
	request.Header.Set("X-Auth-Token", token)
	request.Header.Set("X-Subject-Token", token)
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return openStackTokenInfo{}, fmt.Errorf("inspect OpenStack service catalog: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return openStackTokenInfo{}, fmt.Errorf(
			"inspect OpenStack service catalog: Keystone returned HTTP %d", response.StatusCode,
		)
	}
	var result struct {
		Token struct {
			Project struct {
				ID string `json:"id"`
			} `json:"project"`
			Catalog []openStackCatalogService `json:"catalog"`
		} `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return openStackTokenInfo{}, fmt.Errorf("decode OpenStack service catalog: %w", err)
	}
	return openStackTokenInfo{ProjectID: result.Token.Project.ID, Catalog: result.Token.Catalog}, nil
}

func verifyCredentialProject(ctx context.Context, cfg OpenStackConfig, token string) error {
	tokenInfo, err := readOpenStackToken(ctx, cfg, token)
	if err != nil {
		return err
	}
	if tokenInfo.ProjectID == "" || tokenInfo.ProjectID != cfg.ProjectID {
		return errors.New("OpenStack application credential is not scoped to the mapped Keystone project")
	}
	return nil
}

func openStackCapabilities(
	catalog []openStackCatalogService, region, interfaceName string,
) ComputeCapabilities {
	var capabilities ComputeCapabilities
	for _, service := range catalog {
		available := false
		for _, endpoint := range service.Endpoints {
			endpointRegion := endpoint.Region
			if endpointRegion == "" {
				endpointRegion = endpoint.RegionID
			}
			if strings.TrimSpace(endpoint.URL) != "" &&
				(endpointRegion == "" || endpointRegion == region) &&
				(endpoint.Interface == "" || endpoint.Interface == interfaceName) {
				available = true
				break
			}
		}
		if !available {
			continue
		}
		switch service.Type {
		case "compute":
			capabilities.Compute = true
		case "network", "networking":
			capabilities.Network = true
		case "image":
			capabilities.Image = true
		case "block-storage", "block-store", "volume", "volumev2", "volumev3":
			capabilities.BlockStorage = true
		case "identity":
			capabilities.Identity = true
		case "object-store":
			capabilities.ObjectStorage = true
		case "load-balancer":
			capabilities.LoadBalancer = true
		case "dns":
			capabilities.DNS = true
		case "key-manager":
			capabilities.KeyManagement = true
		case "orchestration":
			capabilities.Orchestration = true
		}
	}
	return capabilities
}

func validOpenStackInterface(interfaceName string) bool {
	return interfaceName == "public" || interfaceName == "internal" || interfaceName == "admin"
}

func openStackAvailability(interfaceName string) gophercloud.Availability {
	switch interfaceName {
	case "internal":
		return gophercloud.AvailabilityInternal
	case "admin":
		return gophercloud.AvailabilityAdmin
	default:
		return gophercloud.AvailabilityPublic
	}
}

func validateAuthURL(authURL string, allowHTTP bool) error {
	parsed, err := url.Parse(authURL)
	if err != nil || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") {
		return errors.New("OpenStack Keystone URL must be a valid HTTP or HTTPS URL without user info, query, or fragment")
	}
	if parsed.Scheme == "http" && !allowHTTP {
		return errors.New("HTTP Keystone requires OPENSTACK_ALLOW_INSECURE_HTTP=true for isolated development environments")
	}
	return nil
}

func (p *OpenStackProvider) Region() string {
	return p.region
}

func (p *OpenStackProvider) Name() string {
	return "openstack"
}

func (p *OpenStackProvider) ProjectProvisioningEnabled() bool {
	return p.allowProjectProvisioning && p.projectID != "" && p.capabilities.Compute
}

func (p *OpenStackProvider) Capabilities() ComputeCapabilities {
	return p.capabilities
}

func (p *OpenStackProvider) ComputeAvailable() bool {
	return p.Compute != nil
}

func (p *OpenStackProvider) ComputeCatalog(ctx context.Context) (ComputeCatalog, error) {
	catalog := ComputeCatalog{
		Region: p.region, Capabilities: p.capabilities,
		Images: []CatalogImage{}, Sizes: []ComputeSize{}, Networks: []CatalogNetwork{},
	}
	if p.Provider == nil || p.Compute == nil {
		return catalog, ErrProviderUnavailable
	}
	if p.capabilities.Image {
		client, err := openstack.NewImageV2(p.Provider, gophercloud.EndpointOpts{
			Region: p.region, Availability: openStackAvailability(p.config.Interface),
		})
		if err != nil {
			return catalog, fmt.Errorf("%w: discover Glance catalog endpoint", ErrProviderRequest)
		}
		endpoint := client.ServiceURL("images")
		marker := ""
		for {
			var result struct {
				Images []openStackImage `json:"images"`
			}
			if _, err := client.Get(ctx, catalogPageURL(endpoint, marker), &result, nil); err != nil {
				return catalog, catalogRequestError("Glance image", err)
			}
			for _, image := range result.Images {
				if image.ID == "" {
					continue
				}
				catalog.Images = append(catalog.Images, CatalogImage{
					ID: image.ID, Name: image.Name, Status: strings.ToUpper(image.Status),
					MinDiskGB: image.MinDisk, MinMemoryMB: image.MinMemory,
				})
			}
			if len(result.Images) < 500 || len(result.Images) == 0 ||
				result.Images[len(result.Images)-1].ID == marker {
				break
			}
			marker = result.Images[len(result.Images)-1].ID
		}
	}
	flavorEndpoint := p.Compute.ServiceURL("flavors", "detail")
	flavorMarker := ""
	for {
		var flavorResult struct {
			Flavors []openStackFlavor `json:"flavors"`
		}
		if _, err := p.Compute.Get(
			ctx, catalogPageURL(flavorEndpoint, flavorMarker), &flavorResult, nil,
		); err != nil {
			return catalog, catalogRequestError("Nova flavor", err)
		}
		for _, flavor := range flavorResult.Flavors {
			if flavor.ID == "" {
				continue
			}
			swapMB, _ := strconv.Atoi(flavor.Swap)
			catalog.Sizes = append(catalog.Sizes, ComputeSize{
				ID: flavor.ID, Name: flavor.Name, VCPUs: flavor.VCPUs, MemoryMB: flavor.RAM,
				DiskGB: flavor.Disk, EphemeralGB: flavor.Ephemeral, SwapMB: swapMB,
			})
		}
		if len(flavorResult.Flavors) < 500 || len(flavorResult.Flavors) == 0 ||
			flavorResult.Flavors[len(flavorResult.Flavors)-1].ID == flavorMarker {
			break
		}
		flavorMarker = flavorResult.Flavors[len(flavorResult.Flavors)-1].ID
	}
	if p.capabilities.Network {
		if p.Network == nil {
			return catalog, ErrSecurityGroupUnavailable
		}
		endpoint := p.Network.ServiceURL("networks")
		marker := ""
		for {
			var result struct {
				Networks []openStackNetwork `json:"networks"`
			}
			if _, err := p.Network.Get(ctx, catalogPageURL(endpoint, marker), &result, nil); err != nil {
				return catalog, catalogRequestError("Neutron network", err)
			}
			for _, network := range result.Networks {
				if network.ID == "" {
					continue
				}
				catalog.Networks = append(catalog.Networks, CatalogNetwork{
					ID: network.ID, Name: network.Name, Status: strings.ToUpper(network.Status),
					AdminStateUp: network.AdminStateUp, Shared: network.Shared, External: network.External,
				})
			}
			if len(result.Networks) < 500 || len(result.Networks) == 0 ||
				result.Networks[len(result.Networks)-1].ID == marker {
				break
			}
			marker = result.Networks[len(result.Networks)-1].ID
		}
	}
	return catalog, nil
}

type openStackSecurityGroup struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	ProjectID   string                  `json:"project_id"`
	TenantID    string                  `json:"tenant_id"`
	Rules       []openStackSecurityRule `json:"security_group_rules"`
}

type openStackSecurityRule struct {
	ID             string `json:"id"`
	Direction      string `json:"direction"`
	Ethertype      string `json:"ethertype"`
	Protocol       string `json:"protocol"`
	PortRangeMin   *int   `json:"port_range_min"`
	PortRangeMax   *int   `json:"port_range_max"`
	RemoteIPPrefix string `json:"remote_ip_prefix"`
}

func (p *OpenStackProvider) ListSecurityGroups(ctx context.Context) ([]SecurityGroup, error) {
	if p.Network == nil || p.projectID == "" {
		return nil, ErrSecurityGroupUnavailable
	}
	endpoint := p.Network.ServiceURL("security-groups")
	query := url.Values{
		"project_id": []string{p.projectID},
		"limit":      []string{"500"},
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid Neutron endpoint", ErrProviderRequest)
	}
	parsed.RawQuery = query.Encode()
	marker := ""
	groups := make([]SecurityGroup, 0)
	for {
		var result struct {
			Groups []openStackSecurityGroup `json:"security_groups"`
		}
		if _, err := p.Network.Get(ctx, catalogPageURL(parsed.String(), marker), &result, nil); err != nil {
			return nil, providerRequestError("Neutron security group list", err)
		}
		for _, group := range result.Groups {
			projectID := group.ProjectID
			if projectID == "" {
				projectID = group.TenantID
			}
			if group.ID == "" || projectID != p.projectID {
				continue
			}
			groups = append(groups, SecurityGroup{
				ID: group.ID, Name: group.Name, Description: group.Description,
				ProjectID: projectID, Rules: convertSecurityRules(group.Rules),
			})
		}
		if len(result.Groups) < 500 || len(result.Groups) == 0 ||
			result.Groups[len(result.Groups)-1].ID == marker {
			break
		}
		marker = result.Groups[len(result.Groups)-1].ID
	}
	return groups, nil
}

func (p *OpenStackProvider) CreateSecurityGroup(
	ctx context.Context, name string,
) (SecurityGroup, error) {
	if p.Network == nil || p.projectID == "" {
		return SecurityGroup{}, ErrSecurityGroupUnavailable
	}
	description := "Managed by AtonixCorp; ingress TCP 443 limited to RFC 5737 test range."
	var result struct {
		Group openStackSecurityGroup `json:"security_group"`
	}
	if _, err := p.Network.Post(ctx, p.Network.ServiceURL("security-groups"), map[string]any{
		"security_group": map[string]any{
			"name": name, "description": description, "project_id": p.projectID,
		},
	}, &result, nil); err != nil {
		return SecurityGroup{}, providerRequestError("Neutron security group create", err)
	}
	group := result.Group
	groupProjectID := group.ProjectID
	if groupProjectID == "" {
		groupProjectID = group.TenantID
	}
	if group.ID == "" || groupProjectID != p.projectID {
		return SecurityGroup{}, fmt.Errorf("%w: Neutron returned an unscoped security group", ErrProviderRequest)
	}
	const safeTestCIDR = "192.0.2.0/24"
	port := 443
	var ruleResult struct {
		Rule openStackSecurityRule `json:"security_group_rule"`
	}
	_, ruleErr := p.Network.Post(ctx, p.Network.ServiceURL("security-group-rules"), map[string]any{
		"security_group_rule": map[string]any{
			"security_group_id": group.ID,
			"project_id":        p.projectID,
			"direction":         "ingress",
			"ethertype":         "IPv4",
			"protocol":          "tcp",
			"port_range_min":    port,
			"port_range_max":    port,
			"remote_ip_prefix":  safeTestCIDR,
		},
	}, &ruleResult, nil)
	if ruleErr != nil {
		_, cleanupErr := p.Network.Delete(ctx, p.Network.ServiceURL("security-groups", group.ID), nil)
		cause := providerRequestError("Neutron security group rule create", ruleErr)
		if cleanupErr != nil {
			cause = errors.Join(cause, providerRequestError("Neutron security group cleanup", cleanupErr))
		}
		return SecurityGroup{}, cause
	}
	group.Rules = append(group.Rules, ruleResult.Rule)
	return SecurityGroup{
		ID: group.ID, Name: group.Name, Description: group.Description,
		ProjectID: groupProjectID, Rules: convertSecurityRules(group.Rules),
	}, nil
}

func convertSecurityRules(rules []openStackSecurityRule) []SecurityGroupRule {
	result := make([]SecurityGroupRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, SecurityGroupRule{
			ID: rule.ID, Direction: rule.Direction, Ethertype: rule.Ethertype,
			Protocol: rule.Protocol, PortRangeMin: rule.PortRangeMin,
			PortRangeMax: rule.PortRangeMax, RemoteIPPrefix: rule.RemoteIPPrefix,
		})
	}
	return result
}

func (p *OpenStackProvider) CreateInstanceForProject(
	ctx context.Context,
	tenantID, projectID uint,
	request ProviderCreateRequest,
) (ProviderInstance, error) {
	if !p.ProjectProvisioningEnabled() {
		return ProviderInstance{}, ErrProjectProvisioningDisabled
	}
	if p.Compute == nil {
		return ProviderInstance{}, ErrProviderUnavailable
	}
	if tenantID == 0 || projectID == 0 {
		return ProviderInstance{}, ErrInvalid
	}
	opts := keypairs.CreateOptsExt{
		CreateOptsBuilder: servers.CreateOpts{
			Name:      request.Name,
			ImageRef:  request.ImageID,
			FlavorRef: request.SizeID,
			Networks:  []servers.Network{{UUID: request.NetworkID}},
			Metadata: map[string]string{
				"atonixcorp_instance_id":          request.ID,
				"atonixcorp_tenant_id":            strconv.FormatUint(uint64(tenantID), 10),
				"atonixcorp_project_id":           strconv.FormatUint(uint64(projectID), 10),
				"atonixcorp_openstack_project_id": p.projectID,
			},
		},
		KeyName: request.KeyName,
	}
	result, err := servers.Create(ctx, p.Compute, opts, nil).Extract()
	if err != nil {
		return ProviderInstance{}, providerRequestError("Nova create server", err)
	}
	return providerInstance(result), nil
}

func (p *OpenStackProvider) GetInstance(ctx context.Context, id string) (ProviderInstance, error) {
	if p.Compute == nil {
		return ProviderInstance{}, ErrProviderUnavailable
	}
	result, err := servers.Get(ctx, p.Compute, id).Extract()
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return ProviderInstance{}, ErrProviderNotFound
		}
		return ProviderInstance{}, providerRequestError("Nova get server", err)
	}
	return providerInstance(result), nil
}

func (p *OpenStackProvider) StartInstance(ctx context.Context, id string) error {
	if p.Compute == nil {
		return ErrProviderUnavailable
	}
	if err := servers.Start(ctx, p.Compute, id).ExtractErr(); err != nil {
		return providerRequestError("Nova start server", err)
	}
	return nil
}

func (p *OpenStackProvider) StopInstance(ctx context.Context, id string) error {
	if p.Compute == nil {
		return ErrProviderUnavailable
	}
	if err := servers.Stop(ctx, p.Compute, id).ExtractErr(); err != nil {
		return providerRequestError("Nova stop server", err)
	}
	return nil
}

func (p *OpenStackProvider) RebootInstance(ctx context.Context, id string) error {
	if p.Compute == nil {
		return ErrProviderUnavailable
	}
	if err := servers.Reboot(ctx, p.Compute, id, servers.RebootOpts{
		Type: servers.SoftReboot,
	}).ExtractErr(); err != nil {
		return providerRequestError("Nova reboot server", err)
	}
	return nil
}

func (p *OpenStackProvider) DeleteInstance(ctx context.Context, id string) error {
	if p.Compute == nil {
		return ErrProviderUnavailable
	}
	err := servers.Delete(ctx, p.Compute, id).ExtractErr()
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil
	}
	if err != nil {
		return providerRequestError("Nova delete server", err)
	}
	return nil
}

func providerInstance(server *servers.Server) ProviderInstance {
	instance := ProviderInstance{ID: server.ID, Status: server.Status}
	for _, rawAddresses := range server.Addresses {
		addresses, ok := rawAddresses.([]any)
		if !ok {
			continue
		}
		for _, rawAddress := range addresses {
			address, ok := rawAddress.(map[string]any)
			if !ok {
				continue
			}
			ip, _ := address["addr"].(string)
			ipType, _ := address["OS-EXT-IPS:type"].(string)
			if ipType == "floating" {
				if instance.PublicIP == "" {
					instance.PublicIP = ip
				}
			} else if instance.PrivateIP == "" {
				instance.PrivateIP = ip
			}
		}
	}
	return instance
}
