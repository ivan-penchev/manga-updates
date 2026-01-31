package notifier

import (
	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type standardOutNotifier struct{}

func (s standardOutNotifier) NotifyForNewChapter(chapter domain.ChapterEntity, fromManga domain.MangaEntity) error {
	slog.Info("Notifying about new chapter",
		"mangaName", fromManga.Name,
		"chapterNumber", chapter.Number,
		"readUrl", chapter.URI,
	)
	return nil
}
