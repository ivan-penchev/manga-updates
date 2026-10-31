package domain

import (
	"context"
	"time"
)

type MangaSource string
type MangaStatus string

const (
	MangaSourceMangaNel MangaSource = "manganel"
	MangaSourceMangaDex MangaSource = "mangadex"

	MangaStatusOngoing  MangaStatus = "ongoing"
	MangaStatusComplete MangaStatus = "complete"
)

type MangaEntity struct {
	Name         string          `json:"name"`
	ShouldNotify bool            `json:"shouldNotify"`
	LastUpdate   time.Time       `json:"lastUpdate"`
	Slug         string          `json:"slug"`
	Status       MangaStatus     `json:"status"`
	Source       MangaSource     `json:"source"`
	Chapters     []ChapterEntity `json:"chapters"`
}

type ChapterEntity struct {
	Number *float64   `json:"name"`
	Slug   *string    `json:"slug"`
	Date   *time.Time `json:"date"`
	URI    string     `json:"uri"`
}

// IsNew reports whether the manga entity has never been synced.
func (m *MangaEntity) IsNew() bool {
	return len(m.Chapters) == 0 && m.LastUpdate.IsZero()
}

// IsOlder reports whether m was updated before n.
func (m *MangaEntity) IsOlder(n MangaEntity) bool {
	return m.LastUpdate.Before(n.LastUpdate)
}

type Provider interface {
	Kind() MangaSource
	GetLatestVersionMangaEntity(ctx context.Context, manga MangaEntity) (*MangaEntity, error)
	GetMangaFromURL(ctx context.Context, url string) (MangaEntity, error)
	IsNewerVersionAvailable(ctx context.Context, manga MangaEntity) (bool, error)
	Supports(url string) bool
	Search(ctx context.Context, query string, offset int) ([]SearchResult, int, error)
}

type SearchResult struct {
	Manga         MangaEntity
	Rank          int
	ImageURL      string
	URL           string
	LatestChapter string
}

type ProviderRouter interface {
	GetProvider(manga MangaEntity) (Provider, error)
	GetProviderForURL(url string) (Provider, error)
}

// GetMissingChapters returns the chapters present at the head of n that are not in m.
func (m *MangaEntity) GetMissingChapters(n MangaEntity) []ChapterEntity {
	if len(n.Chapters) == 0 {
		return []ChapterEntity{}
	}

	if hasChapterIdentity(m.Chapters) || hasChapterIdentity(n.Chapters) {
		missing := make([]ChapterEntity, 0)
		for _, candidate := range n.Chapters {
			if containsChapter(m.Chapters, candidate) {
				break
			}
			missing = append(missing, candidate)
		}
		return missing
	}

	lengthCurrent := len(m.Chapters)
	lengthNewer := len(n.Chapters)
	if lengthNewer <= lengthCurrent {
		return []ChapterEntity{}
	}

	diff := lengthNewer - lengthCurrent
	out := make([]ChapterEntity, diff)
	copy(out, n.Chapters[:diff])
	return out
}

func hasChapterIdentity(chapters []ChapterEntity) bool {
	for _, c := range chapters {
		if c.Number != nil || (c.Slug != nil && *c.Slug != "") || c.URI != "" {
			return true
		}
	}
	return false
}

func containsChapter(chapters []ChapterEntity, target ChapterEntity) bool {
	for _, c := range chapters {
		if sameChapter(c, target) {
			return true
		}
	}
	return false
}

func sameChapter(a, b ChapterEntity) bool {
	if a.Slug != nil && b.Slug != nil && *a.Slug != "" && *b.Slug != "" {
		return *a.Slug == *b.Slug
	}
	if a.Number != nil && b.Number != nil {
		return *a.Number == *b.Number
	}
	if a.URI != "" && b.URI != "" {
		return a.URI == b.URI
	}
	return false
}
