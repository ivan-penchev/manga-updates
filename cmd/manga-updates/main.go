package main

import (
	"fmt"

	"github.com/caarlos0/env"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
	"github.com/ivan-penchev/manga-updates/internal/store"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

type config struct {
	MangaNelGraphQLEndpoint string `env:"API_ENDPOINT" envDefault:"https://api.mghubcdn.com/graphql"`
	SeriesDataFolder        string `env:"SERIES_DATAFOLDER" envDefault:"%{HOME}/data" envExpand:"true"`
	SendGridAPIKey          string `env:"SENDGRID_API_KEY" envDefault:"api-key-here"`
}

// get all persisted manga series
// check if newly added title
//// add new content
//// do not notify
//// persist
// check with short if new updates
//// multiple updates?
////// request longest query
////// persist
//// single update?
////// persist
//// should notify?

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	store := store.NewStore(cfg.SeriesDataFolder)

	persistedMangaSeries := store.GetMangaSeries()

	if len(persistedMangaSeries) == 0 {
		fmt.Println("No series to monitor")
		return
	}

	mangaNelClient := manganelapiclient.NewMangaNelAPIClient(log, cfg.MangaNelGraphQLEndpoint)
	for path, manga := range persistedMangaSeries {
		log.Infof("Looking at %s, data from %s", manga.Name, path)

		mangaResponse, err := mangaNelClient.GetMangaSeriesFull(manga.Slug)
		if err != nil {
			log.Error(err)
			continue
		}

		if manga.IsNew() {
			log.Infof("New manga title (%s) added for updates, it has %d chapters so far", mangaResponse.Name, len(mangaResponse.Chapters))
			err = store.PersistestManagaTitle(path, *mangaResponse)
			if err != nil {
				log.Error(err)
			}
			log.Infof("New manga title (%s) persisted information %s", mangaResponse.Name, path)
			continue
		}

		if manga.IsOlder(*mangaResponse) {
			chaptersMissing := manga.GetMissingChapters(*mangaResponse)
			log.Infof("Manga (%s) has %d new chapters", manga.Name, len(chaptersMissing))
			if len(chaptersMissing) > 0 {

				m := mail.NewV3Mail()
				// 			$email->setFrom("manga@penchev.com", "Manga Notify");
				//   $email->addTo("thefolenangel@gmail.com",
				// 	"",
				// 	[
				// 	"manga_read_url" =>$urlToFetch,
				// 	"manga_name" => $displayName,
				// 	"chapter" => $nextChapter,
				// 	"subject" => $displayName.' update'
				// 	],
				// 	0
				// );
				// $email->setTemplateId("d-b4267c4ab110461e8e6cff80ff4aa0ca");
				//   return $email;

				from := mail.NewEmail("Manga Notify", "manga@penchev.com")
				m.SetFrom(from)
				m.SetTemplateID("d-b4267c4ab110461e8e6cff80ff4aa0ca")

				p := mail.NewPersonalization()
				tos := []*mail.Email{
					&mail.Email{Address: "thefolenangel@gmail.com"},
				}
				p.AddTos(tos...)
				p.SetDynamicTemplateData("manga_read_url", chaptersMissing[0].ManganelURI)
				p.SetDynamicTemplateData("manga_name", manga.Name)
				p.SetDynamicTemplateData("chapter", chaptersMissing[0].Number)
				p.SetDynamicTemplateData("subject", fmt.Sprintf("%s update", manga.Name))
				m.AddPersonalizations(p)

				request := sendgrid.GetRequest(cfg.SendGridAPIKey, "/v3/mail/send", "https://api.sendgrid.com")
				request.Method = "POST"
				var Body = mail.GetRequestBody(m)
				request.Body = Body
				_, err := sendgrid.API(request)
				if err != nil {
					log.Println(err)
				}
				err = store.PersistestManagaTitle(path, *mangaResponse)
				if err != nil {
					log.Error(err)
					continue
				}
				log.Infof("Manga title (%s) persisted information %s", mangaResponse.Name, path)
			}
			continue
		}
		log.Infof("Manga (%s) has no new updates", manga.Name)
	}
}
