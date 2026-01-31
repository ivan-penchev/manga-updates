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

// examine if default values are present on a MangaEntity,
// if they are, it means the entity has never been synced before.
// and should be considered new for the purpose of syncing
//
// Assumes that the default values will always be present, when the entity has never been synced before.
func (m *MangaEntity) IsNew() bool {
	if len(m.Chapters) == 0 && m.LastUpdate.IsZero() {
		return true
	}
	return false
}

// Compares if the current MangaEntity is older
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

// returns the missing chapters between the current manga and the new one
func (m *MangaEntity) GetMissingChapters(n MangaEntity) []ChapterEntity {
	lenthCurrent := len(m.Chapters)
	lenthNewer := len(n.Chapters)
	endArray := 0
	if lenthNewer > lenthCurrent {
		endArray = lenthNewer - lenthCurrent
	}

	vals2 := n.Chapters[0:endArray]

	return vals2
}
