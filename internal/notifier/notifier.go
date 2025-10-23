package notifier

import (
	"log/slog"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type Notifier interface {
	NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error
}

const (
	sendGridNotifierType = "sendgrid"
	sMTP2GONotifierType  = "smtp2go"
)

type notifierConfig struct {
	apiKey     string
	fromEmail  string
	templateID string
	clientType string
	recipients []string
}

type NotifierOption func(*notifierConfig)

// WithSendGridAPIKey sets the SendGrid API key for the notifier
func WithSendGridAPIKey(apiKey string) NotifierOption {
	return func(c *notifierConfig) {
		c.apiKey = apiKey
		c.clientType = sendGridNotifierType
	}
}

// WithSMTP2GOAPIKey sets the SMTP2GO API key for the notifier
func WithSMTP2GOAPIKey(apiKey string) NotifierOption {
	return func(c *notifierConfig) {
		c.apiKey = apiKey
		c.clientType = sMTP2GONotifierType
	}
}

// WithSenderEmail sets the sender email for the notifier
func WithSenderEmail(email string) NotifierOption {
	return func(c *notifierConfig) {
		c.fromEmail = email
	}
}

// WithTemplateID sets the template ID for the notifier
func WithTemplateID(templateID string) NotifierOption {
	return func(c *notifierConfig) {
		c.templateID = templateID
	}
}

// WithRecipients sets the recipient emails for the notifier
func WithRecipients(recipients ...string) NotifierOption {
	return func(c *notifierConfig) {
		c.recipients = recipients
	}
}

func NewNotifier(opts ...NotifierOption) (Notifier, error) {

	config := &notifierConfig{}
	for _, opt := range opts {
		opt(config)
	}

	switch config.clientType {
	case sendGridNotifierType:
		return newSendgridNotifier(config)
	case sMTP2GONotifierType:
		return newSMTP2GONotifier(config)
	default:
		slog.Info("Unknown notifier type, giving a standard output notifier")
		return standardOutNotifier{}, nil
	}
}
