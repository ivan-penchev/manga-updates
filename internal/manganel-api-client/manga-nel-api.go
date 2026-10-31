package manganelapiclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/machinebox/graphql"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultRetryDelay = 150 * time.Millisecond
	maxAttempts       = 3
	defaultOrigin     = "https://manganel.me"
	defaultReferer    = "https://manganel.me/"
	defaultUserAgent  = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Mobile Safari/537.36"
	defaultSecChUA    = `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138""`
)

// Option configures a MangaNelAPIClient.
type Option func(*MangaNelAPIClient)

// WithHTTPClient sets a custom HTTP client on the MangaNelAPIClient.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *MangaNelAPIClient) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithRetryDelay configures the base backoff delay between retry attempts.
func WithRetryDelay(delay time.Duration) Option {
	return func(c *MangaNelAPIClient) {
		if delay >= 0 {
			c.retryDelay = delay
		}
	}
}

type MangaNelAPIClient struct {
	addr       string
	apiKey     string
	httpClient *http.Client
	client     *graphql.Client
	retryDelay time.Duration
}

func NewMangaNelAPIClient(addr string, apiKey string, opts ...Option) *MangaNelAPIClient {
	c := &MangaNelAPIClient{
		addr:   addr,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		retryDelay: defaultRetryDelay,
	}
	for _, opt := range opts {
		opt(c)
	}

	c.client = graphql.NewClient(addr, graphql.WithHTTPClient(c.httpClient))
	return c
}

type mangaQueryResponse struct {
	Manga *mangaGraphQLData `json:"manga"`
}

type mangaGraphQLData struct {
	Title       string               `json:"title"`
	Slug        string               `json:"slug"`
	Status      string               `json:"status"`
	UpdatedDate string               `json:"updatedDate"`
	Chapters    []chapterGraphQLData `json:"chapters"`
}

type chapterGraphQLData struct {
	Number float64 `json:"number"`
	Slug   string  `json:"slug"`
	Date   string  `json:"date"`
}

func (c *MangaNelAPIClient) GetMangaSeriesFull(ctx context.Context, slug string) (*domain.MangaEntity, error) {
	return c.getMangaSeries(ctx, slug, true)
}

func (c *MangaNelAPIClient) GetMangaSeriesShort(ctx context.Context, slug string) (*domain.MangaEntity, error) {
	return c.getMangaSeries(ctx, slug, false)
}

func (c *MangaNelAPIClient) getMangaSeries(ctx context.Context, slug string, shouldIncludeChapters bool) (*domain.MangaEntity, error) {
	req := c.newRequest(getQueryForSlug(slug, shouldIncludeChapters))

	var resp mangaQueryResponse
	if err := c.runWithRetry(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("querying manganel series %q: %w", slug, err)
	}

	if resp.Manga == nil || resp.Manga.Slug == "" {
		return nil, errors.New("manga not found in graphQL response")
	}

	lastUpdate, err := time.Parse(time.RFC3339, resp.Manga.UpdatedDate)
	if err != nil && resp.Manga.UpdatedDate != "" {
		slog.Warn("failed to parse manga updatedDate", "slug", slug, "updatedDate", resp.Manga.UpdatedDate, "error", err)
	}

	manga := domain.MangaEntity{
		Name:       resp.Manga.Title,
		Slug:       resp.Manga.Slug,
		Status:     domain.MangaStatus(resp.Manga.Status),
		Source:     domain.MangaSourceMangaNel,
		LastUpdate: lastUpdate,
		Chapters:   make([]domain.ChapterEntity, 0, len(resp.Manga.Chapters)),
	}
	if !shouldIncludeChapters {
		return &manga, nil
	}

	for _, ch := range resp.Manga.Chapters {
		number := ch.Number
		chapterDate, err := time.Parse(time.RFC3339, ch.Date)
		if err != nil && ch.Date != "" {
			slog.Warn("failed to parse chapter date", "slug", slug, "chapter", number, "date", ch.Date, "error", err)
		}

		chapter := domain.ChapterEntity{
			Number: &number,
			Date:   &chapterDate,
			URI:    fmt.Sprintf("%s/chapter/%s/chapter-%v", defaultOrigin, manga.Slug, number),
		}
		manga.Chapters = append(manga.Chapters, chapter)
	}

	sort.Slice(manga.Chapters, func(i, j int) bool {
		if manga.Chapters[i].Number == nil || manga.Chapters[j].Number == nil {
			return manga.Chapters[i].Number != nil
		}
		return *manga.Chapters[i].Number > *manga.Chapters[j].Number
	})

	return &manga, nil
}

func (c *MangaNelAPIClient) newRequest(query string) *graphql.Request {
	req := graphql.NewRequest(query)
	req.Header.Set("Origin", defaultOrigin)
	req.Header.Set("Referer", defaultReferer)
	req.Header.Set("X-Mhub-Access", c.apiKey)
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Sec-Ch-Ua", defaultSecChUA)
	return req
}

func (c *MangaNelAPIClient) runWithRetry(ctx context.Context, req *graphql.Request, resp any) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = c.client.Run(ctx, req, resp)
		if lastErr == nil {
			return nil
		}

		if attempt < maxAttempts && c.retryDelay > 0 {
			timer := time.NewTimer(c.retryDelay * time.Duration(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return lastErr
}

func getQueryForSlug(slug string, includeChapters bool) string {
	if includeChapters {
		return fmt.Sprintf(`
	{
		latestPopular(x:mn05) {
			id,
			rank,
			title,
			slug,
			image,
			latestChapter,
			updatedDate
		}
		manga(x: mn05, slug: %q) {
			id,
			rank,
			title,
			slug,
			status,
			image,
			latestChapter,
			author,
			artist,
			genres,
			description,
			alternativeTitle,
			mainSlug,
			isYaoi,
			isPorn,
			isSoftPorn,
			noCoverAd,
			isLicensed,
			createdDate,
			updatedDate,
			chapters {
				id,
				number,
				title,
				slug,
				date
			}
		}
	}	
`, slug)
	}
	return fmt.Sprintf(`
{
	manga(x: mn05, slug: %q) {
		title
		slug
		status
		image
		genres
		latestChapter
		createdDate
		updatedDate
	}
}	
`, slug)
}

func (c *MangaNelAPIClient) Search(ctx context.Context, query string, offset int) (*SearchResponse, error) {
	req := c.newRequest(buildSearchQuery(query, offset))

	var res SearchResponse
	if err := c.runWithRetry(ctx, req, &res); err != nil {
		return nil, fmt.Errorf("searching manganel for %q: %w", query, err)
	}
	return &res, nil
}

func buildSearchQuery(q string, offset int) string {
	return fmt.Sprintf(`
{
	search(x:mn05,q:%q,genre:"all",mod:POPULAR,count:true,offset:%d){
		rows{
			id,rank,title,slug,status,author,genres,image,latestChapter,isLicensed,createdDate
		},
		count
	}
}`, q, offset)
}

type SearchResponse struct {
	Search struct {
		Count int `json:"count"`
		Rows  []struct {
			ID            int    `json:"id"`
			Rank          int    `json:"rank"`
			Title         string `json:"title"`
			Slug          string `json:"slug"`
			Status        string `json:"status"`
			Image         string `json:"image"`
			LatestChapter any    `json:"latestChapter"`
		} `json:"rows"`
	} `json:"search"`
}
