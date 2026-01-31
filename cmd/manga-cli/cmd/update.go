package cmd

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/ivan-penchev/manga-updates/internal/notifier"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/ivan-penchev/manga-updates/internal/store"
	updatechecker "github.com/ivan-penchev/manga-updates/internal/update-checker"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates",
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.Default()

		ts := time.Now()
		ctx := context.Background()

		cfg, err := config.Load(cfgFile)
		if err != nil {
			logger.Error("failed to parse configuration", "error", err)
			os.Exit(1)
		}
		store := store.NewStore(cfg.SeriesDataFolder)
		persistedMangaSeries := store.GetMangaSeries(ctx)

		if len(persistedMangaSeries) == 0 {
			logger.Info("No series to monitor")
			return
		}

		notifierOptions := []notifier.NotifierOption{
			notifier.WithRecipients(cfg.Notifier.RecipientEmail),
			notifier.WithSenderEmail(cfg.Notifier.SenderEmail),
		}

		if cfg.Notifier.SendGrid.APIKey != "" {
			notifierOptions = append(notifierOptions, notifier.WithTemplateID(cfg.Notifier.SendGrid.TemplateID))
			notifierOptions = append(notifierOptions, notifier.WithSendGridAPIKey(cfg.Notifier.SendGrid.APIKey))
		} else if cfg.Notifier.SMTP2GO.APIKey != "" {
			notifierOptions = append(notifierOptions, notifier.WithTemplateID(cfg.Notifier.SMTP2GO.TemplateID))
			notifierOptions = append(notifierOptions, notifier.WithSMTP2GOAPIKey(cfg.Notifier.SMTP2GO.APIKey))
		}

		notif, err := notifier.NewNotifier(notifierOptions...)
		if err != nil {
			logger.Error("failed to create notifier", "error", err)
			os.Exit(1)
		}

		providerRouter, err := provider.NewProviderRouter(
			provider.NewMangaNelProviderFactory(provider.MangaNelProviderConfig{
				GraphQLEndpoint: cfg.MangaNelGraphQLEndpoint,
				RemoteChromeURL: cfg.RemoteChromeURL,
			}),
			provider.NewMangaDexProviderFactory(),
		)

		if err != nil {
			logger.Error("failed to create provider router", "error", err)
			os.Exit(1)
		}

		updatecheckerService, err := updatechecker.NewUpdateCheckerService(notif, store, providerRouter, logger)

		if err != nil {
			logger.Error("failed to create update checker service", "error", err)
			os.Exit(1)
		}
		err = updatecheckerService.CheckForUpdates(ctx)
		if err != nil {
			logger.Error("failed to check for updates", "error", err)
			os.Exit(1)
		}
		logger.Info("Completed manga-updates update", "durationInSeconds", time.Since(ts).Seconds())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
