package cmd

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/app"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates",
	Run: func(cmd *cobra.Command, args []string) {
		logger := slog.Default()
		ts := time.Now()
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		if err := app.RunUpdateCheck(ctx, cfgFile, logger); err != nil {
			logger.Error("update check failed", "error", err)
			os.Exit(1)
		}

		logger.Info("Completed manga-updates update", "durationInSeconds", time.Since(ts).Seconds())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
