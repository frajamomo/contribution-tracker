package gitlab

import "time"

// glCommit represents a commit returned by the GitLab API.
type glCommit struct {
	ID             string    `json:"id"`
	ShortID        string    `json:"short_id"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	AuthorName     string    `json:"author_name"`
	AuthorEmail    string    `json:"author_email"`
	AuthoredDate   time.Time `json:"authored_date"`
	CommittedDate  time.Time `json:"committed_date"`
	WebURL         string    `json:"web_url"`
}

// glIssue represents an issue returned by the GitLab API.
type glIssue struct {
	ID        int       `json:"id"`
	IID       int       `json:"iid"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	WebURL    string    `json:"web_url"`
	CreatedAt time.Time `json:"created_at"`
	Labels    []string  `json:"labels"`
}

// glMergeRequest represents a merge request returned by the GitLab API.
type glMergeRequest struct {
	ID        int        `json:"id"`
	IID       int        `json:"iid"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	WebURL    string     `json:"web_url"`
	CreatedAt time.Time  `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
}

// glProject represents a project returned by the GitLab API.
type glProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

// glNote represents a discussion note (used for MR reviews) returned by the GitLab API.
type glNote struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Author    glAuthor  `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
	Noteable  string    `json:"noteable_type"`
}

type glAuthor struct {
	Username string `json:"username"`
}
