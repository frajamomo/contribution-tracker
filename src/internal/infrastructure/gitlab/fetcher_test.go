package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"contribution-tracker/internal/domain"
)

func TestFetchForUser_Commits(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitLab uses URL-encoded project path: group%2Fproject
		if !strings.Contains(r.URL.Path, "/repository/commits") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if q.Get("author") != "john" {
			t.Errorf("expected author=john, got %q", q.Get("author"))
		}
		if !strings.HasPrefix(q.Get("since"), "2024-01-01") {
			t.Errorf("unexpected since: %s", q.Get("since"))
		}
		if !strings.HasPrefix(q.Get("until"), "2024-01-31") {
			t.Errorf("unexpected until: %s", q.Get("until"))
		}

		commits := []glCommit{
			{
				ID:           "def456",
				ShortID:      "def456",
				Title:        "fix: resolve gitlab issue",
				Message:      "fix: resolve gitlab issue",
				AuthoredDate: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				WebURL:       "https://gitlab.com/group/project/-/commit/def456",
			},
		}
		json.NewEncoder(w).Encode(commits)
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repo := domain.Repository{FullName: "group/project", Platform: domain.PlatformGitLab}
	types := []domain.ActivityType{domain.ActivityTypeCommit}

	activities, err := fetcher.FetchForUser(context.Background(), "john", repo, since, until, types)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.GetType() != domain.ActivityTypeCommit {
		t.Errorf("expected COMMIT type, got %v", activity.GetType())
	}
	if activity.GetSummary() != "fix: resolve gitlab issue" {
		t.Errorf("unexpected summary: %s", activity.GetSummary())
	}
}

func TestSearchActivities_Issues(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v4/issues") {
			q := r.URL.Query()
			if q.Get("author_username") != "john" {
				t.Errorf("expected author_username=john, got %q", q.Get("author_username"))
			}

			issues := []glIssue{
				{
					ID:        1,
					IID:       10,
					Title:     "Bug in authentication",
					State:     "opened",
					WebURL:    "https://gitlab.com/group/project/-/issues/10",
					CreatedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
					Labels:    []string{"bug"},
				},
				{
					ID:        2,
					IID:       20,
					Title:     "Issue in other project",
					State:     "opened",
					WebURL:    "https://gitlab.com/other/project/-/issues/20",
					CreatedAt: time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
				},
			}
			json.NewEncoder(w).Encode(issues)
		} else if strings.HasPrefix(r.URL.Path, "/api/v4/merge_requests") {
			json.NewEncoder(w).Encode([]glMergeRequest{})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos := []domain.Repository{
		{FullName: "group/project", Platform: domain.PlatformGitLab},
	}
	types := []domain.ActivityType{domain.ActivityTypeIssue, domain.ActivityTypeMergedPullRequest}

	activities, err := fetcher.SearchActivities(context.Background(), "john", repos, since, until, types)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only include the issue from group/project, not other/project.
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity (filtered to team repos), got %d", len(activities))
	}
	if activities[0].GetType() != domain.ActivityTypeIssue {
		t.Errorf("expected ISSUE type, got %v", activities[0].GetType())
	}
	if activities[0].GetSummary() != "Bug in authentication" {
		t.Errorf("unexpected summary: %s", activities[0].GetSummary())
	}
}

func TestSearchActivities_MergedMRs(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	mergedAt := time.Date(2024, 1, 20, 14, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v4/issues") {
			json.NewEncoder(w).Encode([]glIssue{})
		} else if strings.HasPrefix(r.URL.Path, "/api/v4/merge_requests") {
			mrs := []glMergeRequest{
				{
					ID:        100,
					IID:       5,
					Title:     "Add feature Y",
					State:     "merged",
					WebURL:    "https://gitlab.com/group/project/-/merge_requests/5",
					CreatedAt: time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC),
					MergedAt:  &mergedAt,
				},
			}
			json.NewEncoder(w).Encode(mrs)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos := []domain.Repository{
		{FullName: "group/project", Platform: domain.PlatformGitLab},
	}
	types := []domain.ActivityType{domain.ActivityTypeIssue, domain.ActivityTypeMergedPullRequest}

	activities, err := fetcher.SearchActivities(context.Background(), "john", repos, since, until, types)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].GetType() != domain.ActivityTypeMergedPullRequest {
		t.Errorf("expected MERGED_PR type, got %v", activities[0].GetType())
	}
	if activities[0].GetSummary() != "Add feature Y" {
		t.Errorf("unexpected summary: %s", activities[0].GetSummary())
	}
}

func TestDiscoverUserRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("membership") != "true" {
			t.Errorf("expected membership=true, got %q", r.URL.Query().Get("membership"))
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("expected per_page=100, got %q", r.URL.Query().Get("per_page"))
		}

		projects := []glProject{
			{ID: 1, Name: "project", PathWithNamespace: "group/project", WebURL: "https://gitlab.com/group/project"},
			{ID: 2, Name: "another", PathWithNamespace: "group/another", WebURL: "https://gitlab.com/group/another"},
		}
		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos, err := fetcher.DiscoverUserRepos(context.Background(), "john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "group/project" {
		t.Errorf("unexpected repo: %s", repos[0].FullName)
	}
	if repos[0].Platform != domain.PlatformGitLab {
		t.Errorf("expected GitLab platform, got %v", repos[0].Platform)
	}
}

func TestExtractProjectPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://gitlab.com/group/project/-/issues/10", "group/project"},
		{"https://gitlab.com/group/subgroup/project/-/merge_requests/5", "group/subgroup/project"},
		{"https://gitlab.com/group/project/-/commit/abc123", "group/project"},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		got := extractProjectPath(tt.url)
		if got != tt.want {
			t.Errorf("extractProjectPath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
