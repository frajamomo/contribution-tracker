package domain

type Issue struct {
	data   ActivityData
	Title  string
	State  IssueState
	Labels []string
}

func NewIssue(data ActivityData, title string, state IssueState, labels []string) *Issue {
	return &Issue{data: data, Title: title, State: state, Labels: labels}
}

func (i *Issue) GetData() ActivityData { return i.data }
func (i *Issue) GetType() ActivityType { return ActivityTypeIssue }
func (i *Issue) GetSummary() string    { return i.Title }
