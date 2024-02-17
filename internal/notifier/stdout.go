package notifier

import (
	"log/slog"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type standardOutNotifier struct{}

func (s standardOutNotifier) NotifyForNewChapter(chapter types.ChapterEntity, fromManga types.MangaEntity) error {
	slog.Info("Notifying about new chapter",
		"mangaName", fromManga.Name,
		"chapterNumber", chapter.Number,
		"readUrl", chapter.ManganelURI,
	)
	return nil
}
