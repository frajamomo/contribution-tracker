package application

import (
	"context"
	"testing"
	"time"

	"contribution-tracker/internal/domain"
)

func setupReportService() (*ReportService, *mockTeamRepo, *mockUserRepo, *mockRepoStore) {
	userRepo := newMockUserRepo()
	teamRepo := newMockTeamRepo()
	repoStore := newMockRepoStore()
	registry := NewActivityFetcherRegistry()

	commit := domain.NewCommit(
		domain.ActivityData{Title: "fix bug", URL: "https://github.com/test", CreatedAt: time.Now()},
		"abc123", "fix: resolve issue",
	)

	fetcher := &mockActivityFetcher{
		platforms:  []domain.GitPlatform{domain.PlatformGitHub},
		types:      []domain.ActivityType{domain.ActivityTypeCommit},
		activities: []domain.Activity{commit},
	}
	registry.Register(domain.PlatformGitHub, &mockFetcherFactory{fetcher: fetcher})

	userRepo.users["u-alice"] = &domain.User{ID: "u-alice", Username: "alice"}
	userRepo.users["u-bob"] = &domain.User{ID: "u-bob", Username: "bob"}

	teamRepo.teams["t-eng"] = &domain.Team{
		ID:            "t-eng",
		Name:          "Engineering",
		MemberIDs:     []string{"u-alice", "u-bob"},
		RepositoryIDs: []string{"r-1"},
	}

	repoStore.repos["r-1"] = &domain.Repository{
		ID:       "r-1",
		Name:     "repo",
		FullName: "org/repo",
		Platform: domain.PlatformGitHub,
		APIToken: "test-key",
	}

	svc := NewReportService(userRepo, teamRepo, repoStore, registry)
	return svc, teamRepo, userRepo, repoStore
}

func TestReportService_TeamLeader_SeesAllMembers(t *testing.T) {
	svc, _, _, _ := setupReportService()

	query := ReportQuery{
		TeamID:      "t-eng",
		CallerID:    "u-alice",
		CallerRoles: map[domain.Role]bool{domain.RoleTeamLeader: true},
		Since:       time.Now().Add(-24 * time.Hour),
		Until:       time.Now(),
		Types:       []domain.ActivityType{domain.ActivityTypeCommit},
		ReportType:  domain.ReportTypeActivityLog,
	}

	out := make(chan ReportEvent, 10)
	svc.GenerateReport(context.Background(), query, out)

	var userReports int
	var gotComplete bool
	for event := range out {
		switch event.GetType() {
		case ReportEventTypeUserReport:
			userReports++
		case ReportEventTypeComplete:
			gotComplete = true
		}
	}

	if userReports != 2 {
		t.Errorf("expected 2 user reports (all members), got %d", userReports)
	}
	if !gotComplete {
		t.Error("expected COMPLETE event")
	}
}

func TestReportService_TeamMember_SeesOnlySelf(t *testing.T) {
	svc, _, _, _ := setupReportService()

	query := ReportQuery{
		TeamID:      "t-eng",
		CallerID:    "u-alice",
		CallerRoles: map[domain.Role]bool{domain.RoleTeamMember: true},
		Since:       time.Now().Add(-24 * time.Hour),
		Until:       time.Now(),
		Types:       []domain.ActivityType{domain.ActivityTypeCommit},
	}

	out := make(chan ReportEvent, 10)
	svc.GenerateReport(context.Background(), query, out)

	var userReports int
	for event := range out {
		if event.GetType() == ReportEventTypeUserReport {
			userReports++
			ure := event.(*UserReportEvent)
			if ure.Report.User.ID != "u-alice" {
				t.Errorf("expected only alice's report, got %s", ure.Report.User.ID)
			}
		}
	}

	if userReports != 1 {
		t.Errorf("expected 1 user report (self only), got %d", userReports)
	}
}

func TestReportService_ISPCheck_DiscovererCalled(t *testing.T) {
	userRepo := newMockUserRepo()
	teamRepo := newMockTeamRepo()
	repoStore := newMockRepoStore()
	registry := NewActivityFetcherRegistry()

	discoveredRepo := domain.Repository{
		ID:       "r-discovered",
		Name:     "discovered-repo",
		FullName: "org/discovered-repo",
		Platform: domain.PlatformGitHub,
	}

	fetcher := &mockFetcherWithDiscovery{
		mockActivityFetcher: mockActivityFetcher{
			platforms: []domain.GitPlatform{domain.PlatformGitHub},
			types:     []domain.ActivityType{domain.ActivityTypeCommit},
		},
		discoveredRepos: []domain.Repository{discoveredRepo},
	}
	registry.Register(domain.PlatformGitHub, &mockFetcherFactory{fetcher: fetcher})

	userRepo.users["u-alice"] = &domain.User{ID: "u-alice", Username: "alice"}
	teamRepo.teams["t-eng"] = &domain.Team{
		ID:            "t-eng",
		MemberIDs:     []string{"u-alice"},
		RepositoryIDs: []string{"r-1"},
	}
	repoStore.repos["r-1"] = &domain.Repository{
		ID: "r-1", FullName: "org/repo", Platform: domain.PlatformGitHub, APIToken: "key",
	}

	svc := NewReportService(userRepo, teamRepo, repoStore, registry)

	out := make(chan ReportEvent, 10)
	svc.GenerateReport(context.Background(), ReportQuery{
		TeamID:      "t-eng",
		CallerID:    "u-alice",
		CallerRoles: map[domain.Role]bool{domain.RoleTeamLeader: true},
		Since:       time.Now().Add(-24 * time.Hour),
		Until:       time.Now(),
	}, out)

	for range out {
	}

	if _, err := repoStore.FindByID(context.Background(), "r-discovered"); err != nil {
		t.Error("expected discovered repo to be upserted into store")
	}
}

func TestReportService_SkipsReposWithNoToken(t *testing.T) {
	userRepo := newMockUserRepo()
	teamRepo := newMockTeamRepo()
	repoStore := newMockRepoStore()
	registry := NewActivityFetcherRegistry()

	fetcher := &mockActivityFetcher{
		platforms:  []domain.GitPlatform{domain.PlatformGitHub},
		types:      []domain.ActivityType{domain.ActivityTypeCommit},
		activities: []domain.Activity{},
	}
	registry.Register(domain.PlatformGitHub, &mockFetcherFactory{fetcher: fetcher})

	userRepo.users["u-alice"] = &domain.User{ID: "u-alice", Username: "alice"}
	teamRepo.teams["t-eng"] = &domain.Team{
		ID:            "t-eng",
		MemberIDs:     []string{"u-alice"},
		RepositoryIDs: []string{"r-notoken"},
	}
	repoStore.repos["r-notoken"] = &domain.Repository{
		ID: "r-notoken", FullName: "org/notoken", Platform: domain.PlatformGitHub, APIToken: "",
	}

	svc := NewReportService(userRepo, teamRepo, repoStore, registry)

	out := make(chan ReportEvent, 10)
	svc.GenerateReport(context.Background(), ReportQuery{
		TeamID:      "t-eng",
		CallerID:    "u-alice",
		CallerRoles: map[domain.Role]bool{domain.RoleTeamMember: true},
		Since:       time.Now().Add(-24 * time.Hour),
		Until:       time.Now(),
	}, out)

	for event := range out {
		if event.GetType() == ReportEventTypeUserReport {
			ure := event.(*UserReportEvent)
			if len(ure.Report.Activities) != 0 {
				t.Errorf("expected no activities for tokenless repo, got %d", len(ure.Report.Activities))
			}
		}
	}
}
