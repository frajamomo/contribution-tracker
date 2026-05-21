package domain

type User struct {
	ID                string
	Username          string
	DisplayName       string
	Email             string
	AvatarURL         string
	PlatformUsernames map[GitPlatform]string
}

func (u *User) GetPlatformUsername(platform GitPlatform) string {
	if u.PlatformUsernames != nil {
		if name, ok := u.PlatformUsernames[platform]; ok {
			return name
		}
	}
	return u.Username
}
