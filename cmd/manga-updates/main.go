package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/ivan-penchev/manga-updates/internal/notifier"
	"github.com/ivan-penchev/manga-updates/internal/provider"
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

	providerRouter, err := provider.NewProviderRouter(
		provider.NewMangaNelProviderFactory(cfg.MangaNelGraphQLEndpoint),
	)

	if err != nil {
		logger.Error("failed to create provider router", "error", err)
		os.Exit(1)
	}

	for path, manga := range persistedMangaSeries {
		logger.Info("Looking at", "mangaName", manga.Name, "dataPath", path)
		provider, err := providerRouter.GetProvider(manga)
		if err != nil {
			logger.Error("failed to get provider for manga", "manga", manga, "error", err)
			continue
		}

		IsNewerVersionAvailable, err := provider.IsNewerVersionAvailable(manga)
		if err != nil {
			logger.Error("failed to check for newer version", "manga", manga, "error", err)
			continue
		}

		if IsNewerVersionAvailable {
			mangaResponse, err := provider.GetLatestVersionMangaEntity(manga)
			if err != nil {
				logger.Error("failed to get latest version", "manga", manga, "error", err)
				continue
			}

			err = store.PersistManagaTitle(path, *mangaResponse)
			if err != nil {
				logger.Error("failed to persist manga", "manga", manga, "error", err)
				continue
			}

			if manga.ShouldNotify {
				chaptersMissing := manga.GetMissingChapters(*mangaResponse)
				logger.Info("Manga has new chapters", "mangaName", manga.Name, "numberOfNewChapters", len(chaptersMissing))
				if len(chaptersMissing) > 0 {

					// If we have multiple simultatnions updates they will be ordered descending
					// meaning the newest one will be first, and the olders updates will be last.
					// Take the oldest one by taking the last index.
					indexToTake := len(chaptersMissing) - 1
					err := notifier.NotifyForNewChapter(chaptersMissing[indexToTake], manga)
					if err != nil {
						logger.Error("failed to notify for manga", "manga", manga, "error", err)
					}
				}
			}
		}
	}
	logger.Info("Completed manga-updates main", "durationInSeconds", time.Since(ts).Seconds())
}
