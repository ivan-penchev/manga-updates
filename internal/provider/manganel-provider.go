package provider

import (
	"context"
	"fmt"
	"os"

	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
)

func NewMangaNelProviderFactory(mangaNelGraphQLEndpoint string) func() (Provider, error) {
	return func() (Provider, error) {

		// Increased timeout to allow for browser download if needed
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		var allocCtx context.Context
		var cancelAlloc context.CancelFunc

		remoteURL := os.Getenv("REMOTE_CHROME_URL")
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

		mangaNelClient := manganelapiclient.NewMangaNelAPIClient(mangaNelGraphQLEndpoint, mhubApiAccessToken)

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

func (mp *mangaNelProvider) IsNewerVersionAvailable(manga domain.MangaEntity) (bool, error) {
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

func (*mangaNelProvider) Kind() domain.MangaSource {
	return domain.MangaSourceMangaNel
}

func (mp *mangaNelProvider) GetLatestVersionMangaEntity(manga domain.MangaEntity) (*domain.MangaEntity, error) {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	if cachedManga, ok := mp.cachedResponses[manga.Slug]; ok {
		return cachedManga, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(context.Background(), manga.Slug)
	mangaResponse.ShouldNotify = manga.ShouldNotify

	if err != nil {
		return nil, err
	}
	return mangaResponse, nil
}
