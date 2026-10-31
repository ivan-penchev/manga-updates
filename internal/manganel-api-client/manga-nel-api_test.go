package manganelapiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMangaNelAPIClient_GetMangaSeriesFull(t *testing.T) {
	t.Parallel()

	t.Run("parses full series and sorts chapters descending", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-api-key", r.Header.Get("X-Mhub-Access"))
			assert.Equal(t, defaultOrigin, r.Header.Get("Origin"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"manga": {
						"title": "One Piece",
						"slug": "one-piece_116",
						"status": "ongoing",
						"updatedDate": "2026-01-15T10:00:00Z",
						"chapters": [
							{"number": 1, "slug": "chapter-1", "date": "2020-01-01T00:00:00Z"},
							{"number": 2.5, "slug": "chapter-2.5", "date": "2020-01-02T00:00:00Z"}
						]
					}
				}
			}`))
		}))
		defer srv.Close()

		client := NewMangaNelAPIClient(srv.URL, "test-api-key", WithHTTPClient(srv.Client()), WithRetryDelay(0))
		manga, err := client.GetMangaSeriesFull(context.Background(), "one-piece_116")
		require.NoError(t, err)
		require.NotNil(t, manga)

		assert.Equal(t, "One Piece", manga.Name)
		assert.Equal(t, "one-piece_116", manga.Slug)
		assert.Equal(t, domain.MangaStatusOngoing, manga.Status)
		assert.Equal(t, domain.MangaSourceMangaNel, manga.Source)
		require.Len(t, manga.Chapters, 2)
		assert.Equal(t, 2.5, *manga.Chapters[0].Number)
		assert.Equal(t, "https://manganel.me/chapter/one-piece_116/chapter-2.5", manga.Chapters[0].URI)
		assert.Equal(t, 1.0, *manga.Chapters[1].Number)
	})

	t.Run("retries transient error and succeeds", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if attempts.Add(1) < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"manga": {
						"title": "Naruto",
						"slug": "naruto",
						"status": "complete",
						"updatedDate": "2025-05-01T00:00:00Z",
						"chapters": []
					}
				}
			}`))
		}))
		defer srv.Close()

		client := NewMangaNelAPIClient(srv.URL, "key", WithHTTPClient(srv.Client()), WithRetryDelay(time.Millisecond))
		manga, err := client.GetMangaSeriesShort(context.Background(), "naruto")
		require.NoError(t, err)
		assert.Equal(t, "Naruto", manga.Name)
		assert.Equal(t, int32(2), attempts.Load())
	})

	t.Run("returns error when manga is null in response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data": {"manga": null}}`))
		}))
		defer srv.Close()

		client := NewMangaNelAPIClient(srv.URL, "key", WithHTTPClient(srv.Client()), WithRetryDelay(0))
		manga, err := client.GetMangaSeriesFull(context.Background(), "missing")
		require.Error(t, err)
		assert.Nil(t, manga)
	})
}

func TestMangaNelAPIClient_Search(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"search": {
					"count": 1,
					"rows": [
						{
							"id": 1,
							"rank": 10,
							"title": "Bleach",
							"slug": "bleach",
							"status": "complete",
							"image": "thumb/bleach.jpg",
							"latestChapter": 686
						}
					]
				}
			}
		}`))
	}))
	defer srv.Close()

	client := NewMangaNelAPIClient(srv.URL, "key", WithHTTPClient(srv.Client()), WithRetryDelay(0))
	res, err := client.Search(context.Background(), "Bleach", 0)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.Search.Count)
	require.Len(t, res.Search.Rows, 1)
	assert.Equal(t, "Bleach", res.Search.Rows[0].Title)
}
