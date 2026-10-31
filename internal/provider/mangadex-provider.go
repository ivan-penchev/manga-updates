package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	m "github.com/darylhjd/mangodex"
	"github.com/ivan-penchev/manga-updates/internal/domain"
)

const chapterFloatEpsilon = 0.0001

var (
	_                  domain.Provider = (*mangaDexProvider)(nil)
	mangaDexTitleRegex                 = regexp.MustCompile(`mangadex\.org/title/([a-f0-9\-]+)`)
)

type mangaDexClient interface {
	GetMangaList(params url.Values) (*m.MangaList, error)
	GetMangaChapters(mangaID string, params url.Values) (*m.ChapterList, error)
}

type dexClientAdapter struct {
	client *m.DexClient
}

func (a *dexClientAdapter) GetMangaList(params url.Values) (*m.MangaList, error) {
	return a.client.Manga.GetMangaList(params)
}

func (a *dexClientAdapter) GetMangaChapters(mangaID string, params url.Values) (*m.ChapterList, error) {
	return a.client.Chapter.GetMangaChapters(mangaID, params)
}

type mangaDexProvider struct {
	client mangaDexClient
}

func NewMangaDexProvider(client mangaDexClient) *mangaDexProvider {
	return &mangaDexProvider{
		client: client,
	}
}

func (mdp *mangaDexProvider) Supports(url string) bool {
	return strings.Contains(url, "mangadex.org")
}

func (mdp *mangaDexProvider) GetMangaFromURL(ctx context.Context, u string) (domain.MangaEntity, error) {
	matches := mangaDexTitleRegex.FindStringSubmatch(u)
	if len(matches) < 2 {
		return domain.MangaEntity{}, fmt.Errorf("invalid mangadex url: %s", u)
	}
	mangaID := matches[1]

	v := url.Values{}
	v.Add("ids[]", mangaID)
	mangaList, err := mdp.client.GetMangaList(v)
	if err != nil {
		return domain.MangaEntity{}, fmt.Errorf("failed to fetch manga details: %w", err)
	}

	if mangaList == nil || len(mangaList.Data) == 0 {
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
		return NewMangaDexProvider(&dexClientAdapter{client: m.NewDexClient()}), nil
	}
}

