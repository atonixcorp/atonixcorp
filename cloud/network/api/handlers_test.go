package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateNetworkRejectsInvalidCIDR(t *testing.T) {
	handler := NewHandler(nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/networks",
		strings.NewReader(`{"name":"app","cidr":"10.12.1.1/16"}`),
	)
	response := httptest.NewRecorder()

	handler.CreateNetwork(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_NETWORK"`) {
		t.Fatalf("response does not contain stable validation error: %s", response.Body.String())
	}
}

func TestCreateNetworkRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/networks",
		strings.NewReader(`{"name":"app","cidr":"10.12.0.0/16","tenant_id":"tenant-1","admin":true}`),
	)
	response := httptest.NewRecorder()

	handler.CreateNetwork(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
		t.Fatalf("response does not contain stable request error: %s", response.Body.String())
	}
}

func TestCreateSubnetRejectsGatewayOutsideCIDR(t *testing.T) {
	handler := NewHandler(nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/subnets",
		strings.NewReader(`{"network_id":"network-1","cidr":"10.12.0.0/24","gateway_ip":"10.12.1.1"}`),
	)
	response := httptest.NewRecorder()

	handler.CreateSubnet(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_SUBNET"`) {
		t.Fatalf("response does not contain stable validation error: %s", response.Body.String())
	}
}

func TestNetworkOperationErrorsHideProviderDetails(t *testing.T) {
	response := httptest.NewRecorder()
	writeOperationFailure(
		response, "create network", "NETWORK_CREATE_FAILED",
		"The network could not be created.",
		errors.New("Neutron authentication failed at https://provider.example"),
	)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	body := response.Body.String()
	if strings.Contains(body, "Neutron") || strings.Contains(body, "provider.example") {
		t.Fatalf("provider details leaked in customer response: %s", body)
	}
	if !strings.Contains(body, `"code":"NETWORK_CREATE_FAILED"`) ||
		!strings.Contains(body, `"error":"The network could not be created."`) {
		t.Fatalf("response is missing the stable customer error: %s", body)
	}
}
