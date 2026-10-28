package openstack

import (
	"bytes"
	"context"
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
)

type AuthConfig struct {
	AuthURL         string
	Username        string
	Password        string
	Domain          string
	ProjectID       string
	ComputeEndpoint string
	VolumeEndpoint  string
	NetworkEndpoint string
	DefaultImageID  string
	DatabaseImages  map[string]string
	DatabaseFlavors map[string]string
	DatabaseNetwork string
	SecurityGroupID string
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

	authURL  string
	username string
	password string
	domain   string
	http     *http.Client
	mu       sync.Mutex
	token    string
	expires  time.Time
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
	if cfg.Domain == "" {
		cfg.Domain = "default"
	}
	for _, endpoint := range []struct {
		name string
		url  string
	}{
		{"Keystone", cfg.AuthURL},
		{"Nova", cfg.ComputeEndpoint},
		{"Cinder", cfg.VolumeEndpoint},
	} {
		if err := validateServiceURL(endpoint.url); err != nil {
			return nil, fmt.Errorf("%s endpoint: %w", endpoint.name, err)
		}
	}
	if cfg.Username == "" || cfg.Password == "" || cfg.ProjectID == "" {
		return nil, errors.New("OpenStack username, password, and project ID are required")
	}
	client := &Client{
		ProjectID: cfg.ProjectID, ComputeEndpoint: strings.TrimRight(cfg.ComputeEndpoint, "/"),
		VolumeEndpoint:  strings.TrimRight(cfg.VolumeEndpoint, "/"),
		NetworkEndpoint: strings.TrimRight(cfg.NetworkEndpoint, "/"),
		DefaultImageID:  cfg.DefaultImageID, DatabaseImages: cfg.DatabaseImages,
		DatabaseFlavors: cfg.DatabaseFlavors, DatabaseNetwork: cfg.DatabaseNetwork,
		SecurityGroupID: cfg.SecurityGroupID, authURL: cfg.AuthURL,
		username: cfg.Username, password: cfg.Password, domain: cfg.Domain,
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
	payload := map[string]any{
		"auth": map[string]any{
			"identity": map[string]any{
				"methods": []string{"password"},
				"password": map[string]any{"user": map[string]any{
					"name": c.username, "password": c.password,
					"domain": map[string]string{"name": c.domain},
				}},
			},
			"scope": map[string]any{"project": map[string]string{"id": c.ProjectID}},
		},
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
	c.token, c.expires = token, expires
	return nil
}

func (c *Client) do(method, endpoint string, payload any) ([]byte, error) {
	return c.doContext(context.Background(), method, endpoint, payload)
}

func (c *Client) doContext(
	ctx context.Context, method, endpoint string, payload any,
) ([]byte, error) {
	if err := validateServiceURL(endpoint); err != nil {
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

func validateServiceURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("valid service URL is required")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" {
		host, _, splitErr := net.SplitHostPort(parsed.Host)
		if splitErr != nil {
			host = parsed.Host
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
		if strings.EqualFold(host, "localhost") {
			return nil
		}
	}
	return errors.New("OpenStack service URLs must use HTTPS")
}
