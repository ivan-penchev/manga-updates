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
	Long: `Add a new manga series to the tracking list by providing its URL.
It automatically detects the provider (MangaDex or Manganelo), fetches the manga details,
and saves it to the local store for tracking updates.`,
	Example: `  manga-cli add https://mangadex.org/title/0328cd58-d519-45b6-abd2-049cfe63790b/kanmuri-san-no-tokei-koubou
  manga-cli add https://manganel.me/manga/god-of-martial-arts`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.Default()
		url := args[0]

		cfg, err := config.Load(cfgFile)
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

		p, err := providerRouter.GetProviderForURL(url)
		if err != nil {
			logger.Error("failed to find provider for url", "url", url, "error", err)
			os.Exit(1)
		}
		manga, err := p.GetMangaFromURL(ctx, url)
		if err != nil {
			logger.Error("failed to fetch manga details", "url", url, "error", err)
			os.Exit(1)
		}

		manga.ShouldNotify = true
		manga.Source = p.Kind()

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
