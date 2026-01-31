package integrationtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMangaDexProvider(t *testing.T) {
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

		assert.Equal(t, "633d470a-4146-4dd3-b841-93dd648c23a5", manga.Slug)
		assert.Equal(t, domain.MangaSourceMangaDex, manga.Source)

		assert.Contains(t, []string{"Academy's Genius Swordmaster", "The Academy's Genius Swordmaster", "Academy ui Cheonjae Kaljabi"}, manga.Name)
		assert.True(t, manga.ShouldNotify)
		assert.NotEmpty(t, manga.Chapters, "Manga should have chapters")

		firstChapter := manga.Chapters[0]
		assert.NotEmpty(t, firstChapter.Slug)
		assert.Contains(t, firstChapter.URI, "mangadex.org/chapter/")
	})

	t.Run("IsNewerVersionAvailable", func(t *testing.T) {
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
		url := "https://mangadex.org/title/633d470a-4146-4dd3-b841-93dd648c23a5/academy-ui-cheonjae-kaljabi"
		manga, err := prov.GetMangaFromURL(ctx, url)
		require.NoError(t, err)

		isNewer, err := prov.IsNewerVersionAvailable(ctx, manga)
		require.NoError(t, err)
		assert.False(t, isNewer, "Should not be newer immediately after fetch")
	})

	t.Run("Search_Simple", func(t *testing.T) {
		query := "Solo Leveling"
		results, total, err := prov.Search(ctx, query, 0)
		require.NoError(t, err)
		assert.Greater(t, total, 0)
		assert.NotEmpty(t, results)

		found := false
		for _, r := range results {
			if strings.Contains(r.Manga.Name, "Solo Leveling") {
				found = true
				break
			}
		}

		assert.True(t, found, "Should have found 'Solo Leveling'")
		assert.True(t, len(results) > 0)

		first := results[0]
		assert.NotEmpty(t, first.Manga.Name)
		assert.NotEmpty(t, first.URL)
	})

	t.Run("Search_Pagination", func(t *testing.T) {
		query := "isekai"
		results1, total, err := prov.Search(ctx, query, 0)
		require.NoError(t, err)
		assert.Greater(t, total, 20)

		offset := len(results1)
		results2, _, err := prov.Search(ctx, query, offset)
		require.NoError(t, err)

		assert.NotEmpty(t, results2)
		assert.NotEqual(t, results1[0].Manga.Slug, results2[0].Manga.Slug)
	})
}
