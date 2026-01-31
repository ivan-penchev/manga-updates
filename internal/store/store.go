package store

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"log/slog"

	"github.com/ivan-penchev/manga-updates/internal/domain"
)

type Store interface {
	GetMangaSeries(ctx context.Context) map[string]domain.MangaEntity
	PersistManagaTitle(ctx context.Context, location string, mangaTitle domain.MangaEntity) error
}

type fileStore struct {
	location string
}

// PersistestManagaTitle implements Store
func (f *fileStore) PersistManagaTitle(ctx context.Context, location string, mangaTitle domain.MangaEntity) error {
	file, _ := json.MarshalIndent(mangaTitle, "", " ")
	return os.WriteFile(location, file, 0644)
}

// GetMangaSeries returns the file location and file data
func (f *fileStore) GetMangaSeries(ctx context.Context) map[string]domain.MangaEntity {
	persistedMangaSeries := make(map[string]domain.MangaEntity, 0)
	files := glob(f.location, func(s string) bool {
		return filepath.Ext(s) == ".json"
	})
	for _, file := range files {
		byteValue, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Cant open file")
			continue
		}
		var mangaSeries domain.MangaEntity
		err = json.Unmarshal(byteValue, &mangaSeries)
		if err != nil {
			slog.Error("File is not in correct structure")
			continue
		}
		if mangaSeries.Slug == "" || mangaSeries.Slug == "<insert id string>" {
			slog.Error("Slug for the manga title is not formatted correctly")
			continue
		}
		persistedMangaSeries[file] = mangaSeries
	}

	return persistedMangaSeries
}

func NewStore(location string) Store {
	return &fileStore{
		location: location,
	}
}

func glob(root string, fn func(string) bool) []string {
	var files []string
	err := filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if fn(s) {
			files = append(files, s)
		}
		return nil
	})
	if err != nil {
		return make([]string, 0)
	}
	return files
}
