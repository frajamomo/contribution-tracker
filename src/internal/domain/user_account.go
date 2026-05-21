package domain

type UserAccount struct {
	ID           string
	Username     string
	PasswordHash string
	Roles        map[Role]bool
	UserID       string
}

func (a *UserAccount) HasRole(r Role) bool {
	return a.Roles[r]
}

func (a *UserAccount) HasAnyRole(roles ...Role) bool {
	for _, r := range roles {
		if a.Roles[r] {
			return true
		}
	}
	return false
}

func (a *UserAccount) RoleList() []Role {
	var list []Role
	for r := range a.Roles {
		list = append(list, r)
	}
	return list
}
