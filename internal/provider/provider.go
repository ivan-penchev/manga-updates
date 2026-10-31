package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

// Create a new router and sets one provider per source
func NewProviderRouter(providerFactories ...func() (domain.Provider, error)) (domain.ProviderRouter, error) {
	if len(providerFactories) == 0 {
		return nil, fmt.Errorf("no provider factories provided")
	}
	providersMap := make(map[domain.MangaSource]domain.Provider)
	var initErrors error

	for _, factory := range providerFactories {
		if factory == nil {
			initErrors = errors.Join(initErrors, errors.New("provider factory is nil"))
			continue
		}

		provider, err := factory()
		if err != nil {
			initErrors = errors.Join(initErrors, err)
			continue
		}
		if provider == nil {
			initErrors = errors.Join(initErrors, errors.New("provider factory returned nil provider"))
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
