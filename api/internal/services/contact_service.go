package services

import (
	"atonixcorp/api/internal/config"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type ContactService interface {
	Submit(msg *models.ContactMessage) error
	SubmitSponsorship(msg *models.ContactMessage) error
	GetAll() ([]models.ContactMessage, error)
	SubmitSupportTicket(ticket *models.SupportTicket) (bool, error)
	ListSupportTickets(userID uint) ([]models.SupportTicket, error)
}

type contactService struct {
	repo repositories.ContactRepository
	smtp smtpConfig
}

type smtpConfig struct {
	host             string
	port             string
	user             string
	password         string
	recipient        string
	sponsorRecipient string
}

func NewContactService(repo repositories.ContactRepository, appConfig *config.Config) ContactService {
	return &contactService{
		repo: repo,
		smtp: smtpConfig{
			host:             appConfig.SMTPHost,
			port:             appConfig.SMTPPort,
			user:             appConfig.SMTPUser,
			password:         appConfig.SMTPPassword,
			recipient:        appConfig.SMTPRecipient,
			sponsorRecipient: appConfig.SponsorRecipient,
		},
	}
}

func (s *contactService) Submit(msg *models.ContactMessage) error {
	if err := s.repo.Save(msg); err != nil {
		return err
	}

	return s.sendNotification(msg)
}

func (s *contactService) GetAll() ([]models.ContactMessage, error) {
	return s.repo.GetAll()
}

func (s *contactService) SubmitSupportTicket(ticket *models.SupportTicket) (bool, error) {
	var referenceBytes [6]byte
	if _, err := rand.Read(referenceBytes[:]); err != nil {
		return false, fmt.Errorf("generate support ticket reference: %w", err)
	}
	ticket.Reference = "ATX-" + strings.ToUpper(hex.EncodeToString(referenceBytes[:]))
	ticket.Status = "open"

	if err := s.repo.SaveSupportTicket(ticket); err != nil {
		return false, err
	}

	body := fmt.Sprintf(
		"Ticket: %s\nStatus: %s\nCategory: %s\nPriority: %s\nResource: %s\nName: %s\nEmail: %s\n\nSubject: %s\n\nDescription:\n%s\n",
		ticket.Reference,
		ticket.Status,
		ticket.Category,
		ticket.Priority,
		ticket.ResourceReference,
		ticket.Name,
		ticket.Email,
		ticket.Subject,
		ticket.Description,
	)
	if err := s.sendMail(
		s.smtp.recipient,
		fmt.Sprintf("AtonixCorp support ticket %s: %s", ticket.Reference, ticket.Subject),
		body,
		ticket.Email,
	); err != nil {
		log.Printf("support ticket %s saved, but notification delivery failed: %v", ticket.Reference, err)
		return false, nil
	}

	return true, nil
}

func (s *contactService) ListSupportTickets(userID uint) ([]models.SupportTicket, error) {
	return s.repo.ListSupportTickets(userID)
}

func (s *contactService) sendNotification(msg *models.ContactMessage) error {
	subject := strings.ReplaceAll(strings.ReplaceAll(msg.Subject, "\r", ""), "\n", "")
	message := fmt.Sprintf("Name: %s\nEmail: %s\n\nMessage:\n%s\n", msg.Name, msg.Email, msg.Message)
	return s.sendMail(s.smtp.recipient, "New AtonixCorp inquiry: "+subject, message, msg.Email)
}

func (s *contactService) SubmitSponsorship(msg *models.ContactMessage) error {
	if err := s.repo.Save(msg); err != nil {
		return err
	}

	adminBody := fmt.Sprintf(
		"Name: %s\nEmail: %s\n\nSponsorship request:\n%s\n",
		msg.Name,
		msg.Email,
		msg.Message,
	)
	adminErr := s.sendMail(
		s.smtp.sponsorRecipient,
		"New AtonixCorp sponsorship request",
		adminBody,
		msg.Email,
	)

	confirmationBody := fmt.Sprintf(
		"Hello %s,\n\nThank you for your interest in sponsoring AtonixCorp. We received your request and our team will follow up with you.\n\nYour request:\n%s\n\nAtonixCorp Sponsorship Team\n",
		msg.Name,
		msg.Message,
	)
	confirmationErr := s.sendMail(msg.Email, "We received your AtonixCorp sponsorship request", confirmationBody, "")

	return errors.Join(adminErr, confirmationErr)
}

func (s *contactService) sendMail(recipient, subject, body, replyTo string) error {
	message := fmt.Sprintf(
		"To: %s\r\nFrom: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n",
		recipient,
		s.smtp.user,
		subject,
	)
	if replyTo != "" {
		message += fmt.Sprintf("Reply-To: %s\r\n", replyTo)
	}
	message += "\r\n" + body
	auth := smtp.PlainAuth("", s.smtp.user, s.smtp.password, s.smtp.host)
	return smtp.SendMail(s.smtp.host+":"+s.smtp.port, auth, s.smtp.user, []string{recipient}, []byte(message))
}
