package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"contribution-tracker/internal/domain"
)

type ReportService struct {
	users    UserRepository
	teams    TeamRepository
	repos    RepositoryStore
	config   ConfigRepository
	registry *ActivityFetcherRegistry
}

func NewReportService(
	users UserRepository,
	teams TeamRepository,
	repos RepositoryStore,
	config ConfigRepository,
	registry *ActivityFetcherRegistry,
) *ReportService {
	return &ReportService{
		users:    users,
		teams:    teams,
		repos:    repos,
		config:   config,
		registry: registry,
	}
}

func (s *ReportService) GenerateReport(ctx context.Context, query ReportQuery, out chan<- ReportEvent) {
	defer close(out)

	team, err := s.teams.FindByID(ctx, query.TeamID)
	if err != nil {
		out <- &ReportErrorEvent{Message: fmt.Sprintf("team not found: %v", err)}
		return
	}

	var memberIDs []string
	isOnlyMember := !query.CallerRoles[domain.RoleTeamLeader] && !query.CallerRoles[domain.RoleAdmin]
	if isOnlyMember {
		memberIDs = []string{query.CallerID}
	} else {
		memberIDs = team.MemberIDs
	}

	members, err := s.users.FindByIDs(ctx, memberIDs)
	if err != nil {
		out <- &ReportErrorEvent{Message: fmt.Sprintf("failed to load members: %v", err)}
		return
	}

	teamRepos, err := s.repos.FindByIDs(ctx, team.RepositoryIDs)
	if err != nil {
		out <- &ReportErrorEvent{Message: fmt.Sprintf("failed to load repos: %v", err)}
		return
	}

	reposByPlatform := make(map[domain.GitPlatform][]domain.Repository)
	for _, r := range teamRepos {
		reposByPlatform[r.Platform] = append(reposByPlatform[r.Platform], r)
	}

	apiKeys := make(map[domain.GitPlatform]string)
	for platform := range reposByPlatform {
		key, err := s.config.Get(ctx, platform.Name+"_API_KEY")
		if err != nil {
			slog.Warn("no API key configured", "platform", platform.Name)
			continue
		}
		apiKeys[platform] = key
	}

	for _, member := range members {
		report := s.fetchMemberReport(ctx, member, reposByPlatform, apiKeys, query)
		out <- &UserReportEvent{Report: report}
	}

	out <- &ReportCompleteEvent{}
}

func (s *ReportService) fetchMemberReport(
	ctx context.Context,
	member domain.User,
	reposByPlatform map[domain.GitPlatform][]domain.Repository,
	apiKeys map[domain.GitPlatform]string,
	query ReportQuery,
) UserReport {
	var allActivities []domain.Activity
	var mu sync.Mutex
	var wg sync.WaitGroup

	for platform, repos := range reposByPlatform {
		apiKey, ok := apiKeys[platform]
		if !ok {
			continue
		}

		fetcher, err := s.registry.Build(platform, apiKey)
		if err != nil {
			slog.Error("failed to build fetcher", "platform", platform.Name, "err", err)
			continue
		}

		platformUsername := member.GetPlatformUsername(platform)

		// ISP check — ADR-5
		if discoverer, ok := fetcher.(RepoDiscoverer); ok {
			discovered, err := discoverer.DiscoverUserRepos(ctx, platformUsername)
			if err == nil {
				for _, repo := range discovered {
					s.repos.Upsert(ctx, &repo)
				}
			}
		}

		wg.Add(1)
		go func(f ActivityFetcher, repos []domain.Repository, username string) {
			defer wg.Done()

			for _, repo := range repos {
				activities, err := f.FetchForUser(ctx, username, repo, query.Since, query.Until, query.Types)
				if err != nil {
					slog.Error("fetch error", "repo", repo.FullName, "err", err)
					continue
				}
				mu.Lock()
				allActivities = append(allActivities, activities...)
				mu.Unlock()
			}

			searched, err := f.SearchActivities(ctx, username, repos, query.Since, query.Until, query.Types)
			if err != nil {
				slog.Error("search error", "user", username, "err", err)
			} else {
				mu.Lock()
				allActivities = append(allActivities, searched...)
				mu.Unlock()
			}
		}(fetcher, repos, platformUsername)
	}

	wg.Wait()

	counts := buildActivityCounts(allActivities)

	return UserReport{
		User:       member,
		Counts:     counts,
		Activities: allActivities,
	}
}

func buildActivityCounts(activities []domain.Activity) []ActivityCount {
	countMap := make(map[domain.ActivityType]int)
	for _, a := range activities {
		countMap[a.GetType()]++
	}

	counts := make([]ActivityCount, 0, len(countMap))
	for t, c := range countMap {
		counts = append(counts, ActivityCount{Type: t, Count: c})
	}
	return counts
}
