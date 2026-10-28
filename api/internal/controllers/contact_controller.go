package controllers

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/services"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ContactController struct {
	service services.ContactService
}

type supportTicketRequest struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	Category          string `json:"category"`
	Priority          string `json:"priority"`
	Subject           string `json:"subject"`
	Description       string `json:"description"`
	ResourceReference string `json:"resource_reference"`
}

type supportTicketSummary struct {
	Reference string    `json:"reference"`
	Category  string    `json:"category"`
	Priority  string    `json:"priority"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewContactController(service services.ContactService) *ContactController {
	return &ContactController{service: service}
}

func (c *ContactController) Submit(ctx *gin.Context) {
	var msg models.ContactMessage

	if err := ctx.BindJSON(&msg); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := c.service.Submit(&msg); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Message received"})
}

func (c *ContactController) SubmitSponsorship(ctx *gin.Context) {
	var request models.SponsorshipInquiry
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := validateSponsorshipInquiry(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := fmt.Sprintf(
		"Company: %s\nPhone: %s\nCountry: %s\nAmount: USD %.2f per month\nPackage: %s\nPayment preference: %s\n\nMessage:\n%s",
		request.Company,
		request.Phone,
		request.Country,
		request.Amount,
		request.Package,
		request.PaymentProvider,
		request.Message,
	)
	inquiry := &models.ContactMessage{
		Name:    strings.TrimSpace(request.Name),
		Email:   strings.TrimSpace(request.Email),
		Subject: "Sponsorship request",
		Message: message,
	}

	if err := c.service.SubmitSponsorship(inquiry); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process sponsorship request"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Sponsorship request received"})
}

func validateSponsorshipInquiry(request models.SponsorshipInquiry) error {
	email, err := mail.ParseAddress(strings.TrimSpace(request.Email))
	if err != nil || len(request.Email) > 255 || email.Address != strings.TrimSpace(request.Email) {
		return fmt.Errorf("Enter a valid email address.")
	}
	if !requiredWithin(request.Name, 255) || !requiredWithin(request.Country, 100) {
		return fmt.Errorf("Enter your name and country within the allowed lengths.")
	}
	if len(request.Company) > 255 || len(request.Phone) > 50 || len(request.Package) > 50 {
		return fmt.Errorf("One or more fields exceed the allowed length.")
	}
	if math.IsNaN(request.Amount) || math.IsInf(request.Amount, 0) || request.Amount <= 0 || request.Amount > 10000000 {
		return fmt.Errorf("Enter a monthly sponsorship amount between 1 and 10,000,000.")
	}
	if len(request.Message) > 10000 {
		return fmt.Errorf("Keep your message under 10,000 characters.")
	}
	switch request.PaymentProvider {
	case "payfast", "stripe", "paypal", "bank-transfer", "not-sure":
	default:
		return fmt.Errorf("Choose a supported payment preference.")
	}
	switch request.Package {
	case "", "bronze", "silver", "gold", "platinum":
	default:
		return fmt.Errorf("Choose a supported sponsorship package.")
	}

	return nil
}

func requiredWithin(value string, max int) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && len(trimmed) <= max
}

func (c *ContactController) GetAll(ctx *gin.Context) {
	messages, err := c.service.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

func (c *ContactController) SubmitSupportTicket(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var request supportTicketRequest
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	if err := ctx.ShouldBindJSON(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "support ticket request exceeds the allowed size"})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid support ticket request"})
		return
	}
	if err := validateSupportTicket(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticket := &models.SupportTicket{
		UserID:            userID,
		Name:              strings.TrimSpace(request.Name),
		Email:             strings.TrimSpace(request.Email),
		Category:          request.Category,
		Priority:          request.Priority,
		Subject:           strings.TrimSpace(request.Subject),
		Description:       strings.TrimSpace(request.Description),
		ResourceReference: strings.TrimSpace(request.ResourceReference),
	}
	notificationSent, err := c.service.SubmitSupportTicket(ticket)
	if err != nil {
		log.Printf("save support ticket: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "support ticket could not be saved"})
		return
	}

	notificationStatus := "sent"
	if !notificationSent {
		notificationStatus = "not_sent"
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"ticket": gin.H{
			"reference":  ticket.Reference,
			"status":     ticket.Status,
			"created_at": ticket.CreatedAt,
			"category":   ticket.Category,
			"priority":   ticket.Priority,
			"subject":    ticket.Subject,
		},
		"notification_status": notificationStatus,
	})
}

func (c *ContactController) ListSupportTickets(ctx *gin.Context) {
	userID, ok := authenticatedUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	tickets, err := c.service.ListSupportTickets(userID)
	if err != nil {
		log.Printf("list support tickets: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "support tickets could not be loaded"})
		return
	}

	result := make([]supportTicketSummary, 0, len(tickets))
	for _, ticket := range tickets {
		result = append(result, supportTicketSummary{
			Reference: ticket.Reference,
			Category:  ticket.Category,
			Priority:  ticket.Priority,
			Subject:   ticket.Subject,
			Status:    ticket.Status,
			CreatedAt: ticket.CreatedAt,
			UpdatedAt: ticket.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, gin.H{"tickets": result})
}

func validateSupportTicket(request supportTicketRequest) error {
	email := strings.TrimSpace(request.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || len(email) > 255 || address.Address != email {
		return fmt.Errorf("enter a valid email address")
	}
	if !requiredWithin(request.Name, 255) ||
		!requiredWithin(request.Subject, 255) ||
		!requiredWithin(request.Description, 10000) {
		return fmt.Errorf("name, subject, and description are required and must be within the allowed lengths")
	}
	if strings.ContainsAny(request.Subject, "\r\n") || len(request.ResourceReference) > 255 {
		return fmt.Errorf("subject or resource reference is invalid")
	}
	switch request.Category {
	case "compute", "containers", "storage", "databases", "networking", "security", "billing", "account", "other":
	default:
		return fmt.Errorf("choose a supported support category")
	}
	switch request.Priority {
	case "low", "normal", "high", "urgent":
	default:
		return fmt.Errorf("choose a supported priority")
	}
	return nil
}
