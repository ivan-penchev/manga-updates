package updatechecker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateCheckerService_CheckForUpdates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockNotifier := mocks.NewMockNotifier(t)
	mockStore := mocks.NewMockStore(t)
	mockRouter := mocks.NewMockProviderRouter(t)
	mockProvider := mocks.NewMockProvider(t)

	ch1, ch2, ch3 := 1.0, 2.0, 3.0
	oldManga := domain.MangaEntity{
		Name:         "Naruto",
		Slug:         "naruto",
		Source:       domain.MangaSourceMangaDex,
		ShouldNotify: true,
		LastUpdate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Chapters: []domain.ChapterEntity{
			{Number: &ch1, URI: "https://example.com/1"},
		},
	}

	newManga := domain.MangaEntity{
		Name:         "Naruto",
		Slug:         "naruto",
		Source:       domain.MangaSourceMangaDex,
		ShouldNotify: true,
		LastUpdate:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Chapters: []domain.ChapterEntity{
			{Number: &ch3, URI: "https://example.com/3"},
			{Number: &ch2, URI: "https://example.com/2"},
			{Number: &ch1, URI: "https://example.com/1"},
		},
	}

	mockStore.EXPECT().GetMangaSeries(ctx).Return(map[string]domain.MangaEntity{
		"/data/naruto/data.json": oldManga,
	})
	mockRouter.EXPECT().GetProvider(oldManga).Return(mockProvider, nil)
	mockProvider.EXPECT().IsNewerVersionAvailable(ctx, oldManga).Return(true, nil)
	mockProvider.EXPECT().GetLatestVersionMangaEntity(ctx, oldManga).Return(&newManga, nil)
	mockStore.EXPECT().PersistMangaTitle(ctx, "/data/naruto/data.json", newManga).Return(nil)
	// Expect notification with the oldest of the new chapters (chapter 2)
	mockNotifier.EXPECT().NotifyForNewChapter(ctx, mock.MatchedBy(func(ch domain.ChapterEntity) bool {
		return ch.Number != nil && *ch.Number == 2.0
	}), oldManga).Return(nil)

	svc, err := NewUpdateCheckerService(mockNotifier, mockStore, mockRouter, logger)
	require.NoError(t, err)
	assert.NoError(t, svc.CheckForUpdates(ctx))
}
