package provider

import (
	"fmt"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type providerRouter struct {
	providers map[types.MangaSource]Provider
}

func (p *providerRouter) GetProvider(manga types.MangaEntity) (Provider, error) {
	provider, ok := p.providers[manga.Source]
	if !ok {
		return nil, fmt.Errorf("provider for %s not found", manga.Source)
	}
	return provider, nil
}
