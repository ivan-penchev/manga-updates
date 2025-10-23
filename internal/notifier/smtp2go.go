package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

// this is a blatant ripoff of https://github.com/smtp2go-oss/smtp2go-go
// unfortunately their library is not modular enough
// they do not expose the email client in a way that would allow us to use it here with our api key
// so we reimplement the necessary parts here
type smpt2goEmail struct {
	From          string                      `json:"sender"`
	To            []string                    `json:"to"`
	Cc            []string                    `json:"cc"`
	Bcc           []string                    `json:"bcc"`
	Subject       string                      `json:"subject"`
	TextBody      string                      `json:"text_body"`
	HtmlBody      string                      `json:"html_body"`
	TemplateID    string                      `json:"template_id"`
	TemplateData  interface{}                 `json:"template_data"`
	CustomHeaders []*smpt2goEmailCustomHeader `json:"custom_headers"`
	Attachments   []*smpt2goEmailBinaryData   `json:"attachments"`
	Inlines       []*smpt2goEmailBinaryData   `json:"inlines"`
}

type smpt2goEmailBinaryData struct {
	Filename string `json:"filename"`
	Fileblob string `json:"fileblob"`
	URL      string `json:"url"`
	MimeType string `json:"mimetype"`
}

type smpt2goEmailCustomHeader struct {
	Header string `json:"header"`
	Value  string `json:"value"`
}

type Smtp2goApiResult struct {
	RequestID string `json:"request_id"`

	Data struct {
		Error string `json:"error"`

		FieldValidationErrors struct {
			FieldName string `json:"field_name"`

			Message string `json:"message"`
		} `json:"field_validation_errors"`
	} `json:"data"`
}

func newSMTP2GONotifier(config *notifierConfig) (Notifier, error) {
	if config.fromEmail == "" || !isValidEmail(config.fromEmail) {
		return nil, fmt.Errorf("invalid sender email: %s", config.fromEmail)
	}

	if len(config.recipients) == 0 {
		return nil, fmt.Errorf("no recipient emails provided")
	}
	for _, recipient := range config.recipients {
		if !isValidEmail(recipient) {
			return nil, fmt.Errorf("invalid recipient email: %s", recipient)
		}
	}

	return smtp2goNotifier{
		config: config,
	}, nil
}

type smtp2goNotifier struct {
	config *notifierConfig
}

func (s smtp2goNotifier) sendAPIRequest(endpoint string, requestBody io.Reader) (*Smtp2goApiResult, error) {
	apiRoot := "https://api.smtp2go.com/v3" // Default API root

	client := &http.Client{}
	req, err := http.NewRequest("POST", apiRoot+"/"+endpoint, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Smtp2go-Api", "smtp2go-go")
	req.Header.Add("X-Smtp2go-Api-Version", "1.0.4")
	req.Header.Add("X-Smtp2go-Api-Key", s.config.apiKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	ret := new(Smtp2goApiResult)
	err = json.NewDecoder(res.Body).Decode(ret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return ret, nil
}

func (s smtp2goNotifier) NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error {
	fromEmail := fmt.Sprintf("Manga Notify <%s>", s.config.fromEmail)
	toEmails := make([]string, len(s.config.recipients))
	for i, recipient := range s.config.recipients {
		toEmails[i] = fmt.Sprintf("Recipient <%s>", recipient)
	}

	subject := fmt.Sprintf("%s update", fromManga.Name)

	var htmlBody, textBody string
	var templateData map[string]string

	email := smpt2goEmail{
		From:    fromEmail,
		To:      toEmails,
		Subject: subject,
	}

	if s.config.templateID != "" {
		templateData = map[string]string{
			"manga_name":   fromManga.Name,
			"chapter":      fmt.Sprintf("%.0f", *chapter.Number),
			"chapter_link": chapter.URI,
			"subject":      subject,
		}
		email.TemplateID = s.config.templateID
		email.TemplateData = templateData
	} else {
		htmlBody = fmt.Sprintf("<h1>%s Update!</h1><p>Chapter %s is now available.</p><p>Read it here: <a href=\"%s\">%s</a></p>", fromManga.Name, fmt.Sprintf("%.0f", *chapter.Number), chapter.URI, chapter.URI)
		textBody = fmt.Sprintf("%s Update! Chapter %s is now available. Read it here: %s", fromManga.Name, fmt.Sprintf("%.0f", *chapter.Number), chapter.URI)
		email.HtmlBody = htmlBody
		email.TextBody = textBody
	}

	reqJSON, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	res, err := s.sendAPIRequest("email/send", bytes.NewBuffer(reqJSON))
	if err != nil {
		return err
	}

	if res.Data.Error != "" {
		fieldError := ""
		if res.Data.FieldValidationErrors.FieldName != "" {
			fieldError = res.Data.FieldValidationErrors.Message + "/ "
		}
		return fmt.Errorf("%s - %s", fieldError, res.Data.Error)
	}

	return nil
}
