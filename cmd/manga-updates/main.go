package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
	"github.com/ivan-penchev/manga-updates/internal/notifier"
	"github.com/ivan-penchev/manga-updates/internal/store"
)

type config struct {
	MangaNelGraphQLEndpoint    string `env:"API_ENDPOINT" envDefault:"https://manganel.me/api/graphql"`
	SeriesDataFolder           string `env:"SERIES_DATAFOLDER" envDefault:"$HOME/repos/manga-updates/data" envExpand:"true"`
	SendGridAPIKey             string `env:"SENDGRID_API_KEY"`
	SendGridTemplateId         string `env:"SENDGRID_TEMPLATE_ID"`
	NotificationRecipientEmail string `env:"NOTIFICATION_EMAIL_RECIPIENT,required"`
	NotificationSenderEmail    string `env:"NOTIFICATION_EMAIL_SENDER,required"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ts := time.Now()

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		logger.Error("failed to parse configuration", "error", err)
	}
	store := store.NewStore(cfg.SeriesDataFolder)
	persistedMangaSeries := store.GetMangaSeries()

	if len(persistedMangaSeries) == 0 {
		fmt.Println("No series to monitor")
		return
	}

	innerCtx, innerCancel := chromedp.NewContext(context.Background())
	defer innerCancel()
	// create a timeout
	ctx, cancel := context.WithTimeout(innerCtx, 45*time.Second)
	defer cancel()

	// navigate to a page, wait for an element, click
	var mhubApiAccessToken string
	err := chromedp.Run(ctx,
		chromedp.Emulate(device.IPhone12),
		chromedp.Navigate(`https://manganel.me/manga/my-wife-is-a-demon-queen`),
		chromedp.Sleep(4*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			for _, cookie := range cookies {
				if cookie.Name == "mhub_access" {
					mhubApiAccessToken = cookie.Value
				}
			}

			return nil
		}),
	)

	if err != nil {
		logger.Error("failed to find manganel access cookie", "error", err)
		os.Exit(1)
	}

	notifier, err := notifier.NewNotifier(
		notifier.WithRecipients(cfg.NotificationRecipientEmail),
		notifier.WithSenderEmail(cfg.NotificationSenderEmail),
		notifier.WithTemplateID(cfg.SendGridTemplateId),
		notifier.WithSendGridAPIKey(cfg.SendGridAPIKey),
	)

	if err != nil {
		logger.Error("failed to create notifier", "error", err)
		os.Exit(1)
	}

	mangaNelClient := manganelapiclient.NewMangaNelAPIClient(logger, cfg.MangaNelGraphQLEndpoint, mhubApiAccessToken)
	for path, manga := range persistedMangaSeries {
		logger.Info("Looking at", "mangaName", manga.Name, "dataPath", path)

		mangaResponse, err := mangaNelClient.GetMangaSeriesFull(manga.Slug)
		if err != nil {
			logger.Error("failed to extract full series", err)
			os.Exit(1)
		}

		if manga.IsNew() {
			logMessage := fmt.Sprintf("New manga title (%s) added for updates, it has %d chapters so far", mangaResponse.Name, len(mangaResponse.Chapters))
			logger.Info(logMessage)
			err = store.PersistestManagaTitle(path, *mangaResponse)
			if err != nil {
				logger.Error("failed to persist manga", err)
				os.Exit(1)

			}
			logMessage = fmt.Sprintf("New manga title (%s) persisted information %s", mangaResponse.Name, path)
			logger.Info(logMessage)
			continue
		}

		if manga.IsOlder(*mangaResponse) {
			chaptersMissing := manga.GetMissingChapters(*mangaResponse)
			logger.Info("Manga has new chapters", "mangaName", manga.Name, "numberOfNewChapters", len(chaptersMissing))
			if len(chaptersMissing) > 0 {

				// If we have multiple simultatnions updates they will be ordered descending
				// meaning the newest one will be first, and the olders updates will be last.
				// Take the oldest one by taking the last index.
				indexToTake := len(chaptersMissing) - 1
				err := notifier.NotifyForNewChapter(chaptersMissing[indexToTake], manga)
				if err != nil {
					logger.Error("failed to send email", "error", err)
				}
				err = store.PersistestManagaTitle(path, *mangaResponse)
				if err != nil {
					logger.Error("failed to persist manga title", "error", err)
					os.Exit(1)
				}
				logger.Info("Manga persisted information", "mangaName", mangaResponse.Name, "dataPath", path)
			}
			continue
		}
		logger.Info("Manga has no new updates", "mangaName", manga.Name)
	}

	logger.Info("Completed manga-updates main", "durationInSeconds", time.Since(ts).Seconds())
}
