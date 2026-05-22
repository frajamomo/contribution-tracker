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
	registry *ActivityFetcherRegistry
}

func NewReportService(
	users UserRepository,
	teams TeamRepository,
	repos RepositoryStore,
	registry *ActivityFetcherRegistry,
) *ReportService {
	return &ReportService{
		users:    users,
		teams:    teams,
		repos:    repos,
		registry: registry,
	}
}

type fetcherGroup struct {
	platform domain.GitPlatform
	token    string
	repos    []domain.Repository
}

func (s *ReportService) GenerateReport(ctx context.Context, query ReportQuery, out chan<- ReportEvent) {
	defer close(out)

	team, err := s.teams.FindByID(ctx, query.TeamID)
	if err != nil {
		out <- &ReportErrorEvent{Message: fmt.Sprintf("team not found: %v", err)}
		return
	}

	var memberIDs []string
	isLeaderOfThisTeam := false
	for _, lid := range team.LeaderIDs {
		if lid == query.CallerID {
			isLeaderOfThisTeam = true
			break
		}
	}
	isOnlyMember := !query.CallerRoles[domain.RoleAdmin] && !isLeaderOfThisTeam
	if isOnlyMember {
		memberIDs = []string{query.CallerID}
	} else if query.MemberID != "" {
		memberIDs = []string{query.MemberID}
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

	groupKey := func(r domain.Repository) string {
		return r.Platform.Name + "\x00" + r.APIToken
	}
	groupMap := make(map[string]*fetcherGroup)
	for _, r := range teamRepos {
		if r.APIToken == "" {
			slog.Warn("repository has no API token, skipping", "repo", r.FullName)
			continue
		}
		key := groupKey(r)
		if g, ok := groupMap[key]; ok {
			g.repos = append(g.repos, r)
		} else {
			groupMap[key] = &fetcherGroup{platform: r.Platform, token: r.APIToken, repos: []domain.Repository{r}}
		}
	}

	groups := make([]fetcherGroup, 0, len(groupMap))
	for _, g := range groupMap {
		groups = append(groups, *g)
	}

	for _, member := range members {
		report := s.fetchMemberReport(ctx, member, groups, query)
		out <- &UserReportEvent{Report: report}
	}

	out <- &ReportCompleteEvent{}
}

func (s *ReportService) fetchMemberReport(
	ctx context.Context,
	member domain.User,
	groups []fetcherGroup,
	query ReportQuery,
) UserReport {
	var allActivities []domain.Activity
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, group := range groups {
		fetcher, err := s.registry.Build(group.platform, group.token)
		if err != nil {
			slog.Error("failed to build fetcher", "platform", group.platform.Name, "err", err)
			continue
		}

		platformUsername := member.GetPlatformUsername(group.platform)

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
		}(fetcher, group.repos, platformUsername)
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