// GetLatestVersionMangaEntity implements Provider.
func (mdp *mangaDexProvider) GetLatestVersionMangaEntity(_ context.Context, manga domain.MangaEntity) (*domain.MangaEntity, error) {
	v := url.Values{}
	v.Set("translatedLanguage[]", "en")
	initialResponse, err := mdp.client.GetMangaChapters(manga.Slug, v)

	if err != nil || initialResponse == nil || len(initialResponse.Data) == 0 {
		return nil, errors.Join(err, errors.New("failed to get list of chapters for manga"))
	}

	chapters := initialResponse.Data

	for len(chapters) < initialResponse.Total {
		v.Set("offset", strconv.Itoa(len(chapters)))
		res, err := mdp.client.GetMangaChapters(manga.Slug, v)
		if err != nil || res == nil || len(res.Data) == 0 {
			break
		}

		chapters = append(chapters, res.Data...)
	}

	chapterEntities := make([]domain.ChapterEntity, 0, len(chapters))
	for _, chapter := range chapters {
		chapterEntity, err := convertChapterToEntity(chapter)
		if err == nil {
			chapterEntities = append(chapterEntities, chapterEntity)
		}
	}

	sort.Slice(chapterEntities, func(i, j int) bool {
		return chapterEntities[i].Number != nil &&
			chapterEntities[j].Number != nil &&
			*chapterEntities[i].Number > *chapterEntities[j].Number
	})

	mangaUpdateTime := time.Now()
	if len(chapterEntities) > 0 && chapterEntities[0].Date != nil {
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
func (mdp *mangaDexProvider) IsNewerVersionAvailable(_ context.Context, manga domain.MangaEntity) (bool, error) {
	if manga.IsNew() {
		slog.Info("Manga title that we have never synced before added for update notifications", "mangaName", manga.Name)
		return true, nil
	}

	v := url.Values{}
	v.Set("limit", "1")
	v.Set("translatedLanguage[]", "en")
	v.Set("order[chapter]", "desc")

	chaptersRes, err := mdp.client.GetMangaChapters(manga.Slug, v)
	if err != nil {
		return false, fmt.Errorf("failed to fetch latest chapter: %w", err)
	}

	if chaptersRes == nil || len(chaptersRes.Data) == 0 {
		return false, nil
	}

	latestChapter := chaptersRes.Data[0]

	latestEntity, err := convertChapterToEntity(latestChapter)
	if err != nil {
		return true, nil
	}

	for _, c := range manga.Chapters {
		if c.Slug != nil && latestEntity.Slug != nil && *c.Slug == *latestEntity.Slug {
			return false, nil
		}
	}

	if latestEntity.Number != nil {
		for _, c := range manga.Chapters {
			if c.Number != nil && math.Abs(*c.Number-*latestEntity.Number) < chapterFloatEpsilon {
				return false, nil
			}
		}
	}

	return true, nil
}

func (*mangaDexProvider) Kind() domain.MangaSource {
	return domain.MangaSourceMangaDex
}

func (mdp *mangaDexProvider) Search(_ context.Context, query string, offset int) ([]domain.SearchResult, int, error) {
	v := url.Values{}
	v.Set("title", query)
	v.Set("limit", "20")
	v.Set("offset", strconv.Itoa(offset))
	v.Set("order[followedCount]", "desc")
	v.Set("order[relevance]", "desc")

	v.Add("contentRating[]", "safe")
	v.Add("contentRating[]", "suggestive")
	v.Add("contentRating[]", "erotica")

	v.Add("includes[]", "cover_art")

	mangaList, err := mdp.client.GetMangaList(v)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search mangadex: %w", err)
	}
	if mangaList == nil {
		return nil, 0, nil
	}

	results := make([]domain.SearchResult, 0, len(mangaList.Data))
	for _, item := range mangaList.Data {
		if item.Type != "manga" {
			continue
		}

		title := item.Attributes.Title.GetLocalString("en")
		if title == "" {
			title = item.Attributes.AltTitles.GetLocalString("en")
			if title == "" {
				title = "Unknown Title"
			}
		}

		var imageURL string
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" {
				if attrs, ok := rel.Attributes.(map[string]any); ok {
					if fileName, ok := attrs["fileName"].(string); ok {
						imageURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", item.ID, fileName)
					}
				}
			}
		}

		var latestChapter string
		if item.Attributes.LastChapter != nil {
			latestChapter = *item.Attributes.LastChapter
		}

		var status domain.MangaStatus
		if item.Attributes.Status != nil {
			status = domain.MangaStatus(*item.Attributes.Status)
		}

		results = append(results, domain.SearchResult{
			Manga: domain.MangaEntity{
				Name:   title,
				Slug:   item.ID,
				Status: status,
				Source: domain.MangaSourceMangaDex,
			},
			ImageURL:      imageURL,
			URL:           fmt.Sprintf("https://mangadex.org/title/%s", item.ID),
			LatestChapter: latestChapter,
		})
	}

	return results, mangaList.Total, nil
}

func convertChapterToEntity(chapter m.Chapter) (domain.ChapterEntity, error) {
	var number *float64
	var date *time.Time

	if chapter.Attributes.Chapter != nil {
		chapNum, err := strconv.ParseFloat(chapter.GetChapterNum(), 64)
		if err != nil {
			return domain.ChapterEntity{}, err
		}
		number = &chapNum
	}

	dateStr := chapter.Attributes.PublishAt
	if dateStr == "" {
		dateStr = chapter.Attributes.CreatedAt
	}
	if dateStr != "" {
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil || t.IsZero() {
			return domain.ChapterEntity{}, err
		}
		date = &t
	}

	slug := chapter.ID

	uri := fmt.Sprintf("https://mangadex.org/chapter/%s", chapter.ID)
	if chapter.Attributes.ExternalURL != nil && *chapter.Attributes.ExternalURL != "" {
		uri = *chapter.Attributes.ExternalURL
	}

	return domain.ChapterEntity{
		Number: number,
		Slug:   &slug,
		Date:   date,
		URI:    uri,
	}, nil
}
