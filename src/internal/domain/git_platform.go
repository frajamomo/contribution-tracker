package domain

type GitPlatform struct {
	Name string
}

func (p GitPlatform) MarshalText() ([]byte, error) {
	return []byte(p.Name), nil
}

func (p *GitPlatform) UnmarshalText(text []byte) error {
	p.Name = string(text)
	return nil
}

var (
	PlatformGitHub = GitPlatform{Name: "GITHUB"}
	PlatformGitLab = GitPlatform{Name: "GITLAB"}
)
