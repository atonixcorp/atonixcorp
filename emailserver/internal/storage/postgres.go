package storage

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"atonixcorp/emailserver/internal/imap"
	"atonixcorp/emailserver/internal/pop3"
	"atonixcorp/emailserver/internal/smtp"
	"github.com/google/uuid"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// SaveIncoming persists a canonical message payload.
func (s *PostgresStore) SaveIncoming(msg *Message) error {
	msg.ID = uuid.NewString()
	if msg.Date.IsZero() {
		msg.Date = time.Now()
	}
	msg.Size = len(msg.Raw)

	_, err := s.db.Exec(`
        INSERT INTO messages (id, user_id, mailbox, subject, from_addr, to_addr, date, body, raw, size)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
    `,
		msg.ID,
		msg.UserID,
		msg.Mailbox,
		msg.Subject,
		msg.From,
		msg.To,
		msg.Date,
		msg.Body,
		msg.Raw,
		msg.Size,
	)
	return err
}

func (s *PostgresStore) SaveIncomingMessage(msg *smtp.Message) error {
	incoming := &Message{
		ID:      uuid.NewString(),
		UserID:  "",
		Mailbox: "INBOX",
		Subject: msg.Subject,
		From:    msg.From,
		To:      strings.Join(msg.To, ", "),
		Date:    time.Now(),
		Body:    msg.Body,
		Raw:     msg.Raw,
		Size:    len(msg.Raw),
	}
	return s.SaveIncoming(incoming)
}

// ListMessages satisfies the IMAP storage contract.
func (s *PostgresStore) ListMessages(userID, mailbox string) ([]imap.Message, error) {
	rows, err := s.db.Query(`
        SELECT id, body
        FROM messages
        WHERE user_id=$1 AND mailbox=$2
        ORDER BY date DESC
    `, userID, mailbox)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []imap.Message
	for rows.Next() {
		var msg imap.Message
		if err := rows.Scan(&msg.ID, &msg.Body); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *PostgresStore) GetMessageBody(userID, mailbox, id string) ([]byte, error) {
	var body []byte
	err := s.db.QueryRow(`
        SELECT body
        FROM messages
        WHERE user_id=$1 AND mailbox=$2 AND id=$3
    `, userID, mailbox, id).Scan(&body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *PostgresStore) GetMessage(userID, mailbox, id string) (*Message, error) {
	var m Message
	err := s.db.QueryRow(`
        SELECT id, user_id, mailbox, subject, from_addr, to_addr, date, body, raw, size
        FROM messages
        WHERE user_id=$1 AND mailbox=$2 AND id=$3
    `, userID, mailbox, id).Scan(
		&m.ID,
		&m.UserID,
		&m.Mailbox,
		&m.Subject,
		&m.From,
		&m.To,
		&m.Date,
		&m.Body,
		&m.Raw,
		&m.Size,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStore) DeleteMessage(userID, mailbox, id string) error {
	res, err := s.db.Exec(`
        DELETE FROM messages
        WHERE user_id=$1 AND mailbox=$2 AND id=$3
    `, userID, mailbox, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *PostgresStore) ListPOP3Messages(userID string) ([]pop3.Message, error) {
	rows, err := s.db.Query(`
        SELECT id, body, size
        FROM messages
        WHERE user_id=$1
        ORDER BY date DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []pop3.Message
	for rows.Next() {
		var msg pop3.Message
		if err := rows.Scan(&msg.ID, &msg.Body, &msg.Size); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *PostgresStore) GetPOP3Message(userID, id string) ([]byte, error) {
	var body []byte
	err := s.db.QueryRow(`
        SELECT body
        FROM messages
        WHERE user_id=$1 AND id=$2
    `, userID, id).Scan(&body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *PostgresStore) DeletePOP3Message(userID, id string) error {
	_, err := s.db.Exec(`
        DELETE FROM messages
        WHERE user_id=$1 AND id=$2
    `, userID, id)
	return err
}

func (s *PostgresStore) ListMailboxes(userID string) ([]imap.Mailbox, error) {
	rows, err := s.db.Query(`
        SELECT name
        FROM mailboxes
        WHERE user_id=$1
        ORDER BY name
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []imap.Mailbox
	for rows.Next() {
		var m imap.Mailbox
		if err := rows.Scan(&m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *PostgresStore) CreateMailbox(userID, name string) error {
	_, err := s.db.Exec(`
        INSERT INTO mailboxes (user_id, name)
        VALUES ($1, $2)
    `, userID, name)
	return err
}

func (s *PostgresStore) DeleteMailbox(userID, name string) error {
	_, err := s.db.Exec(`
        DELETE FROM mailboxes
        WHERE user_id=$1 AND name=$2
    `, userID, name)
	return err
}
