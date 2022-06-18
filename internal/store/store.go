package store

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/ivan-penchev/manga-updates/pkg/types"
)

type Store interface {
	GetMangaSeries() map[string]types.MangaEntity
	PersistestManagaTitle(location string, mangaTitle types.MangaEntity) error
}

type fileStore struct {
	location string
}

// PersistestManagaTitle implements Store
func (*fileStore) PersistestManagaTitle(location string, mangaTitle types.MangaEntity) error {
	file, _ := json.MarshalIndent(mangaTitle, "", " ")
	return ioutil.WriteFile(location, file, 0644)
}

// GetMangaSeries returns the file location and file data
func (f *fileStore) GetMangaSeries() map[string]types.MangaEntity {
	persistedMangaSeries := make(map[string]types.MangaEntity, 0)
	files := glob(f.location, func(s string) bool {
		return filepath.Ext(s) == ".json"
	})
	for _, file := range files {
		byteValue, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println("Cant open file")
			continue
		}
		var mangaSeries types.MangaEntity
		err = json.Unmarshal(byteValue, &mangaSeries)
		if err != nil {
			fmt.Println("File is not in correct structure")
			continue
		}
		if mangaSeries.Slug == "" || mangaSeries.Slug == "<insert id string>" {
			fmt.Println("Slug for the manga title is not formatted correctly")
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
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if fn(s) {
			files = append(files, s)
		}
		return nil
	})
	return files
}
