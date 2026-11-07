package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"atonixcorp/api/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCinderDriverCreateInspectDeleteLifecycle(t *testing.T) {
	var volume *cinderVolume
	var authCalls, createCalls, deleteCalls int
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL: "https://keystone.example/v3", Username: "storage-service",
		Password: "test-secret", ProjectID: "project-123", UserDomainID: "users",
		Region: "region-a", Interface: "public", VolumeTypeIDs: []string{"fast"},
		Zones: []string{"zone-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		respond := func(status int, body string) *http.Response {
			return &http.Response{
				StatusCode: status, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader(body)), Request: request,
			}
		}
		if request.URL.Host == "keystone.example" {
			authCalls++
			response := respond(http.StatusCreated,
				`{"token":{"expires_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`","project":{"id":"project-123"},"catalog":[{"type":"block-storage","endpoints":[{"region_id":"region-a","interface":"public","url":"https://cinder.example/v3/%(project_id)s"}]}]}}`,
			)
			response.Header.Set("X-Subject-Token", "scoped-token")
			return response, nil
		}
		if request.Header.Get("X-Auth-Token") != "scoped-token" {
			t.Errorf("Cinder request missing scoped token")
		}
		switch {
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/types"):
			return respond(http.StatusOK, `{"volume_types":[{"id":"fast","name":"Fast encrypted"}]}`), nil
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/types/fast/encryption"):
			return respond(http.StatusOK, `{"encryption":{"provider":"luks"}}`), nil
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/volumes/detail"):
			if volume != nil {
				return respond(http.StatusOK, `{"volumes":[{"id":"provider-1","name":"`+volume.Name+`","status":"creating","metadata":{"atonixcorp_volume_id":"`+volume.Metadata["atonixcorp_volume_id"]+`"}}]}`), nil
			}
			return respond(http.StatusOK, `{"volumes":[]}`), nil
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/volumes"):
			createCalls++
			if err := jsonNewDecoder(request.Body, &struct {
				Volume map[string]any `json:"volume"`
			}{}); err != nil {
				t.Errorf("decode create request: %v", err)
			}
			volume = &cinderVolume{
				ID: "provider-1", Name: "atonix-volume-id", Status: "creating",
				Metadata: map[string]string{"atonixcorp_volume_id": "volume-id"},
			}
			return respond(http.StatusAccepted, `{"volume":{"id":"provider-1","status":"creating"}}`), nil
		case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/volumes/provider-1"):
			if deleteCalls > 0 {
				return respond(http.StatusNotFound, `{}`), nil
			}
			return respond(http.StatusOK, `{"volume":{"id":"provider-1","status":"in-use","size":20,"availability_zone":"zone-a","attachments":[{"attachment_id":"attach-1","server_id":"server-1","device":"/dev/vdb"}]}}`), nil
		case request.Method == http.MethodDelete && strings.HasSuffix(request.URL.Path, "/volumes/provider-1"):
			deleteCalls++
			return respond(http.StatusAccepted, ""), nil
		default:
			t.Errorf("unexpected Cinder request %s %s", request.Method, request.URL)
			return respond(http.StatusNotFound, `{}`), nil
		}
	})}
	ctx := context.Background()
	types, err := driver.VolumeTypes(ctx)
	if err != nil {
		t.Fatalf("list volume types: %v", err)
	}
	if len(types) != 1 || types[0].ID != "fast" || !types[0].Encrypted || types[0].Region != "region-a" {
		t.Fatalf("unexpected volume types: %+v", types)
	}
	if err := driver.ValidateVolume(ctx, "region-a", "zone-a", "fast", true); err != nil {
		t.Fatalf("validate supported volume: %v", err)
	}
	if err := driver.ValidateVolume(ctx, "region-b", "zone-a", "fast", true); err == nil {
		t.Fatal("expected an unsupported region to be rejected")
	}
	customerVolume := models.StorageVolume{
		ID: "volume-id", TenantID: 1, ProjectID: 2, SizeGB: 10,
		StorageClass: "fast", AvailabilityZone: "zone-a",
	}
	provider, err := driver.EnsureVolume(ctx, customerVolume)
	if err != nil {
		t.Fatalf("ensure volume: %v", err)
	}
	if provider.ID != "provider-1" || provider.Status != "creating" || createCalls != 1 {
		t.Fatalf("unexpected provider result: %+v, create calls %d", provider, createCalls)
	}
	provider, err = driver.EnsureVolume(ctx, customerVolume)
	if err != nil {
		t.Fatalf("idempotent ensure: %v", err)
	}
	if provider.ID != "provider-1" || createCalls != 1 {
		t.Fatalf("retry created another provider volume: %+v, create calls %d", provider, createCalls)
	}
	provider, err = driver.InspectVolume(ctx, "provider-1")
	if err != nil || provider.Status != "in-use" || provider.SizeGB != 20 ||
		provider.AvailabilityZone != "zone-a" || len(provider.Attachments) != 1 ||
		provider.Attachments[0].ID != "attach-1" || provider.Attachments[0].ServerID != "server-1" {
		t.Fatalf("inspect volume = %+v, %v", provider, err)
	}
	customerVolume.ProviderID = "provider-1"
	deleted, err := driver.DeleteVolume(ctx, customerVolume)
	if err != nil || deleted || deleteCalls != 1 {
		t.Fatalf("delete request = %t, %v; calls %d", deleted, err, deleteCalls)
	}
	deleted, err = driver.DeleteVolume(ctx, customerVolume)
	if err != nil || !deleted {
		t.Fatalf("delete retry = %t, %v; want completed", deleted, err)
	}
	if authCalls != 1 {
		t.Fatalf("Keystone auth calls = %d, want cached token (1)", authCalls)
	}
}

