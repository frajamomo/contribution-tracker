package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

// Compile-time interface checks.
var (
	_ application.ActivityFetcher = (*ActivityFetcher)(nil)
	_ application.RepoDiscoverer = (*ActivityFetcher)(nil)
)

// ActivityFetcher fetches contribution activity from the GitHub API.
type ActivityFetcher struct {
	client *ApiClient
}

// NewActivityFetcher creates a new GitHub activity fetcher.
func NewActivityFetcher(client *ApiClient) *ActivityFetcher {
	return &ActivityFetcher{client: client}
}

func (f *ActivityFetcher) GetSupportedPlatforms() []domain.GitPlatform {
	return []domain.GitPlatform{domain.PlatformGitHub}
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

	if _, ok := typeSet[domain.ActivityTypePullRequestReview.Name]; ok {
		reviews, err := f.fetchReviews(ctx, username, repo, since, until)
		if err != nil {
			return nil, fmt.Errorf("fetching reviews: %w", err)
		}
		activities = append(activities, reviews...)
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
		prs, err := f.searchMergedPRs(ctx, username, since, until, repoSet)
		if err != nil {
			return nil, fmt.Errorf("searching merged PRs: %w", err)
		}
		activities = append(activities, prs...)
	}

	return activities, nil
}

func (f *ActivityFetcher) DiscoverUserRepos(ctx context.Context, username string) ([]domain.Repository, error) {
	path := fmt.Sprintf("/users/%s/repos?per_page=100", url.PathEscape(username))

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetching user repos: %w", err)
	}

	var repos []domain.Repository
	for _, page := range pages {
		var ghRepos []ghRepository
		if err := json.Unmarshal(page, &ghRepos); err != nil {
			return nil, fmt.Errorf("parsing repos response: %w", err)
		}
		for _, r := range ghRepos {
			repos = append(repos, domain.Repository{
				ID:       strconv.FormatInt(r.ID, 10),
				Name:     r.Name,
				FullName: r.FullName,
				URL:      r.HTMLURL,
				Platform: domain.PlatformGitHub,
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

	path := fmt.Sprintf("/repos/%s/commits?%s", repo.FullName, params.Encode())

	pages, err := f.client.GetAll(ctx, path)
	if err != nil {
		return nil, err
	}

	var activities []domain.Activity
	for _, page := range pages {
		var commits []ghCommit
		if err := json.Unmarshal(page, &commits); err != nil {
			return nil, fmt.Errorf("parsing commits response: %w", err)
		}
		for _, c := range commits {
			data := domain.ActivityData{
				Title:     c.Commit.Message,
				URL:       c.HTMLURL,
				CreatedAt: c.Commit.Author.Date,
			}
			activities = append(activities, domain.NewCommit(data, c.SHA, c.Commit.Message))
		}
	}

	return activities, nil
}

func (f *ActivityFetcher) fetchReviews(
	ctx context.Context,
	username string,
	repo domain.Repository,
	since, until time.Time,
) ([]domain.Activity, error) {
	// GitHub doesn't have a direct "reviews by user" endpoint, so we search for
	// reviewed PRs first, then fetch reviews for each.
	q := fmt.Sprintf("reviewed-by:%s repo:%s created:%s..%s",
		username, repo.FullName,
		since.Format("2006-01-02"), until.Format("2006-01-02"))

	params := url.Values{}
	params.Set("q", q)
	path := "/search/issues?" + params.Encode()

	body, err := f.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result ghSearchResult[ghIssue]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	var activities []domain.Activity
	for _, pr := range result.Items {
		reviewsPath := fmt.Sprintf("/repos/%s/pulls/%d/reviews", repo.FullName, pr.Number)
		reviewBody, err := f.client.Get(ctx, reviewsPath)
		if err != nil {
			return nil, err
		}

		var reviews []ghReview
		if err := json.Unmarshal(reviewBody, &reviews); err != nil {
			return nil, fmt.Errorf("parsing reviews response: %w", err)
		}

		for _, r := range reviews {
			if r.User.Login != username {
				continue
			}
			if r.SubmittedAt.Before(since) || r.SubmittedAt.After(until) {
				continue
			}

			data := domain.ActivityData{
				Title:     pr.Title,
				URL:       r.HTMLURL,
				CreatedAt: r.SubmittedAt,
			}
			state := mapReviewState(r.State)
			activities = append(activities, domain.NewPullRequestReview(data, pr.Title, state, r.SubmittedAt))
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
	var repoFilters []string
	for name := range repoSet {
		repoFilters = append(repoFilters, "repo:"+name)
	}

	var allItems []ghIssue
	for _, rf := range repoFilters {
		q := fmt.Sprintf("author:%s type:issue %s created:%s..%s",
			username, rf,
			since.Format("2006-01-02"), until.Format("2006-01-02"))

		params := url.Values{}
		params.Set("q", q)
		path := "/search/issues?" + params.Encode()

		body, err := f.client.Get(ctx, path)
		if err != nil {
			return nil, err
		}

		var result ghSearchResult[ghIssue]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parsing search response: %w", err)
		}
		allItems = append(allItems, result.Items...)
	}

	var activities []domain.Activity
	for _, issue := range allItems {

		var labels []string
		for _, l := range issue.Labels {
			labels = append(labels, l.Name)
		}

		data := domain.ActivityData{
			Title:     issue.Title,
			URL:       issue.HTMLURL,
			CreatedAt: issue.CreatedAt,
		}
		state := domain.IssueStateOpen
		if issue.State == "closed" {
			state = domain.IssueStateClosed
		}
		activities = append(activities, domain.NewIssue(data, issue.Title, state, labels))
	}

	return activities, nil
}

func (f *ActivityFetcher) searchMergedPRs(
	ctx context.Context,
	username string,
	since, until time.Time,
	repoSet map[string]bool,
) ([]domain.Activity, error) {
	var repoFilters []string
	for name := range repoSet {
		repoFilters = append(repoFilters, "repo:"+name)
	}

	var allItems []ghIssue
	for _, rf := range repoFilters {
		q := fmt.Sprintf("author:%s type:pr is:merged %s created:%s..%s",
			username, rf,
			since.Format("2006-01-02"), until.Format("2006-01-02"))

		params := url.Values{}
		params.Set("q", q)
		path := "/search/issues?" + params.Encode()

		body, err := f.client.Get(ctx, path)
		if err != nil {
			return nil, err
		}

		var result ghSearchResult[ghIssue]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parsing search response: %w", err)
		}
		allItems = append(allItems, result.Items...)
	}

	var activities []domain.Activity
	for _, item := range allItems {

		mergedAt := item.CreatedAt
		if item.PullRequest != nil && item.PullRequest.MergedAt != nil {
			mergedAt = *item.PullRequest.MergedAt
		}

		data := domain.ActivityData{
			Title:     item.Title,
			URL:       item.HTMLURL,
			CreatedAt: item.CreatedAt,
		}
		activities = append(activities, domain.NewMergedPullRequest(data, item.Title, mergedAt))
	}

	return activities, nil
}

func repoFromURL(htmlURL string) string {
	// https://github.com/owner/repo/issues/123 → owner/repo
	const prefix = "https://github.com/"
	if !strings.HasPrefix(htmlURL, prefix) {
		return ""
	}
	parts := strings.SplitN(htmlURL[len(prefix):], "/", 4)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
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

func mapReviewState(state string) domain.ReviewState {
	switch state {
	case "APPROVED":
		return domain.ReviewStateApproved
	case "CHANGES_REQUESTED":
		return domain.ReviewStateChangesRequested
	default:
		return domain.ReviewStateCommented
	}
}
