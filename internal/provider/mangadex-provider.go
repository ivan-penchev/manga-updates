package provider

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	m "github.com/darylhjd/mangodex"
	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type mangaDexProvider struct {
	mangaDexClient  *m.DexClient
	cachedResponses map[string]*types.MangaEntity
	mutex           sync.RWMutex
}

func NewMangaDexProviderFactory() func() (Provider, error) {
	return func() (Provider, error) {

		c := m.NewDexClient()
		v := url.Values{}
		v.Add("title", "One Piece")

		// check if the client is working
		res, err := c.Manga.GetMangaList(v)

		if err != nil || res.Data == nil || len(res.Data) == 0 {
			return nil, errors.New("failed to create a new MangaDex client")
		}

		return &mangaDexProvider{
			mangaDexClient:  c,
			cachedResponses: make(map[string]*types.MangaEntity, 0),
			mutex:           sync.RWMutex{},
		}, nil
	}

}

// GetLatestVersionMangaEntity implements Provider.
func (mdp *mangaDexProvider) GetLatestVersionMangaEntity(manga types.MangaEntity) (*types.MangaEntity, error) {
	v := url.Values{}
	v.Add("translatedLanguage[]", "en") // hardcode it for now
	res, err := mdp.mangaDexClient.Chapter.GetMangaChapters(manga.Slug, v)

	if err != nil || res.Data == nil || len(res.Data) == 0 {
		return nil, errors.Join(err, errors.New("failed to get list of chapters for manga"))
	}

	chapterEntities := ConvertChaptersToEntities(res.Data)

	// sort chapters in decdending order
	sort.Slice(chapterEntities, func(i, j int) bool {
		return chapterEntities[i].Date.After(*chapterEntities[j].Date)
	})

	mangaUpdateTime := time.Now()
	if chapterEntities[0].Date != nil {
		mangaUpdateTime = *chapterEntities[0].Date
	}
	return &types.MangaEntity{
		Name:         manga.Name,
		ShouldNotify: manga.ShouldNotify,
		LastUpdate:   mangaUpdateTime,
		Slug:         manga.Slug,
		Status:       manga.Status,
		Source:       manga.Source,
		Chapters:     chapterEntities,
	}, nil
}

// IsNewerVersionAvailable implements Provider.
func (mdp *mangaDexProvider) IsNewerVersionAvailable(manga types.MangaEntity) (bool, error) {

	if manga.IsNew() {
		return true, nil
	}

	v := url.Values{}
	v.Add("title", "Academy’s Genius Swordmaster")

	res, err := mdp.mangaDexClient.Manga.GetMangaList(v)

	if err != nil {
		return false, err
	}

	// try to find manga in the list response
	for _, m := range res.Data {
		if m.ID == manga.Slug {
			// check if the latest chapter is inside the list of chapters we have
			if m.Attributes.LastChapter != nil {
				for _, c := range manga.Chapters {
					// if we have the latest chapter exist, we don't need to update
					if c.Slug == m.Attributes.LastChapter {
						return false, nil
					}
				}
				// we have not found the latest chapter, we need to update
				return false, nil
			} else {
				return false, errors.New("can't find the manga's latest chapter")
			}
		}
	}
	return false, errors.New("can't find the manga in the list, or the list is empty")
}

// Kind implements Provider.
func (*mangaDexProvider) Kind() types.MangaSource {
	return types.MangaSourceMangaDex
}

func ConvertChaptersToEntities(chapters []m.Chapter) []types.ChapterEntity {
	var chapterEntities []types.ChapterEntity
	for _, chapter := range chapters {
		chapterEntity, err := convertChapterToEntity(chapter)
		if err == nil {
			chapterEntities = append(chapterEntities, chapterEntity)
		}
	}
	return chapterEntities
}

// convertChapterToEntity : Converts a single Chapter to a ChapterEntity.
func convertChapterToEntity(chapter m.Chapter) (types.ChapterEntity, error) {
	var number *float64
	var date *time.Time

	// Convert Chapter to float64 if possible.
	if chapter.Attributes.Chapter != nil {
		chapNum, err := strconv.ParseFloat(chapter.GetChapterNum(), 64)
		if err != nil {
			return types.ChapterEntity{}, err
		}
		number = &chapNum
	}

	// Convert CreatedAt to *time.Time if possible.
	if chapter.Attributes.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339, chapter.Attributes.PublishAt)
		if err != nil || t.IsZero() {
			return types.ChapterEntity{}, err
		}
		date = &t
	}

	slug := &chapter.ID

	return types.ChapterEntity{
		Number: number,
		Slug:   slug,
		Date:   date,
	}, nil
}
