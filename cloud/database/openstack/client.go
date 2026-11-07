package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type AuthConfig struct {
	AuthURL                     string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	Username                    string
	Password                    string
	Domain                      string
	ProjectID                   string
	Region                      string
	Interface                   string
	AllowInsecureHTTP           bool
	ComputeEndpoint             string
	VolumeEndpoint              string
	NetworkEndpoint             string
	DefaultImageID              string
	DatabaseImages              map[string]string
	DatabaseFlavors             map[string]string
	DatabaseNetwork             string
	SecurityGroupID             string
}

type Client struct {
	ProjectID       string
	ComputeEndpoint string
	VolumeEndpoint  string
	NetworkEndpoint string
	DefaultImageID  string
	DatabaseImages  map[string]string
	DatabaseFlavors map[string]string
	DatabaseNetwork string
	SecurityGroupID string

	authURL                     string
	username                    string
	password                    string
	domain                      string
	applicationCredentialID     string
	applicationCredentialSecret string
	region                      string
	interfaceName               string
	allowInsecureHTTP           bool
	http                        *http.Client
	mu                          sync.Mutex
	token                       string
	expires                     time.Time
}

type ResponseError struct {
	Code int
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("OpenStack returned HTTP %d", e.Code)
}

func (e *ResponseError) HTTPStatusCode() int {
	return e.Code
}

func NewClient(cfg AuthConfig) (*Client, error) {
	cfg.AuthURL = strings.TrimRight(strings.TrimSpace(cfg.AuthURL), "/")
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.ProjectID = strings.TrimSpace(cfg.ProjectID)
	cfg.Domain = strings.TrimSpace(cfg.Domain)
	cfg.ApplicationCredentialID = strings.TrimSpace(cfg.ApplicationCredentialID)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Interface = strings.TrimSpace(cfg.Interface)
	if cfg.Domain == "" {
		cfg.Domain = "default"
	}
	if cfg.Region == "" {
		cfg.Region = "RegionOne"
	}
	if cfg.Interface == "" {
		cfg.Interface = "public"
	}
	if cfg.AuthURL == "" {
		return nil, errors.New("Keystone URL is required")
	}
	if err := validateServiceURL(cfg.AuthURL, cfg.AllowInsecureHTTP); err != nil {
		return nil, fmt.Errorf("Keystone endpoint: %w", err)
	}
	for _, endpoint := range []struct {
		name string
		url  string
	}{
		{"Nova", cfg.ComputeEndpoint},
		{"Cinder", cfg.VolumeEndpoint},
		{"Neutron", cfg.NetworkEndpoint},
	} {
		if endpoint.url == "" {
			continue
		}
		if err := validateServiceURL(endpoint.url, cfg.AllowInsecureHTTP); err != nil {
			return nil, fmt.Errorf("%s endpoint: %w", endpoint.name, err)
		}
	}
	hasApplicationCredential := cfg.ApplicationCredentialID != "" ||
		cfg.ApplicationCredentialSecret != ""
	hasPasswordCredential := cfg.Username != "" || cfg.Password != ""
	if cfg.ProjectID == "" {
		return nil, errors.New("OpenStack project ID is required")
	}
	if hasApplicationCredential &&
		(cfg.ApplicationCredentialID == "" || cfg.ApplicationCredentialSecret == "") {
		return nil, errors.New("OpenStack application credential ID and secret must both be provided")
	}
	if !hasApplicationCredential && (!hasPasswordCredential ||
		cfg.Username == "" || cfg.Password == "") {
		return nil, errors.New("OpenStack application credentials or username and password are required")
	}
	client := &Client{
		ProjectID: cfg.ProjectID, ComputeEndpoint: strings.TrimRight(cfg.ComputeEndpoint, "/"),
		VolumeEndpoint:  strings.TrimRight(cfg.VolumeEndpoint, "/"),
		NetworkEndpoint: strings.TrimRight(cfg.NetworkEndpoint, "/"),
		DefaultImageID:  cfg.DefaultImageID, DatabaseImages: cfg.DatabaseImages,
		DatabaseFlavors: cfg.DatabaseFlavors, DatabaseNetwork: cfg.DatabaseNetwork,
		SecurityGroupID: cfg.SecurityGroupID, authURL: cfg.AuthURL,
		username: cfg.Username, password: cfg.Password, domain: cfg.Domain,
		applicationCredentialID:     cfg.ApplicationCredentialID,
		applicationCredentialSecret: cfg.ApplicationCredentialSecret,
		region:                      cfg.Region, interfaceName: cfg.Interface,
		allowInsecureHTTP: cfg.AllowInsecureHTTP,
		http: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	return client, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	identity := make(map[string]any)
	auth := map[string]any{"identity": identity}
	payload := map[string]any{"auth": auth}
	if c.applicationCredentialID != "" {
		identity["methods"] = []string{"application_credential"}
		identity["application_credential"] = map[string]string{
			"id": c.applicationCredentialID, "secret": c.applicationCredentialSecret,
		}
	} else {
		identity["methods"] = []string{"password"}
		identity["password"] = map[string]any{"user": map[string]any{
			"name": c.username, "password": c.password,
			"domain": map[string]string{"name": c.domain},
		}}
		auth["scope"] = map[string]any{
			"project": map[string]string{"id": c.ProjectID},
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.authURL+"/auth/tokens", bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
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
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("decode Keystone response: %w", err)
	}
	token := response.Header.Get("X-Subject-Token")
	if token == "" {
		return errors.New("Keystone response did not contain X-Subject-Token")
	}
	expires, err := time.Parse(time.RFC3339, result.Token.ExpiresAt)
	if err != nil || !expires.After(time.Now()) {
		return errors.New("Keystone returned an invalid token expiry")
	}
	if result.Token.Project.ID != c.ProjectID {
		return errors.New("OpenStack application credential is not scoped to the configured project")
	}
	computeURL, volumeURL, networkURL, err := c.catalogEndpoints(result.Token.Catalog)
	if err != nil {
		return err
	}
	if c.ComputeEndpoint == "" {
		c.ComputeEndpoint = computeURL
	}
	if c.VolumeEndpoint == "" {
		c.VolumeEndpoint = volumeURL
	}
	if c.NetworkEndpoint == "" {
		c.NetworkEndpoint = networkURL
	}
	if c.ComputeEndpoint == "" || c.VolumeEndpoint == "" {
		return errors.New("Keystone catalog is missing the configured Nova or Cinder endpoint")
	}
	c.token, c.expires = token, expires
	return nil
}

func (c *Client) do(method, endpoint string, payload any) ([]byte, error) {
	return c.doContext(context.Background(), method, endpoint, payload)
}

func (c *Client) doContext(
	ctx context.Context, method, endpoint string, payload any,
) ([]byte, error) {
	if err := validateServiceURL(endpoint, c.allowInsecureHTTP); err != nil {
		return nil, err
	}
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.ensureToken(ctx)
		if err != nil {
			return nil, err
		}
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Auth-Token", token)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		response, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("OpenStack request failed: %w", err)
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read OpenStack response: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close OpenStack response: %w", closeErr)
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 {
			c.invalidateToken(token)
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, &ResponseError{Code: response.StatusCode}
		}
		return responseBody, nil
	}
	return nil, errors.New("OpenStack authentication retry was exhausted")
}

func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.expires) > time.Minute {
		return c.token, nil
	}
	if err := c.authenticate(ctx); err != nil {
		return "", err
	}
	return c.token, nil
}

