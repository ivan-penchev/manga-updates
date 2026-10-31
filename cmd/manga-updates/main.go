package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/app"
)

func main() {
	configFlag := flag.String("config", "", "config file path")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ts := time.Now()
	ctx := context.Background()

	if err := app.RunUpdateCheck(ctx, *configFlag, logger); err != nil {
		logger.Error("update check failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Completed manga-updates main", "durationInSeconds", time.Since(ts).Seconds())
}
