package provider

import (
	"context"
	"fmt"

	"log/slog"
	"sync"
	"time"

	"strings"

	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
)

type MangaNelProviderConfig struct {
	GraphQLEndpoint string
	RemoteChromeURL string
}

func NewMangaNelProviderFactory(cfg MangaNelProviderConfig) func() (domain.Provider, error) {
	return func() (domain.Provider, error) {

		// Increased timeout to allow for browser download if needed
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		var allocCtx context.Context
		var cancelAlloc context.CancelFunc

		remoteURL := cfg.RemoteChromeURL
		if remoteURL != "" {
			allocCtx, cancelAlloc = chromedp.NewRemoteAllocator(ctx, remoteURL)

		} else {
			// Always use managed browser to avoid issues with system installs (e.g. shims)
			path, err := launcher.NewBrowser().Get()
			if err != nil {
				return nil, fmt.Errorf("failed to download/find browser: %w", err)
			}

			opts := append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.ExecPath(path),
				chromedp.Flag("no-sandbox", true),
				chromedp.Flag("headless", true),
				chromedp.Flag("disable-gpu", true),
				chromedp.Flag("disable-dev-shm-usage", true),
			)
			allocCtx, cancelAlloc = chromedp.NewExecAllocator(ctx, opts...)
		}

		defer cancelAlloc()

		// Create the chromedp context from the allocator
		innerCtx, cancelInner := chromedp.NewContext(allocCtx)
		defer cancelInner()
		pageToOpen := "https://manganel.me/"

		// navigate to a page, wait for an element, click
		var mhubApiAccessToken string
		err := chromedp.Run(innerCtx,
			chromedp.Emulate(device.IPhone15Pro),
			chromedp.Navigate(pageToOpen),
			chromedp.WaitVisible("body", chromedp.ByQuery),
			chromedp.ActionFunc(func(ctx context.Context) error {
				cookies, err := storage.GetCookies().Do(ctx)
				if err != nil {
					return err
				}

				for _, cookie := range cookies {
					if cookie.Name == "mhub_access" {
						mhubApiAccessToken = cookie.Value
					}
				}

				return nil
			}),
		)

		if mhubApiAccessToken == "" || err != nil {
			slog.Warn("failed to find manganel access cookie", "error", err)
			return nil, err
		}

		mangaNelClient := manganelapiclient.NewMangaNelAPIClient(cfg.GraphQLEndpoint, mhubApiAccessToken)

		return &mangaNelProvider{
			mangaNelClient:  mangaNelClient,
			cachedResponses: make(map[string]*domain.MangaEntity, 0),
			mutex:           sync.RWMutex{},
		}, nil
	}

}

type mangaNelProvider struct {
	mangaNelClient  *manganelapiclient.MangaNelAPIClient
	cachedResponses map[string]*domain.MangaEntity
	mutex           sync.RWMutex
}

func (mp *mangaNelProvider) Supports(url string) bool {
	return strings.Contains(url, "manganel.me")
}

func extractSlugFromURL(u string) (string, error) {
	parts := strings.Split(u, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid url: %s", u)
	}
	slug := parts[len(parts)-1]
	if slug == "" {
		return "", fmt.Errorf("could not extract slug from url: %s", u)
	}
	return slug, nil
}

func (mp *mangaNelProvider) GetMangaFromURL(ctx context.Context, u string) (domain.MangaEntity, error) {
	// url example: https://chapmanganato.to/manga-dr980474 OR https://manganato.com/manga-dr980474
	// The slug is often the last part.
	// NOTE: This implementation is very basic and fragile to URL structure changes.
	slug, err := extractSlugFromURL(u)
	if err != nil {
		return domain.MangaEntity{}, err
	}

	// Create a dummy entity to fetch details
	dummy := domain.MangaEntity{
		Slug:   slug,
		Source: domain.MangaSourceMangaNel,
	}

	// Use GetLatestVersionMangaEntity to fetch actual data
	fetched, err := mp.GetLatestVersionMangaEntity(ctx, dummy)
	if err != nil {
		return domain.MangaEntity{}, err
	}
	if fetched == nil {
		return domain.MangaEntity{}, fmt.Errorf("could not fetch manga details for slug %s", slug)
	}

	// Ensure source is set correctly on returned entity if not already
	fetched.Source = domain.MangaSourceMangaNel

	return *fetched, nil
}

func (mp *mangaNelProvider) IsNewerVersionAvailable(ctx context.Context, manga domain.MangaEntity) (bool, error) {
	if manga.IsNew() {
		logMessage := fmt.Sprintf("Manga title (%s) that we have never synced before added for update notifications", manga.Name)
		slog.Info(logMessage)
		return true, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(context.Background(), manga.Slug)

	if err != nil {
		return false, err
	}

	mangaResponse.ShouldNotify = manga.ShouldNotify

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.cachedResponses[manga.Slug] = mangaResponse

	return manga.IsOlder(*mangaResponse), nil
}

func (mnp *mangaNelProvider) Search(ctx context.Context, query string, offset int) ([]domain.SearchResult, int, error) {
	res, err := mnp.mangaNelClient.Search(ctx, query, offset)
	if err != nil {
		return nil, 0, err
	}

	var results []domain.SearchResult
	for _, row := range res.Search.Rows {
		// Construct URL if possible or just store slug. Manganel URL usually depends on the specific site being used. But we can store the slug.
		results = append(results, domain.SearchResult{
			Manga: domain.MangaEntity{
				Name:   row.Title,
				Slug:   row.Slug,
				Status: domain.MangaStatus(row.Status),
				Source: domain.MangaSourceMangaNel,
			},
			Rank:          row.Rank,
			ImageURL:      "https://avt.mkklcdnv6temp.com/" + row.Image, // This base URL is a guess, might need adjustment or config
			URL:           "https://chapmanganato.to/" + row.Slug,       // Also a guess based on usual behavior
			LatestChapter: fmt.Sprintf("%v", row.LatestChapter),
		})
	}

	return results, res.Search.Count, nil
}

func (*mangaNelProvider) Kind() domain.MangaSource {
	return domain.MangaSourceMangaNel
}

func (mp *mangaNelProvider) GetLatestVersionMangaEntity(ctx context.Context, manga domain.MangaEntity) (*domain.MangaEntity, error) {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	if cachedManga, ok := mp.cachedResponses[manga.Slug]; ok {
		return cachedManga, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(context.Background(), manga.Slug)
	if err != nil {
		return nil, err
	}
	mangaResponse.ShouldNotify = manga.ShouldNotify
	return mangaResponse, nil
}
