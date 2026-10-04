package queue

import (
	"time"

	"atonixcorp/emailserver/internal/smtp"
)

type Logger interface {
	Info(...any)
	Error(...any)
}

type OutgoingMessage struct {
	ID        string
	From      string
	To        []string
	Subject   string
	Body      []byte
	Raw       []byte
	Tenant    string
	Attempts  int
	MaxRetry  int
	NextRetry time.Time
}

type Store interface {
	SaveOutgoing(msg *OutgoingMessage) error
	ListPending(limit int) ([]OutgoingMessage, error)
	MarkDelivered(id string) error
	MarkFailed(id string, err error) error
	UpdateRetry(id string, attempts int, next time.Time) error
}

type Queue struct {
	store  Store
	logger Logger
}

func New(store Store, logger Logger) *Queue {
	return &Queue{store: store, logger: logger}
}

func (q *Queue) EnqueueOutgoing(msg *smtp.Message) error {
	outgoing := &OutgoingMessage{
		ID:      time.Now().UTC().Format(time.RFC3339Nano),
		From:    msg.From,
		To:      msg.To,
		Subject: msg.Subject,
		Body:    msg.Body,
		Raw:     msg.Raw,
		Attempts: 0,
		MaxRetry: 5,
		NextRetry: time.Now(),
	}

	q.logger.Info("Queue: enqueue outgoing message ", outgoing.ID)
	return q.store.SaveOutgoing(outgoing)
}
