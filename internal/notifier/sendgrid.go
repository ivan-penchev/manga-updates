package notifier

import (
	"fmt"

	"github.com/ivan-penchev/manga-updates/pkg/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func newSendgridNotifier(config *notifierConfig) Notifier {
	return sendgridNotifier{
		config: config,
	}
}

type sendgridNotifier struct {
	config *notifierConfig
}

func (s sendgridNotifier) NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error {

	m := mail.NewV3Mail()

	from := mail.NewEmail("Manga Notify", s.config.fromEmail)
	m.SetFrom(from)

	p := mail.NewPersonalization()
	tos := []*mail.Email{}
	for _, v := range s.config.recipients {
		tos = append(tos, &mail.Email{Address: v})
	}
	p.AddTos(tos...)

	if s.config.templateID != "" {
		m.SetTemplateID(s.config.templateID)
		p.SetDynamicTemplateData("manga_read_url", chapter.ManganelURI)
		p.SetDynamicTemplateData("manga_name", fromManga.Name)
		p.SetDynamicTemplateData("chapter", chapter.Number)
		p.SetDynamicTemplateData("subject", fmt.Sprintf("%s update", fromManga.Name))
	} else {
		p.Subject = fmt.Sprintf("%s update", fromManga.Name)
		m.Content = []*mail.Content{mail.NewContent("test", "test")}
	}

	m.AddPersonalizations(p)

	request := sendgrid.GetRequest(s.config.apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(m)
	request.Body = Body
	emailResponse, err := sendgrid.API(request)

	if err != nil {
		return err
	}

	if emailResponse.StatusCode >= 300 {
		return fmt.Errorf("SendGrid returned non-success status code: %d, body: %s", emailResponse.StatusCode, emailResponse.Body)
	}

	return nil
}
