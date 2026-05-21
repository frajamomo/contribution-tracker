package domain

import (
	"testing"
	"time"
)

func TestCommit_ImplementsActivity(t *testing.T) {
	now := time.Now()
	data := ActivityData{Title: "fix bug", URL: "https://example.com/1", CreatedAt: now}
	c := NewCommit(data, "abc123", "fix: resolve nil pointer")

	var a Activity = c
	if a.GetType() != ActivityTypeCommit {
		t.Errorf("expected COMMIT type, got %v", a.GetType())
	}
	if a.GetSummary() != "fix: resolve nil pointer" {
		t.Errorf("unexpected summary: %s", a.GetSummary())
	}
	if a.GetData().Title != "fix bug" {
		t.Errorf("unexpected title: %s", a.GetData().Title)
	}
	if a.GetData().CreatedAt != now {
		t.Error("timestamp mismatch")
	}
}

func TestMergedPullRequest_ImplementsActivity(t *testing.T) {
	now := time.Now()
	data := ActivityData{Title: "Add feature", URL: "https://example.com/pr/1", CreatedAt: now}
	pr := NewMergedPullRequest(data, "Add login feature", now.Add(time.Hour))

	var a Activity = pr
	if a.GetType() != ActivityTypeMergedPullRequest {
		t.Errorf("expected MERGED_PR type, got %v", a.GetType())
	}
	if a.GetSummary() != "Add login feature" {
		t.Errorf("unexpected summary: %s", a.GetSummary())
	}
}

func TestIssue_ImplementsActivity(t *testing.T) {
	data := ActivityData{Title: "Bug report", URL: "https://example.com/issues/1", CreatedAt: time.Now()}
	issue := NewIssue(data, "Login broken", IssueStateOpen, []string{"bug", "urgent"})

	var a Activity = issue
	if a.GetType() != ActivityTypeIssue {
		t.Errorf("expected ISSUE type, got %v", a.GetType())
	}
	if a.GetSummary() != "Login broken" {
		t.Errorf("unexpected summary: %s", a.GetSummary())
	}
	if issue.State != IssueStateOpen {
		t.Errorf("expected OPEN state, got %s", issue.State)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}
}

func TestPullRequestReview_ImplementsActivity(t *testing.T) {
	now := time.Now()
	data := ActivityData{Title: "Review", URL: "https://example.com/pr/1/review", CreatedAt: now}
	review := NewPullRequestReview(data, "Add login feature", ReviewStateApproved, now)

	var a Activity = review
	if a.GetType() != ActivityTypePullRequestReview {
		t.Errorf("expected PR_REVIEW type, got %v", a.GetType())
	}
	if a.GetSummary() != "Add login feature" {
		t.Errorf("unexpected summary: %s", a.GetSummary())
	}
	if review.State != ReviewStateApproved {
		t.Errorf("expected APPROVED state, got %s", review.State)
	}
}
