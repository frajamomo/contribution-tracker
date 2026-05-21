package domain

type Team struct {
	ID            string
	Name          string
	MemberIDs     []string
	RepositoryIDs []string
}
