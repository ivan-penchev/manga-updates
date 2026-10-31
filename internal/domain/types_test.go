package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMangaEntity_IsNew(t *testing.T) {
	t.Parallel()

	now := time.Now()
	chNum := 1.0

	tests := []struct {
		name     string
		entity   MangaEntity
		expected bool
	}{
		{
			name:     "empty chapters and zero lastUpdate is new",
			entity:   MangaEntity{},
			expected: true,
		},
		{
			name: "non-zero lastUpdate is not new",
			entity: MangaEntity{
				LastUpdate: now,
			},
			expected: false,
		},
		{
			name: "non-empty chapters is not new",
			entity: MangaEntity{
				Chapters: []ChapterEntity{{Number: &chNum}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := tt.entity
			assert.Equal(t, tt.expected, e.IsNew())
		})
	}
}

func TestMangaEntity_IsOlder(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	older := MangaEntity{LastUpdate: t1}
	newer := MangaEntity{LastUpdate: t2}

	assert.True(t, older.IsOlder(newer))
	assert.False(t, newer.IsOlder(older))
	assert.False(t, older.IsOlder(older))
}

func TestMangaEntity_GetMissingChapters(t *testing.T) {
	t.Parallel()

	num1, num2, num3 := 1.0, 2.0, 3.0
	slug1, slug2, slug3 := "ch-1", "ch-2", "ch-3"
	ch1 := ChapterEntity{Number: &num1, Slug: &slug1}
	ch2 := ChapterEntity{Number: &num2, Slug: &slug2}
	ch3 := ChapterEntity{Number: &num3, Slug: &slug3}

	tests := []struct {
		name     string
		current  MangaEntity
		latest   MangaEntity
		expected []ChapterEntity
	}{
		{
			name:     "new chapter prepended",
			current:  MangaEntity{Chapters: []ChapterEntity{ch2, ch1}},
			latest:   MangaEntity{Chapters: []ChapterEntity{ch3, ch2, ch1}},
			expected: []ChapterEntity{ch3},
		},
		{
			name:     "no new chapters",
			current:  MangaEntity{Chapters: []ChapterEntity{ch2, ch1}},
			latest:   MangaEntity{Chapters: []ChapterEntity{ch2, ch1}},
			expected: []ChapterEntity{},
		},
		{
			name:     "latest has fewer chapters and no new ones",
			current:  MangaEntity{Chapters: []ChapterEntity{ch2, ch1}},
			latest:   MangaEntity{Chapters: []ChapterEntity{ch1}},
			expected: []ChapterEntity{},
		},
		{
			name:     "new chapter added while old chapter removed upstream",
			current:  MangaEntity{Chapters: []ChapterEntity{ch2, ch1}},
			latest:   MangaEntity{Chapters: []ChapterEntity{ch3, ch2}},
			expected: []ChapterEntity{ch3},
		},
		{
			name:     "fallback to length diff when chapters have no identity",
			current:  MangaEntity{Chapters: []ChapterEntity{{}, {}}},
			latest:   MangaEntity{Chapters: []ChapterEntity{{}, {}, {}}},
			expected: []ChapterEntity{{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cur := tt.current
			got := cur.GetMissingChapters(tt.latest)
			assert.Equal(t, tt.expected, got)
		})
	}
}
