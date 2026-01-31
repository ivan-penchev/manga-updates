package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/ivan-penchev/manga-updates/internal/library"
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

		lib := library.NewLibrary(store, providerRouter)
		err = lib.AddSeries(context.Background(), url)
		if err != nil {
			logger.Error("failed to add series", "error", err)
			os.Exit(1)
		}

		logger.Info("Successfully added series")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
