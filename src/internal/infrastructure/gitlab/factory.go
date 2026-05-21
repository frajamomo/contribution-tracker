package gitlab

import (
	"contribution-tracker/internal/application"
)

// FetcherFactory creates GitLab ActivityFetcher instances.
type FetcherFactory struct {
	baseURL string
}

// NewFetcherFactory creates a new GitLab fetcher factory.
func NewFetcherFactory(baseURL string) *FetcherFactory {
	return &FetcherFactory{baseURL: baseURL}
}

// Build creates a new ActivityFetcher configured with the given API key.
func (f *FetcherFactory) Build(apiKey string) application.ActivityFetcher {
	client := NewApiClient(f.baseURL, apiKey)
	return NewActivityFetcher(client)
}
