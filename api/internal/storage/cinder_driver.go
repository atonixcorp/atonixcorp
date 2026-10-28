package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"atonixcorp/api/internal/models"
)

type CinderConfig struct {
	AuthURL       string
	Username      string
	Password      string
	ProjectID     string
	UserDomainID  string
	Region        string
	Interface     string
	VolumeTypeIDs []string
	Zones         []string
}

type CinderDriver struct {
	config     CinderConfig
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
	baseURL    string
}

type cinderVolumeType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cinderVolume struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Size        int64             `json:"size"`
	Status      string            `json:"status"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

func NewCinderDriver(config CinderConfig) (*CinderDriver, error) {
	config.AuthURL = strings.TrimRight(strings.TrimSpace(config.AuthURL), "/")
	config.Username = strings.TrimSpace(config.Username)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.UserDomainID = strings.TrimSpace(config.UserDomainID)
	config.Region = strings.TrimSpace(config.Region)
	config.Interface = strings.TrimSpace(config.Interface)
	for index := range config.VolumeTypeIDs {
		config.VolumeTypeIDs[index] = strings.TrimSpace(config.VolumeTypeIDs[index])
	}
	for index := range config.Zones {
		config.Zones[index] = strings.TrimSpace(config.Zones[index])
	}
	if config.Interface == "" {
		config.Interface = "public"
	}
	if config.UserDomainID == "" {
		config.UserDomainID = "default"
	}
	parsed, err := url.Parse(config.AuthURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		config.Username == "" || config.Password == "" || config.ProjectID == "" ||
		config.Region == "" || len(config.VolumeTypeIDs) == 0 {
		return nil, errors.New("Cinder requires HTTPS Keystone URL, service credentials, project, region, and allowed volume types")
	}
	if config.Interface != "public" && config.Interface != "internal" && config.Interface != "admin" {
		return nil, errors.New("Cinder interface must be public, internal, or admin")
	}
	return &CinderDriver{config: config, httpClient: &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}, nil
}

func (d *CinderDriver) Name() string { return "openstack-cinder" }

func (d *CinderDriver) VolumeTypes(ctx context.Context) ([]VolumeType, error) {
	var response struct {
		VolumeTypes []cinderVolumeType `json:"volume_types"`
	}
	if err := d.requestJSON(ctx, http.MethodGet, "/types", nil, http.StatusOK, &response); err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(d.config.VolumeTypeIDs))
	for _, id := range d.config.VolumeTypeIDs {
		allowed[strings.TrimSpace(id)] = struct{}{}
	}
	types := make([]VolumeType, 0, len(allowed))
	for _, item := range response.VolumeTypes {
		if _, ok := allowed[item.ID]; !ok {
			continue
		}
		encrypted, err := d.volumeTypeEncryption(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		name := item.Name
		if name == "" {
			name = item.ID
		}
		types = append(types, VolumeType{
			ID: item.ID, Name: name, Driver: d.Name(), Region: d.config.Region,
			AvailabilityZones: append([]string{}, d.config.Zones...), Encrypted: encrypted,
		})
	}
	if len(types) != len(allowed) {
		return nil, errors.New("one or more configured Cinder volume types are unavailable")
	}
	return types, nil
}

func (d *CinderDriver) ValidateVolume(
	ctx context.Context, region, zone, storageClass string, encrypted bool,
) error {
	if region != d.config.Region {
		return fmt.Errorf("Cinder is configured for region %q", d.config.Region)
	}
	if zone != "" && zone != "auto" && !contains(d.config.Zones, zone) {
		return errors.New("availability zone is not configured for this storage region")
	}
	types, err := d.VolumeTypes(ctx)
	if err != nil {
		return fmt.Errorf("load configured Cinder volume types: %w", err)
	}
	for _, volumeType := range types {
		if volumeType.ID == storageClass {
			if volumeType.Encrypted != encrypted {
				return errors.New("requested encryption does not match the selected Cinder volume type")
			}
			return nil
		}
	}
	return errors.New("selected Cinder volume type is not enabled")
}

func (d *CinderDriver) EnsureVolume(ctx context.Context, volume models.StorageVolume) (ProviderVolume, error) {
	providerName := cinderResourceName(volume.ID)
	existing, err := d.findVolume(ctx, providerName)
	if err != nil {
		return ProviderVolume{}, err
	}
	if existing != nil {
		if existing.Metadata["atonixcorp_volume_id"] != volume.ID {
			return ProviderVolume{}, errors.New("provider resource name conflicts with another resource")
		}
		return ProviderVolume{ID: existing.ID, Status: normalizeProviderStatus(existing.Status)}, nil
	}
	metadata, err := cinderMetadata(volume)
	if err != nil {
		return ProviderVolume{}, err
	}
	providerVolume := map[string]any{
		"name": providerName, "description": volume.Description,
		"size": volume.SizeGB, "volume_type": volume.StorageClass,
		"metadata": metadata,
	}
	if volume.Description == "" {
		providerVolume["description"] = "AtonixCorp managed block volume"
	}
	if zone := cinderAvailabilityZone(volume.AvailabilityZone); zone != "" {
		providerVolume["availability_zone"] = zone
	}
	body := map[string]any{"volume": providerVolume}
	if volume.AvailabilityZone != "" && volume.AvailabilityZone != "auto" &&
		!contains(d.config.Zones, volume.AvailabilityZone) {
		return ProviderVolume{}, errors.New("availability zone is not configured for this storage region")
	}
	var response struct {
		Volume cinderVolume `json:"volume"`
	}
	if err := d.requestJSON(ctx, http.MethodPost, "/volumes", body, http.StatusAccepted, &response); err != nil {
		return ProviderVolume{}, err
	}
	if response.Volume.ID == "" {
		return ProviderVolume{}, errors.New("Cinder did not return a volume identifier")
	}
	return ProviderVolume{ID: response.Volume.ID, Status: normalizeProviderStatus(response.Volume.Status)}, nil
}

func (d *CinderDriver) InspectVolume(ctx context.Context, providerID string) (ProviderVolume, error) {
	var response struct {
		Volume cinderVolume `json:"volume"`
	}
	err := d.requestJSON(
		ctx, http.MethodGet, "/volumes/"+url.PathEscape(providerID), nil, http.StatusOK, &response,
	)
	if errors.Is(err, errCinderNotFound) {
		return ProviderVolume{}, ErrProviderNotFound
	}
	if err != nil {
		return ProviderVolume{}, err
	}
	return ProviderVolume{ID: response.Volume.ID, Status: normalizeProviderStatus(response.Volume.Status)}, nil
}

func (d *CinderDriver) DeleteVolume(ctx context.Context, volume models.StorageVolume) (bool, error) {
	providerID := volume.ProviderID
	if providerID == "" {
		existing, err := d.findVolume(ctx, cinderResourceName(volume.ID))
		if err != nil {
			return false, err
		}
		if existing == nil {
			return true, nil
		}
		providerID = existing.ID
	}
	current, err := d.InspectVolume(ctx, providerID)
	if errors.Is(err, ErrProviderNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if current.Status == "deleting" {
		return false, nil
	}
	if current.Status == "error_deleting" {
		return false, errors.New("Cinder reports a volume deletion error")
	}
	path := "/volumes/" + url.PathEscape(providerID)
	if err := d.requestJSON(ctx, http.MethodDelete, path, nil, http.StatusAccepted, nil); err != nil {
		if errors.Is(err, errCinderNotFound) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

var errCinderNotFound = errors.New("Cinder resource not found")

func (d *CinderDriver) findVolume(ctx context.Context, name string) (*cinderVolume, error) {
	query := url.Values{"name": []string{name}, "limit": []string{"2"}}
	var response struct {
		Volumes []cinderVolume `json:"volumes"`
	}
	if err := d.requestJSON(
		ctx, http.MethodGet, "/volumes/detail?"+query.Encode(), nil, http.StatusOK, &response,
	); err != nil {
		return nil, err
	}
	if len(response.Volumes) == 0 {
		return nil, nil
	}
	if len(response.Volumes) > 1 {
		return nil, errors.New("Cinder returned multiple resources for a unique volume name")
	}
	return &response.Volumes[0], nil
}

func (d *CinderDriver) volumeTypeEncryption(ctx context.Context, id string) (bool, error) {
	var response struct {
		Encryption struct {
			Provider string `json:"provider"`
		} `json:"encryption"`
	}
	path := "/types/" + url.PathEscape(id) + "/encryption"
	err := d.requestJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &response)
	if errors.Is(err, errCinderNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(response.Encryption.Provider) != "", nil
}

func (d *CinderDriver) requestJSON(
	ctx context.Context,
	method string,
	path string,
	body any,
	expectedStatus int,
	result any,
) error {
	if err := d.ensureAuthenticated(ctx); err != nil {
		return err
	}
	endpoint, err := d.endpoint()
	if err != nil {
		return err
	}
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequestWithContext(
		ctx, method, endpoint+path, bodyReader,
	)
	if err != nil {
		return err
	}
	token := d.currentToken()
	req.Header.Set("X-Auth-Token", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Cinder request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return errCinderNotFound
	}
	if response.StatusCode != expectedStatus {
		return fmt.Errorf("Cinder returned unexpected HTTP status %d", response.StatusCode)
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(result); err != nil {
		return fmt.Errorf("decode Cinder response: %w", err)
	}
	return nil
}

func (d *CinderDriver) ensureAuthenticated(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.token != "" && time.Until(d.expiresAt) > time.Minute {
		return nil
	}
	payload := map[string]any{
		"auth": map[string]any{
			"identity": map[string]any{
				"methods": []string{"password"},
				"password": map[string]any{"user": map[string]any{
					"name": d.config.Username, "password": d.config.Password,
					"domain": map[string]string{"id": d.config.UserDomainID},
				}},
			},
			"scope": map[string]any{"project": map[string]string{"id": d.config.ProjectID}},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	authURL := strings.TrimRight(d.config.AuthURL, "/") + "/auth/tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authenticate to Keystone: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("Keystone authentication returned HTTP status %d", response.StatusCode)
	}
	var result struct {
		Token struct {
			ExpiresAt string `json:"expires_at"`
			Catalog   []struct {
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
	if err := json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(&result); err != nil {
		return fmt.Errorf("decode Keystone token response: %w", err)
	}
	token := response.Header.Get("X-Subject-Token")
	if token == "" {
		return errors.New("Keystone response did not include a scoped token")
	}
	expiresAt, err := time.Parse(time.RFC3339, result.Token.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now()) {
		return errors.New("Keystone returned an invalid token expiry")
	}
	var volumeURL string
	for _, service := range result.Token.Catalog {
		if service.Type != "volumev3" {
			continue
		}
		for _, candidate := range service.Endpoints {
			region := candidate.RegionID
			if region == "" {
				region = candidate.Region
			}
			if region == d.config.Region && candidate.Interface == d.config.Interface {
				volumeURL = candidate.URL
				break
			}
		}
	}
	if volumeURL == "" {
		return errors.New("Keystone catalog has no matching Cinder endpoint")
	}
	volumeURL = strings.NewReplacer(
		"%(project_id)s", d.config.ProjectID,
		"%(tenant_id)s", d.config.ProjectID,
		"{project_id}", d.config.ProjectID,
		"{tenant_id}", d.config.ProjectID,
	).Replace(volumeURL)
	parsed, err := url.Parse(volumeURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("Keystone returned an invalid Cinder endpoint")
	}
	d.token, d.expiresAt, d.baseURL = token, expiresAt, strings.TrimRight(volumeURL, "/")
	return nil
}

func (d *CinderDriver) currentToken() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.token
}

func (d *CinderDriver) endpoint() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.baseURL == "" {
		return "", errors.New("Cinder endpoint is not initialized")
	}
	return d.baseURL, nil
}

func cinderResourceName(id string) string { return "atonix-" + id }

func cinderMetadata(volume models.StorageVolume) (map[string]string, error) {
	metadata := make(map[string]string)
	if len(volume.TagsJSON) > 0 {
		if err := json.Unmarshal(volume.TagsJSON, &metadata); err != nil {
			return nil, fmt.Errorf("decode storage volume metadata: %w", err)
		}
	}
	metadata["atonixcorp_volume_id"] = volume.ID
	metadata["atonixcorp_tenant_id"] = strconv.FormatUint(uint64(volume.TenantID), 10)
	metadata["atonixcorp_project_id"] = strconv.FormatUint(uint64(volume.ProjectID), 10)
	return metadata, nil
}

func cinderAvailabilityZone(zone string) string {
	if zone == "" || zone == "auto" {
		return ""
	}
	return zone
}

func normalizeProviderStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if strings.TrimSpace(candidate) == value {
			return true
		}
	}
	return false
}
