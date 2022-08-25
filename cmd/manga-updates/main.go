package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
	"github.com/ivan-penchev/manga-updates/internal/store"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

type config struct {
	MangaNelGraphQLEndpoint string `env:"API_ENDPOINT" envDefault:"https://api.mghubcdn.com/graphql"`
	SeriesDataFolder        string `env:"SERIES_DATAFOLDER" envDefault:"$HOME/repos/manga-updates/data" envExpand:"true"`
	SendGridAPIKey          string `env:"SENDGRID_API_KEY" envDefault:"api-key-here"`
}

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	ts := time.Now()

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}
	innerCtx, innerCancel := chromedp.NewContext(context.Background())
	defer innerCancel()
	// create a timeout
	ctx, cancel := context.WithTimeout(innerCtx, 45*time.Second)
	defer cancel()

	// navigate to a page, wait for an element, click
	var mhubApiAccessToken string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://manganel.me/`),
		// wait for footer element is visible (ie, page is loaded)
		chromedp.WaitVisible(`#app > div:nth-child(1) > header`),
		// find and click "Example" link
		// retrieve the text of the textarea
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetAllCookies().Do(ctx)
			if err != nil {
				return err
			}

			for _, cookie := range cookies {
				if cookie.Name == "mhub_access" {
					mhubApiAccessToken = cookie.Value
				}
			}

			if mhubApiAccessToken == "" {
				return errors.New("can't find api key, inside cookiesgit")
			}

			return nil
		}),
	)

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	store := store.NewStore(cfg.SeriesDataFolder)

	persistedMangaSeries := store.GetMangaSeries()

	if len(persistedMangaSeries) == 0 {
		fmt.Println("No series to monitor")
		return
	}

	mangaNelClient := manganelapiclient.NewMangaNelAPIClient(log, cfg.MangaNelGraphQLEndpoint, mhubApiAccessToken)
	for path, manga := range persistedMangaSeries {
		log.Infof("Looking at %s, data from %s", manga.Name, path)

		mangaResponse, err := mangaNelClient.GetMangaSeriesFull(manga.Slug)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}

		if manga.IsNew() {
			log.Infof("New manga title (%s) added for updates, it has %d chapters so far", mangaResponse.Name, len(mangaResponse.Chapters))
			err = store.PersistestManagaTitle(path, *mangaResponse)
			if err != nil {
				log.Error(err)
				os.Exit(1)

			}
			log.Infof("New manga title (%s) persisted information %s", mangaResponse.Name, path)
			continue
		}

		if manga.IsOlder(*mangaResponse) {
			chaptersMissing := manga.GetMissingChapters(*mangaResponse)
			log.Infof("Manga (%s) has %d new chapters", manga.Name, len(chaptersMissing))
			if len(chaptersMissing) > 0 {

				m := mail.NewV3Mail()

				from := mail.NewEmail("Manga Notify", "manga@penchev.com")
				m.SetFrom(from)
				m.SetTemplateID("d-b4267c4ab110461e8e6cff80ff4aa0ca")

				p := mail.NewPersonalization()
				tos := []*mail.Email{
					&mail.Email{Address: "thefolenangel@gmail.com"},
				}
				p.AddTos(tos...)
				// if we have multiple simultatnions updates, take the oldest one.
				indexToTake := len(chaptersMissing) - 1
				p.SetDynamicTemplateData("manga_read_url", chaptersMissing[indexToTake].ManganelURI)
				p.SetDynamicTemplateData("manga_name", manga.Name)
				p.SetDynamicTemplateData("chapter", chaptersMissing[indexToTake].Number)
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
					os.Exit(1)
				}
				log.Infof("Manga title (%s) persisted information %s", mangaResponse.Name, path)
			}
			continue
		}
		log.Infof("Manga (%s) has no new updates", manga.Name)
	}

	log.Infof("Completed manga-updates main (duration=%s)", time.Since(ts))
}
