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
