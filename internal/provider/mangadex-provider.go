package provider

import (
	"errors"
	"fmt"
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
	initialResponse, err := mdp.mangaDexClient.Chapter.GetMangaChapters(manga.Slug, v)

	if err != nil || initialResponse.Data == nil || len(initialResponse.Data) == 0 {
		return nil, errors.Join(err, errors.New("failed to get list of chapters for manga"))
	}

	chapters := initialResponse.Data

	for len(chapters) < initialResponse.Total {
		v.Add("offset", strconv.Itoa(len(chapters)))
		res, err := mdp.mangaDexClient.Chapter.GetMangaChapters(manga.Slug, v)

		if err != nil || res.Data == nil || len(res.Data) == 0 {
			break
		}

		chapters = append(chapters, res.Data...)
	}

	var chapterEntities []types.ChapterEntity
	for _, chapter := range chapters {
		chapterEntity, err := convertChapterToEntity(chapter)
		if err == nil {
			chapterEntities = append(chapterEntities, chapterEntity)
		}
	}

	// sort chapters in decdending order
	sort.Slice(chapterEntities, func(i, j int) bool {
		return chapterEntities[i].Number != nil &&
			chapterEntities[j].Number != nil &&
			*chapterEntities[i].Number > *chapterEntities[j].Number
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

	v := url.Values{}
	v.Add("title", manga.Name)
	v.Add("order[relevance]", "desc")

	res, err := mdp.mangaDexClient.Manga.GetMangaList(v)

	if err != nil {
		return false, err
	}

	// try to find manga in the list response
	for _, m := range res.Data {
		if m.ID == manga.Slug {
			// check if the latest chapter is inside the list of chapters we have
			// but only do so, if the latest chapter is not empty, otherwise we can't compare.
			if m.Attributes.LastChapter != nil && *m.Attributes.LastChapter != "" {
				for _, c := range manga.Chapters {
					// if we have the latest chapter exist, we don't need to update
					if c.Slug == m.Attributes.LastChapter {
						return false, nil
					}
				}
				// we have not found the latest chapter, we need to update
				return true, nil
			} else {
				// if the latest chapter on the manga object is empty,
				// we can't compare, so we need to fetch the whole list of chapters
				// and compare the latest chapter from the list,
				// we do that by saying we need to update.
				return true, nil
			}

		}
	}

	return false, errors.New("can't find the manga in the response from the api, or the response is empty")
}

func (*mangaDexProvider) Kind() types.MangaSource {
	return types.MangaSourceMangaDex
}

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

	uri := fmt.Sprintf("https://mangadex.org/chapter/%s", chapter.ID)
	if chapter.Attributes.ExternalURL != nil && *chapter.Attributes.ExternalURL != "" {
		uri = *chapter.Attributes.ExternalURL
	}

	return types.ChapterEntity{
		Number: number,
		Slug:   slug,
		Date:   date,
		URI:    uri,
	}, nil
}
