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
	updatechecker "github.com/ivan-penchev/manga-updates/internal/update-checker"
)

type config struct {
	MangaNelGraphQLEndpoint    string `env:"API_ENDPOINT" envDefault:"https://api.mghcdn.com/graphql"`
	SeriesDataFolder           string `env:"SERIES_DATAFOLDER" envDefault:"$HOME/repos/manga-updates/data" envExpand:"true"`
	SendGridAPIKey             string `env:"SENDGRID_API_KEY"`
	SendGridTemplateId         string `env:"SENDGRID_TEMPLATE_ID"`
	SMTP2GOApiKey              string `env:"SMTP2GO_API_KEY"`
	SMTP2GOTemplateId          string `env:"SMTP2GO_TEMPLATE_ID"`
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

	notifierOptions := []notifier.NotifierOption{
		notifier.WithRecipients(cfg.NotificationRecipientEmail),
		notifier.WithSenderEmail(cfg.NotificationSenderEmail),
	}

	if cfg.SendGridAPIKey != "" {
		notifierOptions = append(notifierOptions, notifier.WithTemplateID(cfg.SendGridTemplateId))
		notifierOptions = append(notifierOptions, notifier.WithSendGridAPIKey(cfg.SendGridAPIKey))
	} else if cfg.SMTP2GOApiKey != "" {
		notifierOptions = append(notifierOptions, notifier.WithTemplateID(cfg.SMTP2GOTemplateId))
		notifierOptions = append(notifierOptions, notifier.WithSMTP2GOAPIKey(cfg.SMTP2GOApiKey))
	}

	notifier, err := notifier.NewNotifier(notifierOptions...)
	if err != nil {
		logger.Error("failed to create notifier", "error", err)
		os.Exit(1)
	}

	providerRouter, err := provider.NewProviderRouter(
		provider.NewMangaNelProviderFactory(cfg.MangaNelGraphQLEndpoint),
		provider.NewMangaDexProviderFactory(),
	)

	if err != nil {
		logger.Error("failed to create provider router", "error", err)
		os.Exit(1)
	}

	updatecheckerService, err := updatechecker.NewUpdateCheckerService(notifier, store, providerRouter, logger)

	if err != nil {
		logger.Error("failed to create update checker service", "error", err)
		os.Exit(1)
	}
	err = updatecheckerService.CheckForUpdates()
	if err != nil {
		logger.Error("failed to check for updates", "error", err)
		os.Exit(1)
	}
	logger.Info("Completed manga-updates main", "durationInSeconds", time.Since(ts).Seconds())
}
