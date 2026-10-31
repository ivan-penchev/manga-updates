package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/notifier"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/ivan-penchev/manga-updates/internal/store"
	updatechecker "github.com/ivan-penchev/manga-updates/internal/update-checker"
)

// NewNotifierFromConfig constructs a Notifier from the application configuration.
func NewNotifierFromConfig(cfg config.NotifierConfig) (notifier.Notifier, error) {
	opts := []notifier.NotifierOption{
		notifier.WithRecipients(cfg.RecipientEmail),
		notifier.WithSenderEmail(cfg.SenderEmail),
	}

	if cfg.SendGrid.APIKey != "" {
		opts = append(opts,
			notifier.WithTemplateID(cfg.SendGrid.TemplateID),
			notifier.WithSendGridAPIKey(cfg.SendGrid.APIKey),
		)
	} else if cfg.SMTP2GO.APIKey != "" {
		opts = append(opts,
			notifier.WithTemplateID(cfg.SMTP2GO.TemplateID),
			notifier.WithSMTP2GOAPIKey(cfg.SMTP2GO.APIKey),
		)
	}

	return notifier.NewNotifier(opts...)
}

// NewProviderRouterFromConfig constructs the default ProviderRouter with MangaNel and MangaDex providers.
func NewProviderRouterFromConfig(cfg *config.Config) (domain.ProviderRouter, error) {
	return provider.NewProviderRouter(
		provider.NewMangaNelProviderFactory(provider.MangaNelProviderConfig{
			GraphQLEndpoint: cfg.MangaNelGraphQLEndpoint,
			RemoteChromeURL: cfg.RemoteChromeURL,
		}),
		provider.NewMangaDexProviderFactory(),
	)
}

// RunUpdateCheck loads the configuration, initializes dependencies, and checks all tracked series for updates.
func RunUpdateCheck(ctx context.Context, cfgPath string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	st := store.NewStore(cfg.SeriesDataFolder)
	persistedMangaSeries := st.GetMangaSeries(ctx)
	if len(persistedMangaSeries) == 0 {
		logger.Info("No series to monitor")
		return nil
	}

	notif, err := NewNotifierFromConfig(cfg.Notifier)
	if err != nil {
		return fmt.Errorf("creating notifier: %w", err)
	}

	providerRouter, err := NewProviderRouterFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating provider router: %w", err)
	}

	svc, err := updatechecker.NewUpdateCheckerService(notif, st, providerRouter, logger)
	if err != nil {
		return fmt.Errorf("creating update checker service: %w", err)
	}

	if err := svc.CheckForUpdates(ctx); err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	return nil
}
