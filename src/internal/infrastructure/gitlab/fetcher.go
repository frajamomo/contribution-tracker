package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

// Compile-time interface checks.
var (
	_ application.ActivityFetcher = (*ActivityFetcher)(nil)
	_ application.RepoDiscoverer = (*ActivityFetcher)(nil)
)

// ActivityFetcher fetches contribution activity from the GitLab API.
type ActivityFetcher struct {
	client *ApiClient
}

// NewActivityFetcher creates a new GitLab activity fetcher.
func NewActivityFetcher(client *ApiClient) *ActivityFetcher {
	return &ActivityFetcher{client: client}
}

func (f *ActivityFetcher) GetSupportedPlatforms() []domain.GitPlatform {
	return []domain.GitPlatform{domain.PlatformGitLab}
}

func (f *ActivityFetcher) GetSupportedTypes() []domain.ActivityType {
	return []domain.ActivityType{
		domain.ActivityTypeCommit,
		domain.ActivityTypeMergedPullRequest,
		domain.ActivityTypeIssue,
		domain.ActivityTypePullRequestReview,
	}
}

func (f *ActivityFetcher) FetchForUser(
	ctx context.Context,
	username string,
	repo domain.Repository,
	since, until time.Time,
	types []domain.ActivityType,
) ([]domain.Activity, error) {
	var activities []domain.Activity

	typeSet := makeTypeSet(types)

	if _, ok := typeSet[domain.ActivityTypeCommit.Name]; ok {
		commits, err := f.fetchCommits(ctx, username, repo, since, until)
		if err != nil {
			return nil, fmt.Errorf("fetching commits: %w", err)
		}
		activities = append(activities, commits...)
	}

	return activities, nil
}

func (f *ActivityFetcher) SearchActivities(
	ctx context.Context,
	username string,
	repos []domain.Repository,
	since, until time.Time,
	types []domain.ActivityType,
) ([]domain.Activity, error) {
	var activities []domain.Activity

	repoSet := makeRepoSet(repos)
	typeSet := makeTypeSet(types)

	if _, ok := typeSet[domain.ActivityTypeIssue.Name]; ok {
		issues, err := f.searchIssues(ctx, username, since, until, repoSet)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}
		activities = append(activities, issues...)
	}

	if _, ok := typeSet[domain.ActivityTypeMergedPullRequest.Name]; ok {
		mrs, err := f.searchMergedMRs(ctx, username, since, until, repoSet)
		if err != nil {
			return nil, fmt.Errorf("searching merged MRs: %w", err)
		}
		activities = append(activities, mrs...)
	}

	if _, ok := typeSet[domain.ActivityTypePullRequestReview.Name]; ok {
		reviews, err := f.searchMRDiscussions(ctx, username, repos, since, until)
		if err != nil {
			return nil, fmt.Errorf("searching MR discussions: %w", err)
		}
		activities = append(activities, reviews...)
	}

	return activities, nil
}

func (f *ActivityFetcher) DiscoverUserRepos(ctx context.Context, username string) ([]domain.Repository, error) {
	path := "/api/v4/projects?membership=true&per_page=100"

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetching user projects: %w", err)
	}

	var repos []domain.Repository
	for _, page := range pages {
		var projects []glProject
		if err := json.Unmarshal(page, &projects); err != nil {
			return nil, fmt.Errorf("parsing projects response: %w", err)
		}
		for _, p := range projects {
			repos = append(repos, domain.Repository{
				ID:       strconv.Itoa(p.ID),
				Name:     p.Name,
				FullName: p.PathWithNamespace,
				URL:      p.WebURL,
				Platform: domain.PlatformGitLab,
			})
		}
	}

	return repos, nil
}

func (f *ActivityFetcher) fetchCommits(
	ctx context.Context,
	username string,
	repo domain.Repository,
	since, until time.Time,
) ([]domain.Activity, error) {
	params := url.Values{}
	params.Set("author", username)
	params.Set("since", since.Format(time.RFC3339))
	params.Set("until", until.Format(time.RFC3339))

	projectID := url.PathEscape(repo.FullName)
	path := fmt.Sprintf("/api/v4/projects/%s/repository/commits?%s", projectID, params.Encode())

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, err
	}

	var activities []domain.Activity
	for _, page := range pages {
		var commits []glCommit
		if err := json.Unmarshal(page, &commits); err != nil {
			return nil, fmt.Errorf("parsing commits response: %w", err)
		}
		for _, c := range commits {
			data := domain.ActivityData{
				Title:     c.Title,
				URL:       c.WebURL,
				CreatedAt: c.AuthoredDate,
			}
			activities = append(activities, domain.NewCommit(data, c.ID, c.Message))
		}
	}

	return activities, nil
}

