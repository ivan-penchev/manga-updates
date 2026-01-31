package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/ivan-penchev/manga-updates/internal/store"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Add a new manga series",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.Default()
		url := args[0]

		cfg, err := config.Load()
		if err != nil {
			logger.Error("failed to parse configuration", "error", err)
			os.Exit(1)
		}

		store := store.NewStore(cfg.SeriesDataFolder)

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

		ctx := context.Background()

		// 1. Find correct provider
		p, err := providerRouter.GetProviderForURL(url)
		if err != nil {
			logger.Error("failed to find provider for url", "url", url, "error", err)
			os.Exit(1)
		}

		// 2. Fetch series details
		manga, err := p.GetMangaFromURL(ctx, url)
		if err != nil {
			logger.Error("failed to fetch manga details", "url", url, "error", err)
			os.Exit(1)
		}

		// 3. Set defaults
		manga.ShouldNotify = true
		// Source is likely already set by GetMangaFromURL but ensure it matches provider kind
		manga.Source = p.Kind()

		// 4. Save to store
		err = store.AddManga(ctx, manga)
		if err != nil {
			logger.Error("failed to save series to store", "manga", manga.Name, "error", err)
			os.Exit(1)
		}

		logger.Info("Successfully added series", "title", manga.Name)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
