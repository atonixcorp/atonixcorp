package controllers

import (
	"strings"
	"testing"
)

func validSupportTicketRequest() supportTicketRequest {
	return supportTicketRequest{
		Name:        "Ada Lovelace",
		Email:       "ada@example.org",
		Category:    "compute",
		Priority:    "normal",
		Subject:     "Instance does not start",
		Description: "The instance remains in a pending state.",
	}
}

func TestValidateSupportTicket(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*supportTicketRequest)
		wantErr bool
	}{
		{name: "valid ticket"},
		{
			name: "invalid email",
			change: func(request *supportTicketRequest) {
				request.Email = "invalid"
			},
			wantErr: true,
		},
		{
			name: "unsupported category",
			change: func(request *supportTicketRequest) {
				request.Category = "platform-admin"
			},
			wantErr: true,
		},
		{
			name: "unsupported priority",
			change: func(request *supportTicketRequest) {
				request.Priority = "emergency"
			},
			wantErr: true,
		},
		{
			name: "header injection in subject",
			change: func(request *supportTicketRequest) {
				request.Subject = "Help\r\nBcc: attacker@example.org"
			},
			wantErr: true,
		},
		{
			name: "oversized description",
			change: func(request *supportTicketRequest) {
				request.Description = strings.Repeat("x", 10001)
			},
			wantErr: true,
		},
		{
			name: "oversized resource reference",
			change: func(request *supportTicketRequest) {
				request.ResourceReference = strings.Repeat("r", 256)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validSupportTicketRequest()
			if test.change != nil {
				test.change(&request)
			}
			if err := validateSupportTicket(request); (err != nil) != test.wantErr {
				t.Fatalf("validateSupportTicket() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
