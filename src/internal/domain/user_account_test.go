package domain

import "testing"

func TestHasRole(t *testing.T) {
	a := UserAccount{Roles: map[Role]bool{RoleAdmin: true, RoleTeamMember: true}}

	if !a.HasRole(RoleAdmin) {
		t.Error("expected HasRole(ADMIN) to be true")
	}
	if !a.HasRole(RoleTeamMember) {
		t.Error("expected HasRole(TEAM_MEMBER) to be true")
	}
	if a.HasRole(RoleTeamLeader) {
		t.Error("expected HasRole(TEAM_LEADER) to be false")
	}
}

func TestHasAnyRole(t *testing.T) {
	a := UserAccount{Roles: map[Role]bool{RoleTeamMember: true}}

	if !a.HasAnyRole(RoleAdmin, RoleTeamMember) {
		t.Error("expected HasAnyRole(ADMIN, TEAM_MEMBER) to be true")
	}
	if a.HasAnyRole(RoleAdmin, RoleTeamLeader) {
		t.Error("expected HasAnyRole(ADMIN, TEAM_LEADER) to be false")
	}
}

func TestRoleList(t *testing.T) {
	a := UserAccount{Roles: map[Role]bool{RoleAdmin: true, RoleTeamLeader: true}}

	roles := a.RoleList()
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}
