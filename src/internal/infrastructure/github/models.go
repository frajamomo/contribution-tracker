package github

import "time"

// ghCommit represents a commit returned by the GitHub API.
type ghCommit struct {
	SHA    string       `json:"sha"`
	Commit ghCommitData `json:"commit"`
	HTMLURL string     `json:"html_url"`
}

type ghCommitData struct {
	Message string       `json:"message"`
	Author  ghCommitUser `json:"author"`
}

type ghCommitUser struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// ghIssue represents an issue returned by the GitHub API (also used in search results).
type ghIssue struct {
	ID        int64     `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	Labels    []ghLabel `json:"labels"`
	Repository *ghSearchRepo `json:"repository,omitempty"`
	PullRequest *ghPRRef   `json:"pull_request,omitempty"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghSearchRepo struct {
	FullName string `json:"full_name"`
}

type ghPRRef struct {
	MergedAt *time.Time `json:"merged_at,omitempty"`
	HTMLURL  string     `json:"html_url"`
}

// ghPullRequest represents a pull request returned by the GitHub search API.
type ghPullRequest struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	Repository *ghSearchRepo `json:"repository,omitempty"`
	PullRequest *ghPRRef  `json:"pull_request,omitempty"`
}

// ghReview represents a pull request review returned by the GitHub API.
type ghReview struct {
	ID          int64     `json:"id"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	SubmittedAt time.Time `json:"submitted_at"`
	User        ghUser    `json:"user"`
}

type ghUser struct {
	Login string `json:"login"`
}

// ghRepository represents a repository returned by the GitHub API.
type ghRepository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// ghSearchResult is a generic wrapper for GitHub search API responses.
type ghSearchResult[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}
