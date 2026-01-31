package provider

import (
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestProviderRouter_GetProviderForURL(t *testing.T) {
	mockProvider1 := mocks.NewMockProvider(t)
	mockProvider2 := mocks.NewMockProvider(t)

	// Setup router with mocks
	router := &providerRouter{
		providers: map[domain.MangaSource]domain.Provider{
			domain.MangaSourceMangaNel: mockProvider1,
			domain.MangaSourceMangaDex: mockProvider2,
		},
	}

	url := "https://manganel.me/manga-123"

	// Mock Provider 1 supports it
	mockProvider1.EXPECT().Supports(url).Return(true)
	// Provider 2 might be called or not depending on map iteration order
	mockProvider2.EXPECT().Supports(url).Return(false).Maybe()

	p, err := router.GetProviderForURL(url)
	assert.NoError(t, err)
	assert.Equal(t, mockProvider1, p)
}

func TestProviderRouter_GetProviderForURL_NotFound(t *testing.T) {
	mockProvider1 := mocks.NewMockProvider(t)
	router := &providerRouter{
		providers: map[domain.MangaSource]domain.Provider{
			domain.MangaSourceMangaNel: mockProvider1,
		},
	}

	url := "https://unknown.com/manga"
	mockProvider1.EXPECT().Supports(url).Return(false)

	p, err := router.GetProviderForURL(url)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "no provider found")
}
