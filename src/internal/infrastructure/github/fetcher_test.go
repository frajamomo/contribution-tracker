package github

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
		if !strings.HasPrefix(r.URL.Path, "/repos/octocat/hello-world/commits") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if q.Get("author") != "octocat" {
			t.Errorf("expected author=octocat, got %q", q.Get("author"))
		}
		if !strings.HasPrefix(q.Get("since"), "2024-01-01") {
			t.Errorf("unexpected since: %s", q.Get("since"))
		}
		if !strings.HasPrefix(q.Get("until"), "2024-01-31") {
			t.Errorf("unexpected until: %s", q.Get("until"))
		}

		commits := []ghCommit{
			{
				SHA:     "abc123",
				HTMLURL: "https://github.com/octocat/hello-world/commit/abc123",
				Commit: ghCommitData{
					Message: "fix: resolve issue",
					Author:  ghCommitUser{Date: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
				},
			},
		}
		json.NewEncoder(w).Encode(commits)
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repo := domain.Repository{FullName: "octocat/hello-world", Platform: domain.PlatformGitHub}
	types := []domain.ActivityType{domain.ActivityTypeCommit}

	activities, err := fetcher.FetchForUser(context.Background(), "octocat", repo, since, until, types)
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
	if activity.GetSummary() != "fix: resolve issue" {
		t.Errorf("unexpected summary: %s", activity.GetSummary())
	}
	if activity.GetData().URL != "https://github.com/octocat/hello-world/commit/abc123" {
		t.Errorf("unexpected URL: %s", activity.GetData().URL)
	}
}

func TestSearchActivities_Issues(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")

		if strings.Contains(q, "type:issue") {
			result := ghSearchResult[ghIssue]{
				TotalCount: 2,
				Items: []ghIssue{
					{
						Title:      "Bug in login",
						State:      "open",
						HTMLURL:    "https://github.com/octocat/hello-world/issues/1",
						CreatedAt:  time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
						Repository: &ghSearchRepo{FullName: "octocat/hello-world"},
						Labels:     []ghLabel{{Name: "bug"}},
					},
					{
						Title:      "Issue in other repo",
						State:      "open",
						HTMLURL:    "https://github.com/other/repo/issues/5",
						CreatedAt:  time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
						Repository: &ghSearchRepo{FullName: "other/repo"},
					},
				},
			}
			json.NewEncoder(w).Encode(result)
		} else if strings.Contains(q, "type:pr") {
			result := ghSearchResult[ghIssue]{TotalCount: 0, Items: []ghIssue{}}
			json.NewEncoder(w).Encode(result)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos := []domain.Repository{
		{FullName: "octocat/hello-world", Platform: domain.PlatformGitHub},
	}
	types := []domain.ActivityType{domain.ActivityTypeIssue, domain.ActivityTypeMergedPullRequest}

	activities, err := fetcher.SearchActivities(context.Background(), "octocat", repos, since, until, types)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only include the issue from octocat/hello-world, not other/repo.
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity (filtered to team repos), got %d", len(activities))
	}
	if activities[0].GetType() != domain.ActivityTypeIssue {
		t.Errorf("expected ISSUE type, got %v", activities[0].GetType())
	}
	if activities[0].GetSummary() != "Bug in login" {
		t.Errorf("unexpected summary: %s", activities[0].GetSummary())
	}
}

func TestSearchActivities_MergedPRs(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	mergedAt := time.Date(2024, 1, 20, 14, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")

		if strings.Contains(q, "type:issue") {
			result := ghSearchResult[ghIssue]{TotalCount: 0, Items: []ghIssue{}}
			json.NewEncoder(w).Encode(result)
		} else if strings.Contains(q, "type:pr") {
			result := ghSearchResult[ghIssue]{
				TotalCount: 1,
				Items: []ghIssue{
					{
						Title:       "Add feature X",
						State:       "closed",
						HTMLURL:     "https://github.com/octocat/hello-world/pull/42",
						CreatedAt:   time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC),
						Repository:  &ghSearchRepo{FullName: "octocat/hello-world"},
						PullRequest: &ghPRRef{MergedAt: &mergedAt},
					},
				},
			}
			json.NewEncoder(w).Encode(result)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos := []domain.Repository{
		{FullName: "octocat/hello-world", Platform: domain.PlatformGitHub},
	}
	types := []domain.ActivityType{domain.ActivityTypeIssue, domain.ActivityTypeMergedPullRequest}

	activities, err := fetcher.SearchActivities(context.Background(), "octocat", repos, since, until, types)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].GetType() != domain.ActivityTypeMergedPullRequest {
		t.Errorf("expected MERGED_PR type, got %v", activities[0].GetType())
	}
	if activities[0].GetSummary() != "Add feature X" {
		t.Errorf("unexpected summary: %s", activities[0].GetSummary())
	}
}

func TestDiscoverUserRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/octocat/repos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("expected per_page=100, got %q", r.URL.Query().Get("per_page"))
		}

		repos := []ghRepository{
			{ID: 1, Name: "hello-world", FullName: "octocat/hello-world", HTMLURL: "https://github.com/octocat/hello-world"},
			{ID: 2, Name: "spoon-knife", FullName: "octocat/spoon-knife", HTMLURL: "https://github.com/octocat/spoon-knife"},
		}
		json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "token")
	fetcher := NewActivityFetcher(client)

	repos, err := fetcher.DiscoverUserRepos(context.Background(), "octocat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "octocat/hello-world" {
		t.Errorf("unexpected repo: %s", repos[0].FullName)
	}
	if repos[0].Platform != domain.PlatformGitHub {
		t.Errorf("expected GitHub platform, got %v", repos[0].Platform)
	}
}
