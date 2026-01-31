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
		provider.NewMangaNelProviderFactory("https://api.mghcdn.com/graphql"),
	)
	if err != nil {
		panic(err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
