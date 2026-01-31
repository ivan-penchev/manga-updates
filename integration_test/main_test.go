package integrationtest

import (
	"os"
	"testing"

	"github.com/ivan-penchev/manga-updates/internal/domain"
	"github.com/ivan-penchev/manga-updates/internal/provider"
)

var providerRouter domain.ProviderRouter

func TestMain(m *testing.M) {
	var err error
	providerRouter, err = provider.NewProviderRouter(
		provider.NewMangaNelProviderFactory(provider.MangaNelProviderConfig{
			GraphQLEndpoint: "https://api.mghcdn.com/graphql",
			RemoteChromeURL: os.Getenv("REMOTE_CHROME_URL"),
		}),
	)
	if err != nil {
		panic(err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
