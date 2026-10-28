package containers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeRegistryRequestRequiresSecureURLAndPairedCredentials(t *testing.T) {
	tests := []struct {
		name    string
		request RegistryRequest
		wantErr bool
	}{
		{name: "https registry", request: RegistryRequest{Name: " production ", URL: " https://registry.example.com/ "}},
		{name: "insecure scheme", request: RegistryRequest{Name: "production", URL: "http://registry.example.com"}, wantErr: true},
		{name: "embedded credentials", request: RegistryRequest{Name: "production", URL: "https://user:pass@registry.example.com"}, wantErr: true},
		{name: "query credentials", request: RegistryRequest{Name: "production", URL: "https://registry.example.com?token=secret"}, wantErr: true},
		{name: "unpaired username", request: RegistryRequest{Name: "production", URL: "https://registry.example.com", Username: "robot"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := normalizeRegistryRequest(&test.request)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeRegistryRequest() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && (test.request.Name != "production" || test.request.URL != "https://registry.example.com") {
				t.Fatalf("request was not normalized: %+v", test.request)
			}
		})
	}
}

func TestRegistryResponseDoesNotSerializeCredentials(t *testing.T) {
	response := RegistryResponse{
		ID: 1, Name: "prod", URL: "https://registry.example.com", CredentialsStored: true,
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "password") || strings.Contains(string(encoded), "username") {
		t.Fatalf("registry response exposed credential fields: %s", encoded)
	}
}
