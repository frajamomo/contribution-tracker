package application

import (
	"fmt"

	"contribution-tracker/internal/domain"
)

type ActivityFetcherRegistry struct {
	factories map[domain.GitPlatform]FetcherFactory
}

func NewActivityFetcherRegistry() *ActivityFetcherRegistry {
	return &ActivityFetcherRegistry{factories: make(map[domain.GitPlatform]FetcherFactory)}
}

func (r *ActivityFetcherRegistry) Register(platform domain.GitPlatform, factory FetcherFactory) {
	r.factories[platform] = factory
}

func (r *ActivityFetcherRegistry) Build(platform domain.GitPlatform, apiKey string) (ActivityFetcher, error) {
	factory, ok := r.factories[platform]
	if !ok {
		return nil, fmt.Errorf("no fetcher registered for platform %s", platform.Name)
	}
	return factory.Build(apiKey), nil
}

func (r *ActivityFetcherRegistry) Platforms() []domain.GitPlatform {
	platforms := make([]domain.GitPlatform, 0, len(r.factories))
	for p := range r.factories {
		platforms = append(platforms, p)
	}
	return platforms
}
