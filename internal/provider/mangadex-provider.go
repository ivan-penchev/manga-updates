package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	m "github.com/darylhjd/mangodex"
	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type mangaDexProvider struct {
	mangaDexClient  *m.DexClient
	cachedResponses map[string]*domain.MangaEntity
	mutex           sync.RWMutex
}

func (mdp *mangaDexProvider) Supports(url string) bool {
	return strings.Contains(url, "mangadex.org")
}

func (mdp *mangaDexProvider) GetMangaFromURL(ctx context.Context, u string) (domain.MangaEntity, error) {
	// Regex for: https://mangadex.org/title/633d470a-4146-4dd3-b841-93dd648c23a5/...
	// We extract the UUID
	re := regexp.MustCompile(`mangadex\.org/title/([a-f0-9\-]+)`)
	matches := re.FindStringSubmatch(u)
	if len(matches) < 2 {
		return domain.MangaEntity{}, fmt.Errorf("invalid mangadex url: %s", u)
	}
	mangaID := matches[1]

	// Fetch details using GetMangaList with ID filter as GetManga might not exist or verify existence
	v := url.Values{}
	v.Add("ids[]", mangaID)
	mangaList, err := mdp.mangaDexClient.Manga.GetMangaList(v)
	if err != nil {
		return domain.MangaEntity{}, fmt.Errorf("failed to fetch manga details: %w", err)
	}

	if len(mangaList.Data) == 0 {
		return domain.MangaEntity{}, fmt.Errorf("manga details not found for id: %s", mangaID)
	}

	mangaData := mangaList.Data[0]

	title := mangaData.Attributes.Title.GetLocalString("en")
	if title == "" {
		title = mangaData.Attributes.AltTitles.GetLocalString("en")
		if title == "" {
			title = "Unknown Title"
		}
	}

	dummy := domain.MangaEntity{
		Slug:         mangaID,
		Name:         title,
		Source:       domain.MangaSourceMangaDex,
		ShouldNotify: true,
	}

	fetched, err := mdp.GetLatestVersionMangaEntity(ctx, dummy)
	if err != nil {
		return domain.MangaEntity{}, err
	}

	if fetched == nil {
		return domain.MangaEntity{}, fmt.Errorf("failed to fetch chapters for manga %s", mangaID)
	}

	return *fetched, nil
}

func NewMangaDexProviderFactory() func() (domain.Provider, error) {
	return func() (domain.Provider, error) {

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
			cachedResponses: make(map[string]*domain.MangaEntity, 0),
			mutex:           sync.RWMutex{},
		}, nil
	}

}

// GetLatestVersionMangaEntity implements Provider.
func (mdp *mangaDexProvider) GetLatestVersionMangaEntity(ctx context.Context, manga domain.MangaEntity) (*domain.MangaEntity, error) {
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

	var chapterEntities []domain.ChapterEntity
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
	return &domain.MangaEntity{
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
func (mdp *mangaDexProvider) IsNewerVersionAvailable(ctx context.Context, manga domain.MangaEntity) (bool, error) {
	if manga.IsNew() {
		logMessage := fmt.Sprintf("Manga title (%s) that we have never synced before added for update notifications", manga.Name)
		slog.Info(logMessage)
		return true, nil
	}

	v := url.Values{}
	v.Add("limit", "1")
	v.Add("translatedLanguage[]", "en")
	v.Add("order[chapter]", "desc")

	// Fetch the latest chapter for this manga from the API
	chaptersRes, err := mdp.mangaDexClient.Chapter.GetMangaChapters(manga.Slug, v)
	if err != nil {
		return false, fmt.Errorf("failed to fetch latest chapter: %w", err)
	}

	if len(chaptersRes.Data) == 0 {
		return false, nil
	}

	latestChapter := chaptersRes.Data[0]

	// Convert to entity to normalize fields using the same logic as fetch
	latestEntity, err := convertChapterToEntity(latestChapter)
	if err != nil {
		// If we can't parse the latest chapter, we force an update safely?
		// Or we log error? For now, let's assume if parsing fails we might want to check fully.
		return true, nil
	}

	// 1. Check if we already have this specific chapter ID
	for _, c := range manga.Chapters {
		if c.Slug != nil && latestEntity.Slug != nil && *c.Slug == *latestEntity.Slug {
			return false, nil
		}
	}

	// 2. Check if we have this chapter number (to avoid duplicates by different groups)
	if latestEntity.Number != nil {
		for _, c := range manga.Chapters {
			if c.Number != nil {
				diff := *c.Number - *latestEntity.Number
				if diff < 0 {
					diff = -diff
				}
				if diff < 0.0001 {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

func (*mangaDexProvider) Kind() domain.MangaSource {
	return domain.MangaSourceMangaDex
}

func convertChapterToEntity(chapter m.Chapter) (domain.ChapterEntity, error) {
	var number *float64
	var date *time.Time

	// Convert Chapter to float64 if possible.
	if chapter.Attributes.Chapter != nil {
		chapNum, err := strconv.ParseFloat(chapter.GetChapterNum(), 64)
		if err != nil {
			return domain.ChapterEntity{}, err
		}
		number = &chapNum
	}

	// Convert CreatedAt to *time.Time if possible.
	if chapter.Attributes.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339, chapter.Attributes.PublishAt)
		if err != nil || t.IsZero() {
			return domain.ChapterEntity{}, err
		}
		date = &t
	}

	slug := &chapter.ID

	uri := fmt.Sprintf("https://mangadex.org/chapter/%s", chapter.ID)
	if chapter.Attributes.ExternalURL != nil && *chapter.Attributes.ExternalURL != "" {
		uri = *chapter.Attributes.ExternalURL
	}

	return domain.ChapterEntity{
		Number: number,
		Slug:   slug,
		Date:   date,
		URI:    uri,
	}, nil
}
