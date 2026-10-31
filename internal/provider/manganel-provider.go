package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
)

const (
	defaultMangaNelSiteURL  = "https://manganel.me/"
	defaultMangaNelThumbURL = "https://thumb.mghcdn.com/"
	mhubAccessCookieName    = "mhub_access"
	httpCookieTimeout       = 15 * time.Second
	browserCookieTimeout    = 2 * time.Minute
	chromeWSReadTimeout     = 60 * time.Second
	browserMaxAttempts      = 2
	cookieFetchUserAgent    = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Mobile Safari/537.36"
)

var _ domain.Provider = (*mangaNelProvider)(nil)

type mangaNelClient interface {
	GetMangaSeriesFull(ctx context.Context, slug string) (*domain.MangaEntity, error)
	Search(ctx context.Context, query string, offset int) (*manganelapiclient.SearchResponse, error)
}

type MangaNelProviderConfig struct {
	GraphQLEndpoint string
	RemoteChromeURL string
	SiteURL         string
}

func NewMangaNelProvider(client mangaNelClient) *mangaNelProvider {
	return &mangaNelProvider{
		mangaNelClient:  client,
		cachedResponses: make(map[string]*domain.MangaEntity),
	}
}

func NewMangaNelProviderFactory(cfg MangaNelProviderConfig) func() (domain.Provider, error) {
	return func() (domain.Provider, error) {
		siteURL := cfg.SiteURL
		if siteURL == "" {
			siteURL = defaultMangaNelSiteURL
		}

		token, err := acquireMangaNelAccessToken(context.Background(), siteURL, cfg.RemoteChromeURL)
		if err != nil {
			slog.Warn("failed to find manganel access cookie", "error", err)
			return nil, err
		}

		mangaNelClient := manganelapiclient.NewMangaNelAPIClient(cfg.GraphQLEndpoint, token)
		return NewMangaNelProvider(mangaNelClient), nil
	}
}

func acquireMangaNelAccessToken(ctx context.Context, siteURL, remoteChromeURL string) (string, error) {
	// Fetch the cookie over plain HTTP first unless a remote Chrome URL is set.
	if remoteChromeURL == "" {
		token, err := fetchAccessTokenViaHTTP(ctx, siteURL)
		if err == nil && token != "" {
			return token, nil
		}
		slog.Debug("HTTP cookie extraction did not yield mhub_access; falling back to headless browser", "error", err)
	}

	var lastErr error
	for attempt := 1; attempt <= browserMaxAttempts; attempt++ {
		token, err := fetchAccessTokenViaBrowser(ctx, siteURL, remoteChromeURL)
		if err == nil && token != "" {
			return token, nil
		}
		if err == nil {
			err = errors.New("mhub_access cookie not found in browser storage")
		}
		lastErr = err
		slog.Warn("browser attempt to acquire manganel access cookie failed", "attempt", attempt, "error", err)
	}

	return "", fmt.Errorf("acquiring manganel access token: %w", lastErr)
}

func fetchAccessTokenViaHTTP(parentCtx context.Context, siteURL string) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, httpCookieTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, siteURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating cookie request: %w", err)
	}
	req.Header.Set("User-Agent", cookieFetchUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: httpCookieTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing cookie request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == mhubAccessCookieName && cookie.Value != "" {
			return cookie.Value, nil
		}
	}

	return "", errors.New("mhub_access cookie not present in HTTP response")
}

func fetchAccessTokenViaBrowser(parentCtx context.Context, siteURL, remoteChromeURL string) (string, error) {
	ctx, cancel := context.WithTimeout(parentCtx, browserCookieTimeout)
	defer cancel()

	var allocCtx context.Context
	var cancelAlloc context.CancelFunc

	if remoteChromeURL != "" {
		allocCtx, cancelAlloc = chromedp.NewRemoteAllocator(ctx, remoteChromeURL)
	} else {
		path, err := launcher.NewBrowser().Get()
		if err != nil {
			return "", fmt.Errorf("failed to download/find browser: %w", err)
		}

		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(path),
			chromedp.WSURLReadTimeout(chromeWSReadTimeout),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)
		allocCtx, cancelAlloc = chromedp.NewExecAllocator(ctx, opts...)
	}
	defer cancelAlloc()

	innerCtx, cancelInner := chromedp.NewContext(allocCtx)
	defer cancelInner()

	var mhubApiAccessToken string
	err := chromedp.Run(innerCtx,
		chromedp.Emulate(device.IPhone15Pro),
		chromedp.Navigate(siteURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := storage.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			for _, cookie := range cookies {
				if cookie.Name == mhubAccessCookieName && cookie.Value != "" {
					mhubApiAccessToken = cookie.Value
					break
				}
			}

			if mhubApiAccessToken == "" {
				return errors.New("mhub_access cookie not found")
			}

			return nil
		}),
	)
	if err != nil {
		return "", err
	}

	return mhubApiAccessToken, nil
}

