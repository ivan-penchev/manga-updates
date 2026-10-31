package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotifier_Validation(t *testing.T) {
	t.Parallel()

	t.Run("defaults to stdout notifier when no API key is configured", func(t *testing.T) {
		t.Parallel()
		n, err := NewNotifier()
		require.NoError(t, err)
		require.NoError(t, n.NotifyForNewChapter(context.Background(), domain.ChapterEntity{}, domain.MangaEntity{Name: "Test"}))
	})

	t.Run("rejects invalid sender email for smtp2go", func(t *testing.T) {
		t.Parallel()
		_, err := NewNotifier(
			WithSMTP2GOAPIKey("key"),
			WithSenderEmail("not-an-email"),
			WithRecipients("user@example.com"),
		)
		require.Error(t, err)
	})

	t.Run("rejects missing recipients for sendgrid", func(t *testing.T) {
		t.Parallel()
		_, err := NewNotifier(
			WithSendGridAPIKey("key"),
			WithSenderEmail("sender@example.com"),
		)
		require.Error(t, err)
	})
}

func TestSMTP2GONotifier_NotifyForNewChapter(t *testing.T) {
	t.Parallel()

	t.Run("sends templated email and handles fractional or nil chapter number safely", func(t *testing.T) {
		t.Parallel()

		var received smpt2goEmail
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-smtp-key", r.Header.Get("X-Smtp2go-Api-Key"))
			require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"request_id": "req-1", "data": {}}`))
		}))
		defer srv.Close()

		n, err := NewNotifier(
			WithSMTP2GOAPIKey("test-smtp-key"),
			WithSenderEmail("manga@example.com"),
			WithRecipients("reader@example.com"),
			WithTemplateID("tpl-123"),
			WithHTTPClient(srv.Client()),
			WithAPIRoot(srv.URL),
		)
		require.NoError(t, err)

		chNum := 68.5
		err = n.NotifyForNewChapter(
			context.Background(),
			domain.ChapterEntity{Number: &chNum, URI: "https://manganel.me/ch-68.5"},
			domain.MangaEntity{Name: "Solo Leveling: Ragnarok"},
		)
		require.NoError(t, err)
		assert.Equal(t, "tpl-123", received.TemplateID)

		// Also verify nil chapter.Number does not panic
		err = n.NotifyForNewChapter(
			context.Background(),
			domain.ChapterEntity{Number: nil, URI: "https://manganel.me/ch-special"},
			domain.MangaEntity{Name: "Solo Leveling: Ragnarok"},
		)
		require.NoError(t, err)
	})
}
