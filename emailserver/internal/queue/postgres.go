package queue

import (
	"database/sql"
	"errors"
	"sync"
	"time"
)

type PostgresQueueStore struct {
	mu      sync.Mutex
	db      *sql.DB
	pending map[string]OutgoingMessage
}

func NewPostgresQueueStore(db *sql.DB) *PostgresQueueStore {
	return &PostgresQueueStore{db: db, pending: make(map[string]OutgoingMessage)}
}

func (s *PostgresQueueStore) SaveOutgoing(msg *OutgoingMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg.ID == "" {
		msg.ID = time.Now().UTC().Format(time.RFC3339Nano)
	}
	s.pending[msg.ID] = *msg
	return nil
}

func (s *PostgresQueueStore) ListPending(limit int) ([]OutgoingMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]OutgoingMessage, 0, len(s.pending))
	for _, msg := range s.pending {
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *PostgresQueueStore) MarkDelivered(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pending[id]; !ok {
		return errors.New("not found")
	}
	delete(s.pending, id)
	return nil
}

func (s *PostgresQueueStore) MarkFailed(id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pending[id]; !ok {
		return errors.New("not found")
	}
	delete(s.pending, id)
	return nil
}

func (s *PostgresQueueStore) UpdateRetry(id string, attempts int, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg, ok := s.pending[id]
	if !ok {
		return errors.New("not found")
	}
	msg.Attempts = attempts
	msg.NextRetry = next
	s.pending[id] = msg
	return nil
}