type mangaNelProvider struct {
	mangaNelClient  mangaNelClient
	cachedResponses map[string]*domain.MangaEntity
	mutex           sync.RWMutex
}

func (mp *mangaNelProvider) Supports(url string) bool {
	return strings.Contains(url, "manganel.me")
}

func extractSlugFromURL(u string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		return "", fmt.Errorf("invalid url: %s", u)
	}
	parts := strings.Split(trimmed, "/")
	slug := parts[len(parts)-1]
	if slug == "" || strings.Contains(slug, ".") {
		return "", fmt.Errorf("could not extract slug from url: %s", u)
	}
	return slug, nil
}

func (mp *mangaNelProvider) GetMangaFromURL(ctx context.Context, u string) (domain.MangaEntity, error) {
	slug, err := extractSlugFromURL(u)
	if err != nil {
		return domain.MangaEntity{}, err
	}

	dummy := domain.MangaEntity{
		Slug:   slug,
		Source: domain.MangaSourceMangaNel,
	}

	fetched, err := mp.GetLatestVersionMangaEntity(ctx, dummy)
	if err != nil {
		return domain.MangaEntity{}, err
	}
	if fetched == nil {
		return domain.MangaEntity{}, fmt.Errorf("could not fetch manga details for slug %s", slug)
	}

	fetched.Source = domain.MangaSourceMangaNel
	return *fetched, nil
}

func (mp *mangaNelProvider) IsNewerVersionAvailable(ctx context.Context, manga domain.MangaEntity) (bool, error) {
	if manga.IsNew() {
		slog.Info("Manga title that we have never synced before added for update notifications", "mangaName", manga.Name)
		return true, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(ctx, manga.Slug)
	if err != nil {
		return false, err
	}

	mangaResponse.ShouldNotify = manga.ShouldNotify

	mp.mutex.Lock()
	if mp.cachedResponses == nil {
		mp.cachedResponses = make(map[string]*domain.MangaEntity)
	}
	mp.cachedResponses[manga.Slug] = mangaResponse
	mp.mutex.Unlock()

	return manga.IsOlder(*mangaResponse), nil
}

func (mp *mangaNelProvider) Search(ctx context.Context, query string, offset int) ([]domain.SearchResult, int, error) {
	res, err := mp.mangaNelClient.Search(ctx, query, offset)
	if err != nil {
		return nil, 0, err
	}

	results := make([]domain.SearchResult, 0, len(res.Search.Rows))
	for _, row := range res.Search.Rows {
		results = append(results, domain.SearchResult{
			Manga: domain.MangaEntity{
				Name:   row.Title,
				Slug:   row.Slug,
				Status: domain.MangaStatus(row.Status),
				Source: domain.MangaSourceMangaNel,
			},
			Rank:          row.Rank,
			ImageURL:      defaultMangaNelThumbURL + row.Image,
			URL:           defaultMangaNelSiteURL + row.Slug,
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
	cachedManga, ok := mp.cachedResponses[manga.Slug]
	mp.mutex.RUnlock()

	if ok && cachedManga != nil {
		copyManga := *cachedManga
		copyManga.ShouldNotify = manga.ShouldNotify
		return &copyManga, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(ctx, manga.Slug)
	if err != nil {
		return nil, err
	}
	mangaResponse.ShouldNotify = manga.ShouldNotify

	mp.mutex.Lock()
	if mp.cachedResponses == nil {
		mp.cachedResponses = make(map[string]*domain.MangaEntity)
	}
	mp.cachedResponses[manga.Slug] = mangaResponse
	mp.mutex.Unlock()

	return mangaResponse, nil
}