func (f *ActivityFetcher) searchIssues(
	ctx context.Context,
	username string,
	since, until time.Time,
	repoSet map[string]bool,
) ([]domain.Activity, error) {
	params := url.Values{}
	params.Set("author_username", username)
	params.Set("created_after", since.Format(time.RFC3339))
	params.Set("created_before", until.Format(time.RFC3339))
	params.Set("per_page", "100")

	path := "/api/v4/issues?" + params.Encode()

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, err
	}

	var activities []domain.Activity
	for _, page := range pages {
		var issues []glIssue
		if err := json.Unmarshal(page, &issues); err != nil {
			return nil, fmt.Errorf("parsing issues response: %w", err)
		}
		for _, issue := range issues {
			// GitLab issues endpoint doesn't include project full path directly,
			// so we extract it from the web_url.
			repoName := extractProjectPath(issue.WebURL)
			if !repoSet[repoName] {
				continue
			}

			data := domain.ActivityData{
				Title:     issue.Title,
				URL:       issue.WebURL,
				CreatedAt: issue.CreatedAt,
			}
			state := domain.IssueStateOpen
			if issue.State == "closed" {
				state = domain.IssueStateClosed
			}
			activities = append(activities, domain.NewIssue(data, issue.Title, state, issue.Labels))
		}
	}

	return activities, nil
}

func (f *ActivityFetcher) searchMergedMRs(
	ctx context.Context,
	username string,
	since, until time.Time,
	repoSet map[string]bool,
) ([]domain.Activity, error) {
	params := url.Values{}
	params.Set("author_username", username)
	params.Set("state", "merged")
	params.Set("created_after", since.Format(time.RFC3339))
	params.Set("created_before", until.Format(time.RFC3339))
	params.Set("per_page", "100")

	path := "/api/v4/merge_requests?" + params.Encode()

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, err
	}

	var activities []domain.Activity
	for _, page := range pages {
		var mrs []glMergeRequest
		if err := json.Unmarshal(page, &mrs); err != nil {
			return nil, fmt.Errorf("parsing merge requests response: %w", err)
		}
		for _, mr := range mrs {
			repoName := extractProjectPath(mr.WebURL)
			if !repoSet[repoName] {
				continue
			}

			mergedAt := mr.CreatedAt
			if mr.MergedAt != nil {
				mergedAt = *mr.MergedAt
			}

			data := domain.ActivityData{
				Title:     mr.Title,
				URL:       mr.WebURL,
				CreatedAt: mr.CreatedAt,
			}
			activities = append(activities, domain.NewMergedPullRequest(data, mr.Title, mergedAt))
		}
	}

	return activities, nil
}

func (f *ActivityFetcher) searchMRDiscussions(
	ctx context.Context,
	username string,
	repos []domain.Repository,
	since, until time.Time,
) ([]domain.Activity, error) {
	// For each repo, find MRs updated in the period, then check for notes by the user.
	var activities []domain.Activity

	for _, repo := range repos {
		params := url.Values{}
		params.Set("state", "all")
		params.Set("updated_after", since.Format(time.RFC3339))
		params.Set("updated_before", until.Format(time.RFC3339))
		params.Set("per_page", "100")

		projectID := url.PathEscape(repo.FullName)
		path := fmt.Sprintf("/api/v4/projects/%s/merge_requests?%s", projectID, params.Encode())

		mrBody, err := f.client.Get(ctx, path)
		if err != nil {
			return nil, err
		}

		var mrs []glMergeRequest
		if err := json.Unmarshal(mrBody, &mrs); err != nil {
			return nil, fmt.Errorf("parsing merge requests response: %w", err)
		}

		for _, mr := range mrs {
			notesPath := fmt.Sprintf("/api/v4/projects/%s/merge_requests/%d/notes?per_page=100",
				projectID, mr.IID)

			notesBody, err := f.client.Get(ctx, notesPath)
			if err != nil {
				return nil, err
			}

			var notes []glNote
			if err := json.Unmarshal(notesBody, &notes); err != nil {
				return nil, fmt.Errorf("parsing notes response: %w", err)
			}

			for _, note := range notes {
				if note.System || note.Author.Username != username {
					continue
				}
				if note.CreatedAt.Before(since) || note.CreatedAt.After(until) {
					continue
				}

				data := domain.ActivityData{
					Title:     mr.Title,
					URL:       mr.WebURL,
					CreatedAt: note.CreatedAt,
				}
				activities = append(activities, domain.NewPullRequestReview(
					data, mr.Title, domain.ReviewStateCommented, note.CreatedAt))
			}
		}
	}

	return activities, nil
}

// extractProjectPath extracts the project path from a GitLab web URL.
// For example: "https://gitlab.com/group/project/-/issues/1" -> "group/project"
func extractProjectPath(webURL string) string {
	u, err := url.Parse(webURL)
	if err != nil {
		return ""
	}
	path := u.Path
	// Remove leading slash.
	path = path[1:]
	// Find "/-/" separator and take everything before it.
	idx := 0
	for i := 0; i < len(path); i++ {
		if i+3 <= len(path) && path[i:i+3] == "/-/" {
			idx = i
			break
		}
	}
	if idx > 0 {
		return path[:idx]
	}
	return ""
}

func makeTypeSet(types []domain.ActivityType) map[string]bool {
	set := make(map[string]bool, len(types))
	for _, t := range types {
		set[t.Name] = true
	}
	return set
}

func makeRepoSet(repos []domain.Repository) map[string]bool {
	set := make(map[string]bool, len(repos))
	for _, r := range repos {
		set[r.FullName] = true
	}
	return set
}
