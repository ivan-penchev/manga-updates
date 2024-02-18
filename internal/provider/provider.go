package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type Provider interface {
	Kind() types.MangaSource
	GetLatestVersionMangaEntity(manga types.MangaEntity) (*types.MangaEntity, error)
	IsNewerVersionAvailable(manga types.MangaEntity) (bool, error)
}

type ProviderRouter interface {
	GetProvider(manga types.MangaEntity) (Provider, error)
}

// Create a new router and sets one provider per source
func NewProviderRouter(providerFactories ...func() (Provider, error)) (ProviderRouter, error) {
	if len(providerFactories) == 0 {
		return nil, fmt.Errorf("no provider factories provided")
	}
	providersMap := make(map[types.MangaSource]Provider)
	var initErrors error

	for _, factory := range providerFactories {
		provider, err := factory()
		if err != nil {
			initErrors = errors.Join(initErrors, err)
			continue
		}

		source := provider.Kind()
		if _, exists := providersMap[source]; exists {
			// Warn about duplicate providers but do not treat as a critical error
			slog.Warn("Trying to add a provider that already exists", "providerKind", source)
			continue
		}

		providersMap[source] = provider
	}

	if initErrors != nil {
		return nil, initErrors
	}

	return &providerRouter{
		providers: providersMap,
	}, nil
}
