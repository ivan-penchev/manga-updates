package notifier

import (
	"log/slog"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type Notifier interface {
	NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error
}

type notifierConfig struct {
	apiKey     string
	fromEmail  string
	templateID string
	recipients []string
}

type notifierOption func(*notifierConfig)

// WithSendGridAPIKey sets the SendGrid API key for the notifier
func WithSendGridAPIKey(apiKey string) notifierOption {
	return func(c *notifierConfig) {
		c.apiKey = apiKey
	}
}

// WithSenderEmail sets the sender email for the notifier
func WithSenderEmail(email string) notifierOption {
	return func(c *notifierConfig) {
		c.fromEmail = email
	}
}

// WithTemplateID sets the template ID for the notifier
func WithTemplateID(templateID string) notifierOption {
	return func(c *notifierConfig) {
		c.templateID = templateID
	}
}

// WithRecipients sets the recipient emails for the notifier
func WithRecipients(recipients ...string) notifierOption {
	return func(c *notifierConfig) {
		c.recipients = recipients
	}
}

func NewNotifier(opts ...notifierOption) (Notifier, error) {

	config := &notifierConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.apiKey == "" {
		slog.Info("apiKey is empty, giving a standard output notifier")
		return standardOutNotifier{}, nil
	}

	return newSendgridNotifier(config)
}
