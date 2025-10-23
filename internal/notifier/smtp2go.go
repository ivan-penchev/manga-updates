package notifier

import (
	"fmt"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

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

func (s smtp2goNotifier) NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error {
	// 	// SMTP2GO server details
	// 	smtpHost := "mail.smtp2go.com"
	// 	smtpPort := "2525" // or 8025, 587, 443

	// 	// Authentication
	// 	auth := smtp.PlainAuth("", s.config.fromEmail, s.config.apiKey, smtpHost)

	// 	// Prepare email content
	// 	to := s.config.recipients
	// 	subject := fmt.Sprintf("%s update", fromManga.Name)

	// 	var body bytes.Buffer
	// 	if s.config.templateID != "" {
	// 		// For simplicity, let's assume templateID directly maps to a template name or content
	// 		// In a real application, you would load a template by ID
	// 		// For now, we'll use a basic HTML template
	// 		tmpl, err := template.New("email").Parse(`
	// 			<html>
	// 			<body>
	// 				<h1>{{.MangaName}} Update!</h1>
	// 				<p>Chapter {{.ChapterNumber}} is now available.</p>
	// 				<p>Read it here: <a href="{{.MangaReadURL}}">{{.MangaReadURL}}</a></p>
	// 			</body>
	// 			</html>
	// 		`)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to parse email template: %w", err)
	// 		}

	// 		data := struct {
	// 			MangaName     string
	// 			ChapterNumber string
	// 			MangaReadURL  string
	// 		}{
	// 			MangaName:     fromManga.Name,
	// 			ChapterNumber: chapter.Number,
	// 			MangaReadURL:  chapter.URI,
	// 		}

	// 		err = tmpl.Execute(&body, data)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to execute email template: %w", err)
	// 		}
	// 	} else {
	// 		body.WriteString(fmt.Sprintf("Subject: %s
	// ", subject))
	// 		body.WriteString(fmt.Sprintf("To: %s
	// ", to))
	// 		body.WriteString("MIME-version: 1.0;
	// Content-Type: text/html; charset="UTF-8";

	// ")
	// 		body.WriteString(fmt.Sprintf("<h1>%s Update!</h1><p>Chapter %s is now available.</p><p>Read it here: <a href="%s">%s</a></p>", fromManga.Name, chapter.Number, chapter.URI, chapter.URI))
	// 	}

	// 	msg := []byte(fmt.Sprintf("Subject: %s
	// "+
	// 		"To: %s
	// "+
	// 		"MIME-version: 1.0;
	// Content-Type: text/html; charset="UTF-8";

	// %s",
	// 		subject,
	// 		to,
	// 		body.String()))

	// 	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, s.config.fromEmail, to, msg)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to send email via SMTP2GO: %w", err)
	// 	}

	return nil
}
