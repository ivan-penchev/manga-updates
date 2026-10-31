package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_RoundTrip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	st := NewStore(tmpDir)
	ctx := context.Background()

	manga := domain.MangaEntity{
		Name:         "One Piece",
		Slug:         "one-piece_116",
		Status:       domain.MangaStatusOngoing,
		Source:       domain.MangaSourceMangaNel,
		ShouldNotify: true,
	}

	require.NoError(t, st.AddManga(ctx, manga))
	// Adding duplicate should fail
	require.Error(t, st.AddManga(ctx, manga))

	series := st.GetMangaSeries(ctx)
	expectedPath := filepath.Join(tmpDir, "one-piece_116.json")
	require.Contains(t, series, expectedPath)
	assert.Equal(t, "One Piece", series[expectedPath].Name)
}
