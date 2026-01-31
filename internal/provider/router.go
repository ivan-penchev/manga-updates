package provider

import (
	"fmt"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type providerRouter struct {
	providers map[domain.MangaSource]Provider
}

func (p *providerRouter) GetProvider(manga domain.MangaEntity) (Provider, error) {
	provider, ok := p.providers[manga.Source]
	if !ok {
		return nil, fmt.Errorf("provider for %s not found", manga.Source)
	}
	return provider, nil
}
