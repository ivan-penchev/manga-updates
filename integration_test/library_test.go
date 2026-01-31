package integrationtest

import (
	"context"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/library"
	"github.com/ivan-penchev/manga-updates/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestLibrary_AddSeries_Manganel(t *testing.T) {
	// This test uses the real Provider (router) initiated in TestMain
	// It relies on fetching data from a real URL using the Provider's implementation.

	memStore := store.NewMemoryStore()
	lib := library.NewLibrary(memStore, providerRouter) // providerRouter from main_test.go

	// Use a known stable manga URL
	url := "https://manganel.me/manga/solo-leveling_102"
	// Or equivalent if that one is redirected or invalid. Solo leveling often changes or ends?
	// The other test used "solo-leveling_102" as Slug, which implies it might be related.
	// Let's stick to the URL format supported by validation.

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := lib.AddSeries(ctx, url)
	if err != nil {
		t.Fatalf("AddSeries failed: %v", err)
	}

	// Verify persistence in memory store
	series := memStore.GetMangaSeries(ctx)
	manga, ok := series["solo-leveling_102"]

	assert.True(t, ok, "Manga should be present in store")
	assert.Equal(t, domain.MangaSourceMangaNel, manga.Source)
	assert.Equal(t, "solo-leveling_102", manga.Slug)
	assert.NotEmpty(t, manga.Name)
	assert.NotEmpty(t, manga.Chapters)
}
