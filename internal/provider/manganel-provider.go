package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
	"github.com/ivan-penchev/manga-updates/pkg/types"
)

func NewMangaNelProviderFactory(mangaNelGraphQLEndpoint string) func() (Provider, error) {
	return func() (Provider, error) {

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		innerCtx, _ := chromedp.NewContext(ctx)

		randomPageFromProviderToOpen := "https://manganel.me/manga/my-wife-is-a-demon-queen"

		// navigate to a page, wait for an element, click
		var mhubApiAccessToken string
		err := chromedp.Run(innerCtx,
			chromedp.Emulate(device.IPhone12),
			chromedp.Navigate(randomPageFromProviderToOpen),
			chromedp.Sleep(4*time.Second),
			chromedp.ActionFunc(func(ctx context.Context) error {
				cookies, err := network.GetCookies().Do(ctx)
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
			cachedResponses: make(map[string]*types.MangaEntity, 0),
			mutex:           sync.RWMutex{},
		}, nil
	}

}

type mangaNelProvider struct {
	mangaNelClient  *manganelapiclient.MangaNelAPIClient
	cachedResponses map[string]*types.MangaEntity
	mutex           sync.RWMutex
}

func (mp *mangaNelProvider) IsNewerVersionAvailable(manga types.MangaEntity) (bool, error) {
	if manga.IsNew() {
		logMessage := fmt.Sprintf("Manga title (%s) that we have never synced before added for update notifications", manga.Name)
		slog.Info(logMessage)
		return true, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(manga.Slug)

	if err != nil {
		return false, err
	}

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.cachedResponses[manga.Slug] = mangaResponse

	return manga.IsOlder(*mangaResponse), nil
}

func (*mangaNelProvider) Kind() types.MangaSource {
	return types.MangaSourceMangaNel
}

func (mp *mangaNelProvider) GetLatestVersionMangaEntity(manga types.MangaEntity) (*types.MangaEntity, error) {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	if cachedManga, ok := mp.cachedResponses[manga.Slug]; ok {
		return cachedManga, nil
	}

	mangaResponse, err := mp.mangaNelClient.GetMangaSeriesFull(manga.Slug)

	if err != nil {
		return nil, err
	}
	return mangaResponse, nil
}