func TestCinderUsesConfiguredEncryptionWhenProviderForbidsLookup(t *testing.T) {
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL:                 "https://keystone.example/v3",
		ApplicationCredentialID: "credential-id", ApplicationCredentialSecret: "credential-secret",
		ProjectID: "project-id", Region: "RegionOne", VolumeTypeIDs: []string{"fast"},
		UnencryptedVolumeTypeIDs: []string{"fast"},
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		respond := func(status int, body string) *http.Response {
			return &http.Response{
				StatusCode: status, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader(body)), Request: request,
			}
		}
		if request.URL.Host == "keystone.example" {
			response := respond(http.StatusCreated,
				`{"token":{"expires_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`","project":{"id":"project-id"},"catalog":[{"type":"block-storage","endpoints":[{"region_id":"RegionOne","interface":"public","url":"https://cinder.example/v3"}]}]}}`,
			)
			response.Header.Set("X-Subject-Token", "scoped-token")
			return response, nil
		}
		switch request.URL.Path {
		case "/v3/types":
			return respond(http.StatusOK, `{"volume_types":[{"id":"fast","name":"Fast"}]}`), nil
		case "/v3/types/fast/encryption":
			return respond(http.StatusForbidden, `{}`), nil
		default:
			t.Errorf("unexpected Cinder request %s %s", request.Method, request.URL)
			return respond(http.StatusNotFound, `{}`), nil
		}
	})}

	types, err := driver.VolumeTypes(context.Background())
	if err != nil {
		t.Fatalf("list volume types: %v", err)
	}
	if len(types) != 1 || types[0].ID != "fast" || types[0].Encrypted {
		t.Fatalf("unexpected volume type classification: %+v", types)
	}
}

func TestCinderRejectsWrongTokenProject(t *testing.T) {
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL: "https://keystone.example/v3", ApplicationCredentialID: "credential",
		ApplicationCredentialSecret: "secret", ProjectID: "tenant-project",
		Region: "region", VolumeTypeIDs: []string{"fast"},
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Request: request,
			Body: io.NopCloser(strings.NewReader(`{"token":{"expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","project":{"id":"shared-project"},"catalog":[]}}`))}
		response.Header.Set("X-Subject-Token", "test-token")
		return response, nil
	})}
	if err := driver.ensureAuthenticated(context.Background()); err == nil {
		t.Fatal("wrong project token was accepted")
	}
}

func TestCinderRejectsCatalogEndpointForAnotherProject(t *testing.T) {
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL: "https://keystone.example/v3", ApplicationCredentialID: "credential",
		ApplicationCredentialSecret: "secret", ProjectID: "tenant-project",
		Region: "region", VolumeTypeIDs: []string{"fast"},
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Request: request,
			Body: io.NopCloser(strings.NewReader(`{"token":{"expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","project":{"id":"tenant-project"},"catalog":[{"type":"volumev3","endpoints":[{"region_id":"region","interface":"public","url":"https://cinder.example/v3/shared-project"}]}]}}`))}
		response.Header.Set("X-Subject-Token", "test-token")
		return response, nil
	})}
	if err := driver.ensureAuthenticated(context.Background()); err == nil {
		t.Fatal("foreign project Cinder endpoint was accepted")
	}
}

