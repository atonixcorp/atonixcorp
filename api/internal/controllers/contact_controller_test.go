package controllers

import (
	"atonixcorp/api/internal/models"
	"math"
	"strings"
	"testing"
)

func validSponsorshipInquiry() models.SponsorshipInquiry {
	return models.SponsorshipInquiry{
		Name:            "Ada Sponsor",
		Company:         "Example Foundation",
		Email:           "ada@example.org",
		Phone:           "+1 555 0100",
		Country:         "United States",
		Amount:          100,
		Package:         "silver",
		PaymentProvider: "stripe",
		Message:         "Supporting open source.",
	}
}

func TestValidateSponsorshipInquiry(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*models.SponsorshipInquiry)
		wantErr bool
	}{
		{name: "valid inquiry"},
		{
			name: "invalid email",
			change: func(inquiry *models.SponsorshipInquiry) {
				inquiry.Email = "not-an-email"
			},
			wantErr: true,
		},
		{
			name: "invalid amount",
			change: func(inquiry *models.SponsorshipInquiry) {
				inquiry.Amount = math.NaN()
			},
			wantErr: true,
		},
		{
			name: "unsupported package",
			change: func(inquiry *models.SponsorshipInquiry) {
				inquiry.Package = "diamond"
			},
			wantErr: true,
		},
		{
			name: "oversized message",
			change: func(inquiry *models.SponsorshipInquiry) {
				inquiry.Message = strings.Repeat("x", 10001)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inquiry := validSponsorshipInquiry()
			if test.change != nil {
				test.change(&inquiry)
			}

			err := validateSponsorshipInquiry(inquiry)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateSponsorshipInquiry() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
