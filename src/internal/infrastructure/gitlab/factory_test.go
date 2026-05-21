package gitlab

import (
	"testing"

	"contribution-tracker/internal/application"
)

func TestFetcherFactory_Build_ReturnsActivityFetcher(t *testing.T) {
	factory := NewFetcherFactory("https://gitlab.com")
	fetcher := factory.Build("test-key")

	if fetcher == nil {
		t.Fatal("expected non-nil fetcher")
	}

	// Verify it implements ActivityFetcher.
	var _ application.ActivityFetcher = fetcher

	platforms := fetcher.GetSupportedPlatforms()
	if len(platforms) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(platforms))
	}
}

func TestFetcherFactory_Build_ReturnsRepoDiscoverer(t *testing.T) {
	factory := NewFetcherFactory("https://gitlab.com")
	fetcher := factory.Build("test-key")

	discoverer, ok := fetcher.(application.RepoDiscoverer)
	if !ok {
		t.Fatal("expected fetcher to implement RepoDiscoverer")
	}
	if discoverer == nil {
		t.Fatal("expected non-nil discoverer")
	}
}

func TestFetcherFactory_ImplementsInterface(t *testing.T) {
	var _ application.FetcherFactory = (*FetcherFactory)(nil)
}
