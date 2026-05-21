package domain

type Repository struct {
	ID       string
	Name     string
	FullName string
	URL      string
	Platform GitPlatform
}
