package integrationtest

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/mocks"
	updatechecker "github.com/ivan-penchev/manga-updates/internal/update-checker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUpdateCheckerServiceWithMocks(t *testing.T, mangaPathsWithMans map[string]domain.MangaEntity, shouldNotify bool) (*mocks.MockStore, *mocks.MockNotifier, *updatechecker.UpdateCheckerService) {
	mockStore := mocks.NewMockStore(t)
	mockNotifier := mocks.NewMockNotifier(t)
	mockStore.On("GetMangaSeries", mock.Anything).Return(mangaPathsWithMans)
	mockStore.On("PersistManagaTitle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	if shouldNotify {
		mockNotifier.On("NotifyForNewChapter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	updateChecker, err := updatechecker.NewUpdateCheckerService(mockNotifier, mockStore, providerRouter, logger)
	if err != nil {
		t.Fatal(err)
	}

	return mockStore, mockNotifier, updateChecker
}

func TestMangaNel(t *testing.T) {
	t.Parallel()
	t.Run("FirstTimeSyncManga", func(t *testing.T) {
		mangaPath := "manga1"
		newMangaWeKnowIsPresentAtSource := domain.MangaEntity{
			Name:         "Solo Leveling",
			Slug:         "solo-leveling_102",
			ShouldNotify: true,
			Source:       domain.MangaSourceMangaNel,
			Chapters:     []domain.ChapterEntity{},
		}
		pathsWithMangas := map[string]domain.MangaEntity{
			mangaPath: newMangaWeKnowIsPresentAtSource,
		}
		mockStore, mockNotifier, updateChecker := setupUpdateCheckerServiceWithMocks(t,
			pathsWithMangas,
			newMangaWeKnowIsPresentAtSource.ShouldNotify,
		)

		err := updateChecker.CheckForUpdates(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		// Check if everything specified with On and Return
		// was in fact called as expected. If not, the test will fail.
		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("FirstTimeSyncMangaShouldNotify", func(t *testing.T) {
		mangaPath := "manga1"
		newMangaWeKnowIsPresentAtSource := domain.MangaEntity{
			Name:         "Solo Leveling",
			Slug:         "solo-leveling_102",
			ShouldNotify: true,
			Source:       domain.MangaSourceMangaNel,
			Chapters:     []domain.ChapterEntity{},
		}
		pathsWithMangas := map[string]domain.MangaEntity{
			mangaPath: newMangaWeKnowIsPresentAtSource,
		}

		mockStore, mockNotifier, updateChecker := setupUpdateCheckerServiceWithMocks(t,
			pathsWithMangas,
			newMangaWeKnowIsPresentAtSource.ShouldNotify,
		)

		err := updateChecker.CheckForUpdates(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		// Check if everything specified with On and Return
		// was in fact called as expected. If not, the test will fail.
		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})
	t.Run("FirstTimeSyncMangaVerifyMangaContentAfterUpdate", func(t *testing.T) {
		mangaPath := "manga1"
		newMangaWeKnowIsPresentAtSource := domain.MangaEntity{
			Name:         "Solo Leveling",
			Slug:         "solo-leveling_102",
			ShouldNotify: true,
			Source:       domain.MangaSourceMangaNel,
			Chapters:     []domain.ChapterEntity{},
		}
		pathsWithMangas := map[string]domain.MangaEntity{
			mangaPath: newMangaWeKnowIsPresentAtSource,
		}

		mockStore, mockNotifier, updateChecker := setupUpdateCheckerServiceWithMocks(t,
			pathsWithMangas,
			newMangaWeKnowIsPresentAtSource.ShouldNotify,
		)

		err := updateChecker.CheckForUpdates(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)

		foundMockStoreInvocation := false
		for _, call := range mockStore.Calls {
			if call.Method == "PersistManagaTitle" {
				foundMockStoreInvocation = true
				savedMangaEntity, ok := call.Maybe().Arguments.Get(2).(domain.MangaEntity)
				assert.True(t, ok)
				assert.NotEqual(t, savedMangaEntity, newMangaWeKnowIsPresentAtSource)
				assert.Greater(t, len(savedMangaEntity.Chapters), len(newMangaWeKnowIsPresentAtSource.Chapters))
				assert.Equal(t, savedMangaEntity.Name, newMangaWeKnowIsPresentAtSource.Name)
				assert.Equal(t, savedMangaEntity.Slug, newMangaWeKnowIsPresentAtSource.Slug)
				assert.Equal(t, savedMangaEntity.Source, newMangaWeKnowIsPresentAtSource.Source)
				assert.Equal(t, savedMangaEntity.ShouldNotify, newMangaWeKnowIsPresentAtSource.ShouldNotify)
				assert.NotEqual(t, savedMangaEntity.Status, newMangaWeKnowIsPresentAtSource.Status)
			}
		}

		assert.True(t, foundMockStoreInvocation)
	})

	t.Run("MangaUpdateWithMultipleChaptersNotifyOnFirstOnly", func(t *testing.T) {
		mangaPath := "manga1"
		chapterNumber := float64(0)
		chapterDate := time.Date(2018, time.November, 21, 8, 44, 11, 0, time.UTC)
		newMangaWeKnowIsPresentAtSource := domain.MangaEntity{
			Name:         "Solo Leveling",
			Slug:         "solo-leveling_102",
			ShouldNotify: true,
			Source:       domain.MangaSourceMangaNel,
			Chapters: []domain.ChapterEntity{
				{
					Number: &chapterNumber,
					Date:   &chapterDate,
					URI:    "https://manganel.me/chapter/solo-leveling_102/chapter-0",
				},
			},
		}
		pathsWithMangas := map[string]domain.MangaEntity{
			mangaPath: newMangaWeKnowIsPresentAtSource,
		}

		mockStore, mockNotifier, updateChecker := setupUpdateCheckerServiceWithMocks(t,
			pathsWithMangas,
			newMangaWeKnowIsPresentAtSource.ShouldNotify,
		)

		err := updateChecker.CheckForUpdates(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)

		foundMockStoreInvocation := false
		for _, call := range mockNotifier.Calls {
			if call.Method == "NotifyForNewChapter" {
				foundMockStoreInvocation = true
				notifiedChapter, ok := call.Maybe().Arguments.Get(1).(domain.ChapterEntity)
				assert.True(t, ok)
				assert.Equal(t, *notifiedChapter.Number, float64(1))
			}
		}

		assert.True(t, foundMockStoreInvocation)
	})
}

func TestMangaNelProvider_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get Manganel provider from the router
	prov, err := providerRouter.GetProvider(domain.MangaEntity{
		Source: domain.MangaSourceMangaNel,
	})
	if err != nil {
		t.Fatalf("failed to get manganel provider: %v", err)
	}

	ctx := context.Background()

	t.Run("Search_Simple", func(t *testing.T) {
		query := "Solo Leveling"
		results, total, err := prov.Search(ctx, query, 0)
		assert.NoError(t, err)
		assert.Greater(t, total, 0)

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
		assert.NotEmpty(t, first.Manga.Slug)
		assert.NotEmpty(t, first.URL)
		assert.NotEmpty(t, first.ImageURL)
	})

	t.Run("Search_Pagination", func(t *testing.T) {
		query := "Isekai" // Broad term to ensure many results

		// Page 1
		results1, total1, err := prov.Search(ctx, query, 0)
		assert.NoError(t, err)
		assert.Greater(t, total1, 20) // Assume more than 20 results, as api defaults to 30

		// Page 2 - skip logic depends on API page size, usually 20 or 30.
		offset := len(results1)

		results2, _, err := prov.Search(ctx, query, offset)
		assert.NoError(t, err)
		assert.NotEmpty(t, results2)

		// Ensure different results
		assert.NotEqual(t, results1[0].Manga.Slug, results2[0].Manga.Slug)
	})
}
