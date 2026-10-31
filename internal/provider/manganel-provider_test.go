package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	manganelapiclient "github.com/ivan-penchev/manga-updates/internal/manganel-api-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMangaNelClient struct {
	series *domain.MangaEntity
	search *manganelapiclient.SearchResponse
	err    error
	calls  int
}

func (s *stubMangaNelClient) GetMangaSeriesFull(_ context.Context, _ string) (*domain.MangaEntity, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	copySeries := *s.series
	return &copySeries, nil
}

func (s *stubMangaNelClient) Search(_ context.Context, _ string, _ int) (*manganelapiclient.SearchResponse, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.search, nil
}

func TestFetchAccessTokenViaHTTP(t *testing.T) {
	t.Parallel()

	t.Run("extracts mhub_access cookie from HTTP response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:  mhubAccessCookieName,
				Value: "cookie-secret-123",
				Path:  "/",
			})
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		token, err := fetchAccessTokenViaHTTP(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "cookie-secret-123", token)
	})

	t.Run("returns error when cookie is absent", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		token, err := fetchAccessTokenViaHTTP(context.Background(), srv.URL)
		require.Error(t, err)
		assert.Empty(t, token)
	})
}

func TestExtractSlugFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "standard manganel url",
			url:  "https://manganel.me/manga/one-piece_116",
			want: "one-piece_116",
		},
		{
			name: "trailing slash is trimmed",
			url:  "https://manganel.me/manga/one-piece_116/",
			want: "one-piece_116",
		},
		{
			name:    "root url without slug fails",
			url:     "https://manganel.me/",
			wantErr: true,
		},
		{
			name:    "empty url fails",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractSlugFromURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMangaNelProvider_UpdateAndCacheFlow(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	ch1 := 1.0

	stub := &stubMangaNelClient{
		series: &domain.MangaEntity{
			Name:       "One Piece",
			Slug:       "one-piece_116",
			Status:     domain.MangaStatusOngoing,
			Source:     domain.MangaSourceMangaNel,
			LastUpdate: t2,
			Chapters:   []domain.ChapterEntity{{Number: &ch1}},
		},
	}

	p := NewMangaNelProvider(stub)
	assert.Equal(t, domain.MangaSourceMangaNel, p.Kind())
	assert.True(t, p.Supports("https://manganel.me/manga/one-piece_116"))

	current := domain.MangaEntity{
		Name:         "One Piece",
		Slug:         "one-piece_116",
		ShouldNotify: true,
		LastUpdate:   t1,
		Chapters:     []domain.ChapterEntity{{Number: &ch1}},
	}

	newer, err := p.IsNewerVersionAvailable(context.Background(), current)
	require.NoError(t, err)
	assert.True(t, newer)
	assert.Equal(t, 1, stub.calls)

	// Subsequent GetLatestVersionMangaEntity should be served from cache without an extra network call.
	latest, err := p.GetLatestVersionMangaEntity(context.Background(), current)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.True(t, latest.ShouldNotify)
	assert.Equal(t, 1, stub.calls)
}
