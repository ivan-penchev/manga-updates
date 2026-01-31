package library_test

import (
	"context"
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/library"
	"github.com/ivan-penchev/manga-updates/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Kind() domain.MangaSource {
	args := m.Called()
	return args.Get(0).(domain.MangaSource)
}

func (m *MockProvider) GetLatestVersionMangaEntity(ctx context.Context, manga domain.MangaEntity) (*domain.MangaEntity, error) {
	args := m.Called(ctx, manga)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MangaEntity), args.Error(1)
}

func (m *MockProvider) GetMangaFromURL(ctx context.Context, url string) (domain.MangaEntity, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(domain.MangaEntity), args.Error(1)
}

func (m *MockProvider) IsNewerVersionAvailable(ctx context.Context, manga domain.MangaEntity) (bool, error) {
	args := m.Called(ctx, manga)
	return args.Bool(0), args.Error(1)
}

func (m *MockProvider) Supports(url string) bool {
	args := m.Called(url)
	return args.Bool(0)
}

type MockRouter struct {
	mock.Mock
}

func (m *MockRouter) GetProvider(manga domain.MangaEntity) (domain.Provider, error) {
	args := m.Called(manga)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(domain.Provider), args.Error(1)
}

func TestAddSeries(t *testing.T) {
	memStore := store.NewMemoryStore()
	mockRouter := new(MockRouter)
	mockProvider := new(MockProvider)
	lib := library.NewLibrary(memStore, mockRouter)

	url := "https://manganel.me/manga-123"
	// The library currently maps this URL to MangaNel
	expectedSource := domain.MangaSourceMangaNel

	mockRouter.On("GetProvider", mock.MatchedBy(func(m domain.MangaEntity) bool {
		return m.Source == expectedSource
	})).Return(mockProvider, nil)

	expectedManga := domain.MangaEntity{
		Name:         "Test Manga",
		Slug:         "test-manga",
		Source:       expectedSource,
		ShouldNotify: true,
	}
	mockProvider.On("GetMangaFromURL", mock.Anything, url).Return(expectedManga, nil)

	err := lib.AddSeries(context.Background(), url)
	assert.NoError(t, err)

	// Verify persistence in memory store
	series := memStore.GetMangaSeries(context.Background())
	assert.Len(t, series, 1)
	savedManga, ok := series["test-manga"]
	assert.True(t, ok)
	assert.Equal(t, expectedManga.Name, savedManga.Name)

	mockRouter.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestAddSeries_InvalidURL(t *testing.T) {
	memStore := store.NewMemoryStore()
	mockRouter := new(MockRouter)
	lib := library.NewLibrary(memStore, mockRouter)

	err := lib.AddSeries(context.Background(), "invalid-url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid url")
}

func TestAddSeries_UnsupportedSource(t *testing.T) {
	memStore := store.NewMemoryStore()
	mockRouter := new(MockRouter)
	lib := library.NewLibrary(memStore, mockRouter)

	err := lib.AddSeries(context.Background(), "https://unknown-site.com/manga")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no provider found")
}
