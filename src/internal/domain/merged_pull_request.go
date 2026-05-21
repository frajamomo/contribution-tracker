package domain

import "time"

type MergedPullRequest struct {
	data     ActivityData
	Title    string
	MergedAt time.Time
}

func NewMergedPullRequest(data ActivityData, title string, mergedAt time.Time) *MergedPullRequest {
	return &MergedPullRequest{data: data, Title: title, MergedAt: mergedAt}
}

func (pr *MergedPullRequest) GetData() ActivityData { return pr.data }
func (pr *MergedPullRequest) GetType() ActivityType { return ActivityTypeMergedPullRequest }
func (pr *MergedPullRequest) GetSummary() string    { return pr.Title }
