package repositories

import (
	"atonixcorp/api/internal/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListSupportTicketsOnlyReturnsRequestingUsersTickets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.SupportTicket{}); err != nil {
		t.Fatalf("migrate support ticket model: %v", err)
	}

	repo := NewContactRepository(db)
	for _, ticket := range []models.SupportTicket{
		{Reference: "ATX-USERONE01", UserID: 1, Name: "One", Email: "one@example.org", Category: "compute", Priority: "normal", Subject: "Ticket one", Description: "Details", Status: "open"},
		{Reference: "ATX-USERTWO01", UserID: 2, Name: "Two", Email: "two@example.org", Category: "storage", Priority: "low", Subject: "Ticket two", Description: "Details", Status: "open"},
	} {
		if err := repo.SaveSupportTicket(&ticket); err != nil {
			t.Fatalf("save support ticket: %v", err)
		}
	}

	tickets, err := repo.ListSupportTickets(1)
	if err != nil {
		t.Fatalf("list support tickets: %v", err)
	}
	if len(tickets) != 1 || tickets[0].Reference != "ATX-USERONE01" {
		t.Fatalf("ListSupportTickets() = %#v, want only user one's ticket", tickets)
	}
}
