package provider

import (
	"fmt"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type providerRouter struct {
	providers map[domain.MangaSource]domain.Provider
}

func (p *providerRouter) GetProvider(manga domain.MangaEntity) (domain.Provider, error) {
	provider, ok := p.providers[manga.Source]
	if !ok {
		return nil, fmt.Errorf("provider for %s not found", manga.Source)
	}
	return provider, nil
}

func (p *providerRouter) GetProviderForURL(url string) (domain.Provider, error) {
	for _, provider := range p.providers {
		if provider.Supports(url) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no provider found for url: %s", url)
}
