package updatechecker

import (
	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/notifier"
	"github.com/ivan-penchev/manga-updates/internal/provider"
	"github.com/ivan-penchev/manga-updates/internal/store"
)

type UpdateCheckerService struct {
	notifier  notifier.Notifier
	store     store.Store
	providers provider.ProviderRouter
}

func NewUpdateCheckerService(notifier notifier.Notifier, store store.Store, providers provider.ProviderRouter) (*UpdateCheckerService, error) {
	return &UpdateCheckerService{
		notifier:  notifier,
		store:     store,
		providers: providers,
	}, nil
}

func (ucs *UpdateCheckerService) CheckForUpdates() error {
	persistedMangaSeries := ucs.store.GetMangaSeries()

	if len(persistedMangaSeries) == 0 {
		return nil
	}

	for path, manga := range persistedMangaSeries {
		slog.Info("Looking at", "mangaName", manga.Name, "dataPath", path)
		provider, err := ucs.providers.GetProvider(manga)

		if err != nil {
			return err
		}

		IsNewerVersionAvailable, err := provider.IsNewerVersionAvailable(manga)
		if err != nil {
			slog.Error("failed to check for newer version", "manga", manga, "error", err)
			continue
		}

		if IsNewerVersionAvailable {
			mangaResponse, err := provider.GetLatestVersionMangaEntity(manga)

			if err != nil {
				slog.Error("failed to get latest version", "manga", manga, "error", err)
				continue
			}

			err = ucs.store.PersistManagaTitle(path, *mangaResponse)
			if err != nil {
				slog.Error("failed to persist manga", "manga", manga, "error", err)
				continue
			}

			if manga.ShouldNotify {
				chaptersMissing := manga.GetMissingChapters(*mangaResponse)
				slog.Info("Manga has new chapters", "mangaName", manga.Name, "numberOfNewChapters", len(chaptersMissing))
				if len(chaptersMissing) > 0 {

					// If we have multiple simultatnions updates they will be ordered descending
					// meaning the newest one will be first, and the olders updates will be last.
					// Take the oldest one by taking the last index.
					indexToTake := len(chaptersMissing) - 1
					err := ucs.notifier.NotifyForNewChapter(chaptersMissing[indexToTake], manga)
					if err != nil {
						slog.Error("failed to notify for manga", "manga", manga, "error", err)
					}
				}
			}
		}
	}

	return nil
}
