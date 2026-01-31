package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type memoryStore struct {
	mu     sync.RWMutex
	mangas map[string]domain.MangaEntity
}

// NewMemoryStore creates an in-memory implementation of the Store interface.
// Useful for testing or ephemeral storage.
func NewMemoryStore() Store {
	return &memoryStore{
		mangas: make(map[string]domain.MangaEntity),
	}
}

func (s *memoryStore) GetMangaSeries(ctx context.Context) map[string]domain.MangaEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]domain.MangaEntity, len(s.mangas))
	for k, v := range s.mangas {
		result[k] = v
	}
	return result
}

func (s *memoryStore) PersistManagaTitle(ctx context.Context, location string, mangaTitle domain.MangaEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mangas[location] = mangaTitle
	return nil
}

func (s *memoryStore) AddManga(ctx context.Context, manga domain.MangaEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := manga.Slug
	if _, exists := s.mangas[key]; exists {
		return fmt.Errorf("manga with slug %s already exists", manga.Slug)
	}
	s.mangas[key] = manga
	return nil
}
