package presentation

import (
	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	Roles []string `json:"roles"`
	User  UserDTO  `json:"user"`
}

type ReportRequestDTO struct {
	TeamID     string   `json:"teamId"`
	Since      string   `json:"since"`
	Until      string   `json:"until"`
	Types      []string `json:"types"`
	ReportType string   `json:"reportType"`
}

type UserReportDTO struct {
	User       UserDTO           `json:"user"`
	Counts     []ActivityCountDTO `json:"counts"`
	Activities []ActivityDTO      `json:"activities"`
}

type UserDTO struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

type ActivityDTO struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	CreatedAt   string `json:"createdAt"`
	Summary     string `json:"summary"`
}

type ActivityCountDTO struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Count       int    `json:"count"`
}

type PlatformUsernameRequest struct {
	Platform string `json:"platform"`
	Username string `json:"username"`
}

type AddRepoRequest struct {
	RepoID string `json:"repoId"`
}

type ConfigSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SSEEventData struct {
	Type   string      `json:"type"`
	Report *UserReportDTO `json:"report,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func UserToDTO(u domain.User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

func ActivityToDTO(a domain.Activity) ActivityDTO {
	data := a.GetData()
	t := a.GetType()
	return ActivityDTO{
		Type:        t.Name,
		DisplayName: t.DisplayName,
		Title:       data.Title,
		URL:         data.URL,
		CreatedAt:   data.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Summary:     a.GetSummary(),
	}
}

func UserReportToDTO(r application.UserReport) UserReportDTO {
	counts := make([]ActivityCountDTO, len(r.Counts))
	for i, c := range r.Counts {
		counts[i] = ActivityCountDTO{
			Type:        c.Type.Name,
			DisplayName: c.Type.DisplayName,
			Count:       c.Count,
		}
	}

	activities := make([]ActivityDTO, len(r.Activities))
	for i, a := range r.Activities {
		activities[i] = ActivityToDTO(a)
	}

	return UserReportDTO{
		User:       UserToDTO(r.User),
		Counts:     counts,
		Activities: activities,
	}
}
