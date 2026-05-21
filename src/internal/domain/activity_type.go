package domain

type ActivityType struct {
	Name        string
	DisplayName string
}

var (
	ActivityTypeCommit            = ActivityType{Name: "COMMIT", DisplayName: "Commit"}
	ActivityTypeMergedPullRequest = ActivityType{Name: "MERGED_PR", DisplayName: "Merged PR"}
	ActivityTypeIssue             = ActivityType{Name: "ISSUE", DisplayName: "Issue"}
	ActivityTypePullRequestReview = ActivityType{Name: "PR_REVIEW", DisplayName: "PR Review"}
)
