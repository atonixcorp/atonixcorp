package storage

import (
	"errors"
	"testing"
)

func TestNormalizeVolumeRequestValidatesMetadata(t *testing.T) {
	tests := []struct {
		name    string
		request CreateVolumeRequest
		wantErr bool
	}{
		{
			name: "valid development volume",
			request: CreateVolumeRequest{
				Name: " production-db ", Region: "local-development",
				StorageClass: "local-file", SizeGB: 50,
				Tags: map[string]string{"team": "platform"},
			},
		},
		{
			name: "invalid name",
			request: CreateVolumeRequest{
				Name: "../volume", Region: "local-development",
				StorageClass: "local-file", SizeGB: 50,
			},
			wantErr: true,
		},
		{
			name: "oversized metadata",
			request: CreateVolumeRequest{
				Name: "production-db", Region: "local-development",
				StorageClass: "local-file", SizeGB: 50,
				Tags: map[string]string{"": "bad"},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := normalizeVolumeRequest(&test.request)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeVolumeRequest() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.name == "valid development volume" && test.request.Name != "production-db" {
				t.Fatalf("name not normalized: %q", test.request.Name)
			}
			if test.wantErr && err != nil && !errors.Is(err, ErrInvalid) {
				t.Fatalf("validation error is not categorized: %v", err)
			}
		})
	}
}
