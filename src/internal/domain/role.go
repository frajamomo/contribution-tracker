package domain

type Role string

const (
	RoleAdmin      Role = "ADMIN"
	RoleTeamLeader Role = "TEAM_LEADER"
	RoleTeamMember Role = "TEAM_MEMBER"
)

var AllRoles = []Role{RoleAdmin, RoleTeamLeader, RoleTeamMember}
