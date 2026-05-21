package application

import (
	"testing"

	"contribution-tracker/internal/domain"
)

func TestRegistry_RegisterAndBuild(t *testing.T) {
	registry := NewActivityFetcherRegistry()
	fetcher := &mockActivityFetcher{platforms: []domain.GitPlatform{domain.PlatformGitHub}}
	factory := &mockFetcherFactory{fetcher: fetcher}

	registry.Register(domain.PlatformGitHub, factory)

	built, err := registry.Build(domain.PlatformGitHub, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if built != fetcher {
		t.Error("expected the mock fetcher to be returned")
	}
}

func TestRegistry_UnknownPlatform(t *testing.T) {
	registry := NewActivityFetcherRegistry()

	_, err := registry.Build(domain.GitPlatform{Name: "BITBUCKET"}, "key")
	if err == nil {
		t.Fatal("expected error for unknown platform")
	}
}

func TestRegistry_Platforms(t *testing.T) {
	registry := NewActivityFetcherRegistry()
	registry.Register(domain.PlatformGitHub, &mockFetcherFactory{})
	registry.Register(domain.PlatformGitLab, &mockFetcherFactory{})

	platforms := registry.Platforms()
	if len(platforms) != 2 {
		t.Errorf("expected 2 platforms, got %d", len(platforms))
	}
}
