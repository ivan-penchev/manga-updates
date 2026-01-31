package library

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type Store interface {
	GetMangaSeries(ctx context.Context) map[string]domain.MangaEntity
	PersistManagaTitle(ctx context.Context, location string, mangaTitle domain.MangaEntity) error
	AddManga(ctx context.Context, manga domain.MangaEntity) error
}

type Library struct {
	store          Store
	providerRouter domain.ProviderRouter
}

func NewLibrary(store Store, providerRouter domain.ProviderRouter) *Library {
	return &Library{
		store:          store,
		providerRouter: providerRouter,
	}
}

func (l *Library) AddSeries(ctx context.Context, u string) error {
	// 1. Validate URL
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsedUrl.Scheme == "" || parsedUrl.Host == "" {
		return fmt.Errorf("invalid url: missing scheme or host")
	}

	// 2. Find provider
	// We iterate through all known providers (via router?)
	// Actually router maps Source -> Provider. We don't have a "Get all providers" easily exposed yet.
	// But `domain.ProviderRouter` only exposes `GetProvider(MangaEntity)`.
	// This is a limitation. We need the library to be able to iterate providers to ask "Do you support this URL?".
	// For now, we can try to guess the source from the URL host.

	// Poor man's router for adding:
	var source domain.MangaSource
	if strings.Contains(u, "manganel.me") {
		source = domain.MangaSourceMangaNel
	} else if strings.Contains(u, "mangadex.org") {
		source = domain.MangaSourceMangaDex
	} else {
		return fmt.Errorf("no provider found supporting url: %s", u)
	}

	// 3. Get provider instance
	provider, err := l.providerRouter.GetProvider(domain.MangaEntity{Source: source})
	if err != nil {
		return fmt.Errorf("failed to get provider for source %s: %w", source, err)
	}

	// 4. Fetch details
	manga, err := provider.GetMangaFromURL(ctx, u)
	if err != nil {
		return fmt.Errorf("failed to fetch manga details: %w", err)
	}

	// 5. Configure default settings
	manga.ShouldNotify = true
	manga.Source = source

	// 6. Persist
	return l.store.AddManga(ctx, manga)
}
