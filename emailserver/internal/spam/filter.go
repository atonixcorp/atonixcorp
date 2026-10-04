package spam

import (
	"strings"

	"atonixcorp/emailserver/internal/smtp"
)

type Logger interface {
	Info(...any)
	Error(...any)
}

type Message = smtp.Message

type Filter interface {
	Check(msg *Message) (bool, error) // true = spam
}

type SimpleFilter struct {
	logger      Logger
	blocklist   []string
	suspicious  []string
	maxBodySize int
}

func NewSimpleFilter(logger Logger) *SimpleFilter {
	return &SimpleFilter{
		logger:      logger,
		blocklist:   []string{"spam@example.com", "noreply@bad-domain.com"},
		suspicious:  []string{"viagra", "lottery", "crypto giveaway", "free money"},
		maxBodySize: 10 * 1024 * 1024, // 10 MB
	}
}

func (f *SimpleFilter) Check(msg *Message) (bool, error) {
	bodyStr := strings.ToLower(string(msg.Body))

	for _, b := range f.blocklist {
		if strings.EqualFold(msg.From, b) {
			f.logger.Info("SpamFilter: blocklisted sender:", msg.From)
			return true, nil
		}
	}

	for _, kw := range f.suspicious {
		if strings.Contains(bodyStr, kw) {
			f.logger.Info("SpamFilter: suspicious keyword:", kw)
			return true, nil
		}
	}

	if len(msg.Body) > f.maxBodySize {
		f.logger.Info("SpamFilter: body too large:", len(msg.Body))
		return true, nil
	}

	return false, nil
}