func (c *Client) invalidateToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == token {
		c.token = ""
	}
}

func validateServiceURL(value string, allowHTTP bool) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("valid service URL is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("valid service URL is required")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && allowHTTP {
		return nil
	}
	return errors.New("OpenStack service URLs must use HTTPS")
}

func (c *Client) catalogEndpoints(catalog []struct {
	Type      string `json:"type"`
	Endpoints []struct {
		Region    string `json:"region"`
		RegionID  string `json:"region_id"`
		Interface string `json:"interface"`
		URL       string `json:"url"`
	} `json:"endpoints"`
}) (string, string, string, error) {
	endpoints := map[string]string{}
	serviceTypes := map[string][]string{
		"compute": {"compute"},
		"volume":  {"volumev3", "volumev2", "volume"},
		"network": {"network"},
	}
	for service, types := range serviceTypes {
		for _, candidateType := range types {
			for _, entry := range catalog {
				if entry.Type != candidateType {
					continue
				}
				for _, endpoint := range entry.Endpoints {
					region := endpoint.RegionID
					if region == "" {
						region = endpoint.Region
					}
					if region == c.region && endpoint.Interface == c.interfaceName {
						resolved := strings.NewReplacer(
							"%(project_id)s", c.ProjectID,
							"%(tenant_id)s", c.ProjectID,
							"{project_id}", c.ProjectID,
							"{tenant_id}", c.ProjectID,
						).Replace(endpoint.URL)
						if err := validateServiceURL(resolved, c.allowInsecureHTTP); err != nil {
							return "", "", "", fmt.Errorf("Keystone catalog %s endpoint: %w", service, err)
						}
						endpoints[service] = strings.TrimRight(resolved, "/")
						break
					}
				}
			}
			if endpoints[service] != "" {
				break
			}
		}
	}
	return endpoints["compute"], endpoints["volume"], endpoints["network"], nil
}
