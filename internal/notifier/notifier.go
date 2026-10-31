package notifier

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type Notifier interface {
	NotifyForNewChapter(ctx context.Context, chapter domain.ChapterEntity, fromManga domain.MangaEntity) error
}

const (
	sendGridNotifierType  = "sendgrid"
	sMTP2GONotifierType   = "smtp2go"
	defaultHTTPTimeout    = 15 * time.Second
	defaultSMTP2GOAPIRoot = "https://api.smtp2go.com/v3"
)

type notifierConfig struct {
	apiKey     string
	fromEmail  string
	templateID string
	clientType string
	recipients []string
	httpClient *http.Client
	apiRoot    string
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

// WithHTTPClient sets a custom HTTP client for HTTP-backed notifiers.
func WithHTTPClient(httpClient *http.Client) NotifierOption {
	return func(c *notifierConfig) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithAPIRoot sets a custom API base URL for HTTP-backed notifiers.
func WithAPIRoot(apiRoot string) NotifierOption {
	return func(c *notifierConfig) {
		if apiRoot != "" {
			c.apiRoot = apiRoot
		}
	}
}

func formatChapterNumber(number *float64) string {
	if number == nil {
		return "unknown"
	}
	return strconv.FormatFloat(*number, 'f', -1, 64)
}

func NewNotifier(opts ...NotifierOption) (Notifier, error) {
	config := &notifierConfig{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		apiRoot:    defaultSMTP2GOAPIRoot,
	}
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
