package types

import (
	"time"
)

type MangaEntity struct {
	Name         string          `json:"name"`
	ShouldNotify bool            `json:"shouldNotify"`
	LastUpdate   time.Time       `json:"lastUpdate"`
	Slug         string          `json:"slug"`
	Status       string          `json:"status"`
	Chapters     []ChapterEntity `json:"chapters"`
}

type ChapterEntity struct {
	Number      *float64   `json:"name"`
	Slug        *string    `json:"slug"`
	Date        *time.Time `json:"date"`
	ManganelURI string     `json:"mangaNelUri"`
}

// looks if default values are present on a MangaEntity
// assumes if default values are present, that the entity has never been synced before
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

// Compares if the current MangaEntity is older
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
