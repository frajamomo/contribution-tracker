package application

import (
	"time"

	"contribution-tracker/internal/domain"
)

type ReportQuery struct {
	TeamID      string
	MemberID    string
	CallerID    string
	CallerRoles map[domain.Role]bool
	Since       time.Time
	Until       time.Time
	Types       []domain.ActivityType
	ReportType  domain.ReportType
}

type ActivityCount struct {
	Type  domain.ActivityType
	Count int
}

type UserReport struct {
	User       domain.User
	Counts     []ActivityCount
	Activities []domain.Activity
}

type AuthToken struct {
	Value     string
	ExpiresAt time.Time
	AccountID string
	Roles     map[domain.Role]bool
}

type AuthContext struct {
	AccountID string
	UserID    string
	Roles     map[domain.Role]bool
}

func (ac *AuthContext) IsAdmin() bool      { return ac.Roles[domain.RoleAdmin] }
func (ac *AuthContext) IsTeamLeader() bool  { return ac.Roles[domain.RoleTeamLeader] }
func (ac *AuthContext) IsTeamMember() bool  { return ac.Roles[domain.RoleTeamMember] }
