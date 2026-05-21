package domain

type Commit struct {
	data    ActivityData
	SHA     string
	Message string
}

func NewCommit(data ActivityData, sha, message string) *Commit {
	return &Commit{data: data, SHA: sha, Message: message}
}

func (c *Commit) GetData() ActivityData { return c.data }
func (c *Commit) GetType() ActivityType { return ActivityTypeCommit }
func (c *Commit) GetSummary() string    { return c.Message }
