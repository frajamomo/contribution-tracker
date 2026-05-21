package domain

import "time"

type PullRequestReview struct {
	data             ActivityData
	PullRequestTitle string
	State            ReviewState
	SubmittedAt      time.Time
}

func NewPullRequestReview(data ActivityData, prTitle string, state ReviewState, submittedAt time.Time) *PullRequestReview {
	return &PullRequestReview{data: data, PullRequestTitle: prTitle, State: state, SubmittedAt: submittedAt}
}

func (r *PullRequestReview) GetData() ActivityData { return r.data }
func (r *PullRequestReview) GetType() ActivityType { return ActivityTypePullRequestReview }
func (r *PullRequestReview) GetSummary() string    { return r.PullRequestTitle }
