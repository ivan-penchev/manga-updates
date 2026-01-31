package integrationtest

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/mocks"
	updatechecker "github.com/ivan-penchev/manga-updates/internal/update-checker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUpdateCheckerServiceWithMocks(t *testing.T, mangaPathsWithMans map[string]domain.MangaEntity, shouldNotify bool) (*mocks.StoreMock, *mocks.NotifierMock, *updatechecker.UpdateCheckerService) {
	mockStore := mocks.NewStoreMock(t)
	mockNotifier := mocks.NewNotifierMock(t)
	mockStore.On("GetMangaSeries").Return(mangaPathsWithMans)
	mockStore.On("PersistManagaTitle", mock.Anything, mock.Anything).Return(nil)
	if shouldNotify {
		mockNotifier.On("NotifyForNewChapter", mock.Anything, mock.Anything).Return(nil)
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

		err := updateChecker.CheckForUpdates()
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

		err := updateChecker.CheckForUpdates()
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

		err := updateChecker.CheckForUpdates()
		if err != nil {
			t.Fatal(err)
		}

		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)

		foundMockStoreInvocation := false
		for _, call := range mockStore.Calls {
			if call.Method == "PersistManagaTitle" {
				foundMockStoreInvocation = true
				savedMangaEntity, ok := call.Maybe().Arguments.Get(1).(domain.MangaEntity)
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

		err := updateChecker.CheckForUpdates()
		if err != nil {
			t.Fatal(err)
		}

		mockStore.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)

		foundMockStoreInvocation := false
		for _, call := range mockNotifier.Calls {
			if call.Method == "NotifyForNewChapter" {
				foundMockStoreInvocation = true
				notifiedChapter, ok := call.Maybe().Arguments.Get(0).(domain.ChapterEntity)
				assert.True(t, ok)
				assert.Equal(t, *notifiedChapter.Number, float64(1))
			}
		}

		assert.True(t, foundMockStoreInvocation)
	})
}
