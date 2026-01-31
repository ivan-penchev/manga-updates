package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/spf13/cobra"
)

var providersList []string
var offset int

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for a manga series",
	Long: `Search for a manga series across supported providers.
Currently supported providers: manganel, mangadex (not yet implemented).
Examples:
  manga-cli search "naruto"
  manga-cli search "one piece" --providers manganel,mangadex
  manga-cli search "test" --offset 30
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		logger := slog.Default()

		cfg, err := config.Load(cfgFile)
		if err != nil {
			logger.Error("failed to parse configuration", "error", err)
			os.Exit(1)
		}

		var factories []func() (domain.Provider, error)

		uniqueProviders := make(map[string]bool)
		for _, p := range providersList {
			uniqueProviders[p] = true
		}

		if uniqueProviders["manganel"] {
			factories = append(factories, provider.NewMangaNelProviderFactory(provider.MangaNelProviderConfig{
				GraphQLEndpoint: cfg.MangaNelGraphQLEndpoint,
				RemoteChromeURL: cfg.RemoteChromeURL,
			}))
		}
		if uniqueProviders["mangadex"] {
			factories = append(factories, provider.NewMangaDexProviderFactory())
		}

		if len(factories) == 0 {
			logger.Warn("No valid providers selected. Defaulting to manganel.")
			factories = append(factories, provider.NewMangaNelProviderFactory(provider.MangaNelProviderConfig{
				GraphQLEndpoint: cfg.MangaNelGraphQLEndpoint,
				RemoteChromeURL: cfg.RemoteChromeURL,
			}))
		}

		for _, factory := range factories {
			p, err := factory()
			if err != nil {
				logger.Error("failed to init provider", "error", err)
				continue
			}

			results, totalCount, err := p.Search(cmd.Context(), query, offset)
			if err != nil {
				logger.Warn("search failed for provider", "provider", p.Kind(), "error", err)
				continue
			}
			if len(results) == 0 {
				fmt.Printf("No results found for '%s' in %s (Offset: %d)\n", query, p.Kind(), offset)
				continue
			}

			fmt.Printf("Results from %s (Total: %d, Showing: %d, Offset: %d):\n", p.Kind(), totalCount, len(results), offset)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "TITLE\tLATEST CHAPTER\tRANK\tURL\tIMAGE")

			for _, r := range results {
				rankStr := "-"
				if r.Rank > 0 {
					rankStr = fmt.Sprintf("%d", r.Rank)
				}
				latestStr := "-"
				if r.LatestChapter != "" {
					latestStr = r.LatestChapter
				}

				// Truncate title if too long to keep table readable
				title := r.Manga.Name
				if len(title) > 50 {
					title = title[:47] + "..."
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", title, latestStr, rankStr, r.URL, r.ImageURL)
			}
			w.Flush()
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringSliceVarP(&providersList, "providers", "p", []string{"manganel"}, "List of providers to search (manganel, mangadex)")
	searchCmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
}