func TestNewCinderDriverRequiresSafeCompleteConfig(t *testing.T) {
	base := CinderConfig{
		AuthURL: "https://keystone.example/v3", Username: "service", Password: "secret",
		ProjectID: "project", Region: "region", VolumeTypeIDs: []string{"type"},
	}
	if _, err := NewCinderDriver(base); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	base.AuthURL = "http://keystone.example/v3"
	if _, err := NewCinderDriver(base); err == nil {
		t.Fatal("expected plaintext Keystone URL to be rejected")
	}
	base.AllowInsecureHTTP = true
	if _, err := NewCinderDriver(base); err != nil {
		t.Fatalf("explicitly enabled development HTTP rejected: %v", err)
	}
	base.AllowInsecureHTTP = false
	base.AuthURL = "https://user:pass@keystone.example/v3"
	if _, err := NewCinderDriver(base); err == nil {
		t.Fatal("expected credentials in Keystone URL to be rejected")
	}
	base.AuthURL = "https://keystone.example/v3"
	base.VolumeTypeIDs = nil
	if _, err := NewCinderDriver(base); err == nil {
		t.Fatal("expected an explicit volume type allowlist")
	}
	base.VolumeTypeIDs = []string{"type"}
	base.UnencryptedVolumeTypeIDs = []string{"other"}
	if _, err := NewCinderDriver(base); err == nil {
		t.Fatal("expected encryption classifications to use allowed volume types")
	}
}

func TestCinderUsesApplicationCredentialAuth(t *testing.T) {
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL:                     "http://keystone.example/v3",
		ApplicationCredentialID:     "credential-id",
		ApplicationCredentialSecret: "credential-secret",
		ProjectID:                   "project-id", Region: "RegionOne",
		VolumeTypeIDs: []string{"fast"}, AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewCinderDriver(): %v", err)
	}
	var authBody map[string]any
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v3/auth/tokens" {
			t.Fatalf("Keystone request path = %q", request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&authBody); err != nil {
			t.Fatalf("decode Keystone auth request: %v", err)
		}
		response := &http.Response{
			StatusCode: http.StatusCreated, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"token":{"expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) +
					`","project":{"id":"project-id"},"catalog":[{"type":"volumev3","endpoints":[{"region_id":"RegionOne","interface":"public","url":"http://cinder.example/v3/%(project_id)s"}]}]}}`,
			)), Request: request,
		}
		response.Header.Set("X-Subject-Token", "test-token")
		return response, nil
	})}
	if err := driver.ensureAuthenticated(context.Background()); err != nil {
		t.Fatalf("ensureAuthenticated(): %v", err)
	}
	auth, ok := authBody["auth"].(map[string]any)
	if !ok {
		t.Fatalf("missing auth payload: %#v", authBody)
	}
	identity, ok := auth["identity"].(map[string]any)
	if !ok || identity["application_credential"] == nil {
		t.Fatalf("Cinder auth did not use application credential: %#v", authBody)
	}
	if _, exists := auth["scope"]; exists {
		t.Fatalf("application credential auth unexpectedly includes scope: %#v", authBody)
	}
	endpoint, err := driver.endpoint()
	if err != nil || endpoint != "http://cinder.example/v3/project-id" {
		t.Fatalf("discovered Cinder endpoint = %q, error %v", endpoint, err)
	}
}

func jsonNewDecoder(reader io.Reader, target any) error {
	return json.NewDecoder(reader).Decode(target)
}

func TestCinderNotFoundErrorIsProviderNotFound(t *testing.T) {
	driver, err := NewCinderDriver(CinderConfig{
		AuthURL: "https://keystone.example/v3", Username: "service", Password: "secret",
		ProjectID: "project", Region: "region", VolumeTypeIDs: []string{"type"},
	})
	if err != nil {
		t.Fatal(err)
	}
	driver.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := &http.Response{
			StatusCode: http.StatusCreated, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"token":{"expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","project":{"id":"project"},"catalog":[{"type":"volumev3","endpoints":[{"region":"region","interface":"public","url":"https://cinder.example/v3"}]}]}}`,
			)), Request: request,
		}
		if request.URL.Host == "keystone.example" {
			response.Header.Set("X-Subject-Token", "token")
			return response, nil
		}
		response.StatusCode = http.StatusNotFound
		return response, nil
	})}
	_, err = driver.InspectVolume(context.Background(), "missing")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("InspectVolume error = %v, want ErrProviderNotFound", err)
	}
}

func TestFindVolumeQueryEscapesGeneratedName(t *testing.T) {
	name := cinderResourceName("id")
	parsed, err := url.Parse("https://cinder.example/v3/volumes/detail?name=" + url.QueryEscape(name))
	if err != nil || parsed.Query().Get("name") != name {
		t.Fatalf("generated volume query did not round-trip: %v, %v", parsed, err)
	}
}
