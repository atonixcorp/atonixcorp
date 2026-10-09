package services

import (
	"atonixcorp/api/internal/config"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/repositories"
	"fmt"
	"net/smtp"
	"strings"
)

type ContactService interface {
	Submit(msg *models.ContactMessage) error
	GetAll() ([]models.ContactMessage, error)
}

type contactService struct {
	repo repositories.ContactRepository
	smtp smtpConfig
}

type smtpConfig struct {
	host      string
	port      string
	user      string
	password  string
	recipient string
}

func NewContactService(repo repositories.ContactRepository, appConfig *config.Config) ContactService {
	return &contactService{
		repo: repo,
		smtp: smtpConfig{
			host:      appConfig.SMTPHost,
			port:      appConfig.SMTPPort,
			user:      appConfig.SMTPUser,
			password:  appConfig.SMTPPassword,
			recipient: appConfig.SMTPRecipient,
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

func (s *contactService) sendNotification(msg *models.ContactMessage) error {
	subject := strings.ReplaceAll(strings.ReplaceAll(msg.Subject, "\r", ""), "\n", "")
	message := fmt.Sprintf(
		"To: %s\r\nFrom: %s\r\nReply-To: %s\r\nSubject: New AtonixCorp inquiry: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nName: %s\nEmail: %s\n\nMessage:\n%s\n",
		s.smtp.recipient,
		s.smtp.user,
		msg.Email,
		subject,
		msg.Name,
		msg.Email,
		msg.Message,
	)

	auth := smtp.PlainAuth("", s.smtp.user, s.smtp.password, s.smtp.host)
	return smtp.SendMail(s.smtp.host+":"+s.smtp.port, auth, s.smtp.user, []string{s.smtp.recipient}, []byte(message))
}
