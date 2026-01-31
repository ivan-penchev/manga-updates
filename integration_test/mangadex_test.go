package integrationtest

import (
	"context"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMangaDexProvider(t *testing.T) {
	// Skip integration tests if short mode is enabled
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	factory := provider.NewMangaDexProviderFactory()

	prov, err := factory()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("GetMangaFromURL", func(t *testing.T) {
		// Example URL from user request
		url := "https://mangadex.org/title/633d470a-4146-4dd3-b841-93dd648c23a5/academy-ui-cheonjae-kaljabi"

		manga, err := prov.GetMangaFromURL(ctx, url)
		require.NoError(t, err)

		// Assertions based on expected real-world data (subject to change if manga updates)
		assert.Equal(t, "633d470a-4146-4dd3-b841-93dd648c23a5", manga.Slug)
		assert.Equal(t, domain.MangaSourceMangaDex, manga.Source)
		// The title might vary slightly depending on API response localization
		// Updated to include romanized title which API might return as default
		assert.Contains(t, []string{"Academy's Genius Swordmaster", "The Academy's Genius Swordmaster", "Academy ui Cheonjae Kaljabi"}, manga.Name)
		assert.True(t, manga.ShouldNotify)
		assert.NotEmpty(t, manga.Chapters, "Manga should have chapters")

		// Verify chapter structure
		firstChapter := manga.Chapters[0]
		assert.NotEmpty(t, firstChapter.Slug)
		assert.Contains(t, firstChapter.URI, "mangadex.org/chapter/")
	})

	t.Run("IsNewerVersionAvailable", func(t *testing.T) {
		// Use a known existing series, but give it an empty chapter list so it forces an update check
		manga := domain.MangaEntity{
			Slug:     "633d470a-4146-4dd3-b841-93dd648c23a5",
			Name:     "Academy's Genius Swordmaster",
			Source:   domain.MangaSourceMangaDex,
			Chapters: []domain.ChapterEntity{},
		}

		isNewer, err := prov.IsNewerVersionAvailable(ctx, manga)
		require.NoError(t, err)
		assert.True(t, isNewer, "Should be newer since local chapter list is empty")
	})

	t.Run("IsNewerVersionAvailable_NoUpdateNeeded", func(t *testing.T) {
		// First fetch real data
		url := "https://mangadex.org/title/633d470a-4146-4dd3-b841-93dd648c23a5/academy-ui-cheonjae-kaljabi"
		manga, err := prov.GetMangaFromURL(ctx, url)
		require.NoError(t, err)

		// Then check if update is needed (should be false as we just fetched it)
		// This assumes no chapter was released in the milliseconds between calls
		isNewer, err := prov.IsNewerVersionAvailable(ctx, manga)
		require.NoError(t, err)
		assert.False(t, isNewer, "Should not be newer immediately after fetch")
	})
}
