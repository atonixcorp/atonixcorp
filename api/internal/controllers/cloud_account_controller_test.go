package controllers

import (
	"net/http"
	"testing"
)

func TestAccountManagementRequestIsTLSRequiresTrustedProxy(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://api.example.test/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "203.0.113.10:8443"
	request.Header.Set("X-Forwarded-Proto", "https")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "")

	secure, err := accountManagementRequestIsTLS(request)
	if err != nil {
		t.Fatalf("accountManagementRequestIsTLS() error = %v", err)
	}
	if secure {
		t.Fatal("untrusted forwarded protocol was accepted")
	}
}

func TestAccountManagementRequestIsTLSAcceptsConfiguredProxy(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://api.example.test/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "10.1.2.3:8443"
	request.Header.Set("X-Forwarded-Proto", "https")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "10.1.0.0/16")

	secure, err := accountManagementRequestIsTLS(request)
	if err != nil {
		t.Fatalf("accountManagementRequestIsTLS() error = %v", err)
	}
	if !secure {
		t.Fatal("TLS forwarded by a configured proxy was rejected")
	}
}

func TestAccountManagementRequestIsTLSRejectsInvalidProxyConfig(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://api.example.test/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "10.1.2.3:8443"
	request.Header.Set("X-Forwarded-Proto", "https")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "not-a-cidr")

	if _, err := accountManagementRequestIsTLS(request); err == nil {
		t.Fatal("invalid trusted proxy CIDR was accepted")
	}
}

func TestAccountManagementRequestIsTLSRejectsDuplicateForwardedProto(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://api.example.test/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "10.1.2.3:8443"
	request.Header.Add("X-Forwarded-Proto", "https")
	request.Header.Add("X-Forwarded-Proto", "http")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "10.1.0.0/16")

	secure, err := accountManagementRequestIsTLS(request)
	if err != nil {
		t.Fatalf("accountManagementRequestIsTLS() error = %v", err)
	}
	if secure {
		t.Fatal("duplicate forwarded protocol headers were accepted")
	}
}

func TestAccountManagementRequestAllowsOptInLoopbackHTTP(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://127.0.0.1/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "127.0.0.1:8080"
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_ALLOW_LOCAL_HTTP", "true")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "")

	secure, err := accountManagementRequestIsTLS(request)
	if err != nil {
		t.Fatalf("accountManagementRequestIsTLS() error = %v", err)
	}
	if !secure {
		t.Fatal("opted-in loopback request was rejected")
	}
}

func TestAccountManagementRequestDoesNotAllowRemoteHTTPWithLocalOptIn(t *testing.T) {
	request, _ := http.NewRequest(http.MethodPut, "http://api.example.test/internal/cloud/accounts/1", nil)
	request.RemoteAddr = "192.0.2.10:8080"
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_ALLOW_LOCAL_HTTP", "true")
	t.Setenv("OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS", "")

	secure, err := accountManagementRequestIsTLS(request)
	if err != nil {
		t.Fatalf("accountManagementRequestIsTLS() error = %v", err)
	}
	if secure {
		t.Fatal("local HTTP opt-in accepted a non-loopback peer")
	}
}
