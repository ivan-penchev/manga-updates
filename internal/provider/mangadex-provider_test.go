package provider

import (
	"context"
	"net/url"
	"testing"
	"time"

	m "github.com/darylhjd/mangodex"
	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMangaDexClient struct {
	mangaList   *m.MangaList
	chapterPages map[string]*m.ChapterList
	seenOffsets []string
	err         error
}

func (s *stubMangaDexClient) GetMangaList(_ url.Values) (*m.MangaList, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.mangaList, nil
}

func (s *stubMangaDexClient) GetMangaChapters(_ string, params url.Values) (*m.ChapterList, error) {
	if s.err != nil {
		return nil, s.err
	}
	offsets := params["offset"]
	offsetKey := ""
	if len(offsets) > 0 {
		// Verify offset is never duplicated in query params (v.Set vs v.Add bug)
		if len(offsets) > 1 {
			offsetKey = "DUPLICATE_OFFSET"
		} else {
			offsetKey = offsets[0]
		}
		s.seenOffsets = append(s.seenOffsets, offsetKey)
	}
	if page, ok := s.chapterPages[offsetKey]; ok {
		return page, nil
	}
	return &m.ChapterList{}, nil
}

func makeDexChapter(id, chapNum, publishAt string) m.Chapter {
	var ch m.Chapter
	ch.ID = id
	ch.Attributes.Chapter = &chapNum
	ch.Attributes.PublishAt = publishAt
	ch.Attributes.CreatedAt = publishAt
	return ch
}

func TestMangaDexProvider_GetLatestVersionMangaEntity_Pagination(t *testing.T) {
	t.Parallel()

	stub := &stubMangaDexClient{
		chapterPages: map[string]*m.ChapterList{
			"": {
				Total: 3,
				Data: []m.Chapter{
					makeDexChapter("id-1", "1", "2025-01-01T00:00:00Z"),
				},
			},
			"1": {
				Total: 3,
				Data: []m.Chapter{
					makeDexChapter("id-2", "2", "2025-01-02T00:00:00Z"),
				},
			},
			"2": {
				Total: 3,
				Data: []m.Chapter{
					makeDexChapter("id-3", "3", "2025-01-03T00:00:00Z"),
				},
			},
		},
	}

	p := NewMangaDexProvider(stub)
	manga, err := p.GetLatestVersionMangaEntity(context.Background(), domain.MangaEntity{
		Name:   "Test Manga",
		Slug:   "uuid-123",
		Source: domain.MangaSourceMangaDex,
	})
	require.NoError(t, err)
	require.NotNil(t, manga)
	require.Len(t, manga.Chapters, 3)
	assert.Equal(t, []string{"1", "2"}, stub.seenOffsets)
	assert.Equal(t, 3.0, *manga.Chapters[0].Number)
	assert.Equal(t, 2.0, *manga.Chapters[1].Number)
	assert.Equal(t, 1.0, *manga.Chapters[2].Number)
}

func TestMangaDexProvider_IsNewerVersionAvailable(t *testing.T) {
	t.Parallel()

	now := time.Now()
	slug1 := "id-1"
	num1 := 1.0

	stub := &stubMangaDexClient{
		chapterPages: map[string]*m.ChapterList{
			"": {
				Total: 1,
				Data: []m.Chapter{
					makeDexChapter("id-2", "2", "2025-01-02T00:00:00Z"),
				},
			},
		},
	}

	p := NewMangaDexProvider(stub)

	t.Run("detects newer chapter when neither slug nor number exists", func(t *testing.T) {
		t.Parallel()
		current := domain.MangaEntity{
			Name:       "Test",
			Slug:       "uuid-123",
			LastUpdate: now,
			Chapters:   []domain.ChapterEntity{{Slug: &slug1, Number: &num1}},
		}
		newer, err := p.IsNewerVersionAvailable(context.Background(), current)
		require.NoError(t, err)
		assert.True(t, newer)
	})

	t.Run("returns false when chapter number already exists from another group", func(t *testing.T) {
		t.Parallel()
		num2 := 2.0
		otherSlug := "other-id"
		current := domain.MangaEntity{
			Name:       "Test",
			Slug:       "uuid-123",
			LastUpdate: now,
			Chapters:   []domain.ChapterEntity{{Slug: &otherSlug, Number: &num2}},
		}
		newer, err := p.IsNewerVersionAvailable(context.Background(), current)
		require.NoError(t, err)
		assert.False(t, newer)
	})
}
