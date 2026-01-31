package updatechecker

import (
	"context"
	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type Notifier interface {
	NotifyForNewChapter(ctx context.Context, chapter domain.ChapterEntity, fromManga domain.MangaEntity) error
}

type Store interface {
	GetMangaSeries(ctx context.Context) map[string]domain.MangaEntity
	PersistMangaTitle(ctx context.Context, location string, mangaTitle domain.MangaEntity) error
}

type UpdateCheckerService struct {
	notifier  Notifier
	store     Store
	providers domain.ProviderRouter
	logger    *slog.Logger
}

func NewUpdateCheckerService(notifier Notifier, store Store, providers domain.ProviderRouter, logger *slog.Logger) (*UpdateCheckerService, error) {
	return &UpdateCheckerService{
		notifier:  notifier,
		store:     store,
		providers: providers,
		logger:    logger,
	}, nil
}

func (ucs *UpdateCheckerService) CheckForUpdates(ctx context.Context) error {
	persistedMangaSeries := ucs.store.GetMangaSeries(ctx)

	if len(persistedMangaSeries) == 0 {
		return nil
	}

	for path, manga := range persistedMangaSeries {
		ucs.logger.Info("Looking at", "mangaName", manga.Name, "dataPath", path)
		provider, err := ucs.providers.GetProvider(manga)

		if err != nil {
			return err
		}

		IsNewerVersionAvailable, err := provider.IsNewerVersionAvailable(ctx, manga)
		if err != nil {
			ucs.logger.Error("failed to check for newer version", "manga", manga.Name, "error", err)
			continue
		}

		if IsNewerVersionAvailable {
			mangaResponse, err := provider.GetLatestVersionMangaEntity(ctx, manga)

			if err != nil {
				ucs.logger.Error("failed to get latest version", "manga", manga, "error", err)
				continue
			}

			err = ucs.store.PersistMangaTitle(ctx, path, *mangaResponse)
			if err != nil {
				ucs.logger.Error("failed to persist manga", "manga", manga, "error", err)
				continue
			}

			if manga.ShouldNotify {
				chaptersMissing := manga.GetMissingChapters(*mangaResponse)
				ucs.logger.Info("Manga has new chapters", "mangaName", manga.Name, "numberOfNewChapters", len(chaptersMissing))
				if len(chaptersMissing) > 0 {

					// If we have multiple simultatnions updates they will be ordered descending
					// meaning the newest one will be first, and the olders updates will be last.
					// Take the oldest one by taking the last index.
					indexToTake := len(chaptersMissing) - 1
					err := ucs.notifier.NotifyForNewChapter(ctx, chaptersMissing[indexToTake], manga)
					if err != nil {
						slog.Error("failed to notify for manga", "manga", manga, "error", err)
					}
				}
			}
		}
	}

	return nil
}
