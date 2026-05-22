package domain

type Team struct {
	ID            string
	Name          string
	LeaderIDs     []string
	MemberIDs     []string
	RepositoryIDs []string
}
