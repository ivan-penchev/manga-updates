package app

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/config"
	"github.com/stretchr/testify/require"
)

func TestNewNotifierFromConfig(t *testing.T) {
	t.Parallel()

	n, err := NewNotifierFromConfig(config.NotifierConfig{})
	require.NoError(t, err)
	require.NotNil(t, n)
}

func TestRunUpdateCheck_EmptyFolder(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SERIES_DATA_FOLDER", tmpDir)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := RunUpdateCheck(context.Background(), "", logger)
	require.NoError(t, err)
}
