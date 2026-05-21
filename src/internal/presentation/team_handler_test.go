package presentation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
)

func TestListTeams_Admin(t *testing.T) {
	repo := &mockTeamRepo{
		findAllFn: func(ctx context.Context) ([]domain.Team, error) {
			return []domain.Team{
				{ID: "t-1", Name: "Engineering"},
				{ID: "t-2", Name: "Design"},
			}, nil
		},
	}

	handler := NewTeamHandler(repo)
	req := newAuthenticatedRequest(http.MethodGet, "/api/teams", nil, &application.AuthContext{
		UserID: "u-admin",
		Roles:  map[domain.Role]bool{domain.RoleAdmin: true},
	})
	rr := httptest.NewRecorder()

	handler.ListTeams(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListTeams_Member(t *testing.T) {
	repo := &mockTeamRepo{
		findByMemberIDFn: func(ctx context.Context, memberID string) ([]domain.Team, error) {
			if memberID != "u-1" {
				t.Errorf("expected member ID u-1, got %s", memberID)
			}
			return []domain.Team{{ID: "t-1", Name: "Engineering"}}, nil
		},
	}

	handler := NewTeamHandler(repo)
	req := newAuthenticatedRequest(http.MethodGet, "/api/teams", nil, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})
	rr := httptest.NewRecorder()

	handler.ListTeams(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAddRepository_Success(t *testing.T) {
	var addedTeamID, addedRepoID string
	repo := &mockTeamRepo{
		addRepoFn: func(ctx context.Context, teamID, repoID string) error {
			addedTeamID = teamID
			addedRepoID = repoID
			return nil
		},
	}

	handler := NewTeamHandler(repo)
	body := strings.NewReader(`{"repoId":"r-1"}`)

	r := chi.NewRouter()
	r.Post("/api/teams/{teamId}/repositories", handler.AddRepository)

	req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/repositories", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if addedTeamID != "t-1" || addedRepoID != "r-1" {
		t.Errorf("expected t-1/r-1, got %s/%s", addedTeamID, addedRepoID)
	}
}

func TestAddRepository_MissingRepoID(t *testing.T) {
	handler := NewTeamHandler(&mockTeamRepo{})
	body := strings.NewReader(`{"repoId":""}`)

	r := chi.NewRouter()
	r.Post("/api/teams/{teamId}/repositories", handler.AddRepository)

	req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/repositories", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestRemoveRepository_Success(t *testing.T) {
	var removedTeamID, removedRepoID string
	repo := &mockTeamRepo{
		removeRepoFn: func(ctx context.Context, teamID, repoID string) error {
			removedTeamID = teamID
			removedRepoID = repoID
			return nil
		},
	}

	handler := NewTeamHandler(repo)

	r := chi.NewRouter()
	r.Delete("/api/teams/{teamId}/repositories/{repoId}", handler.RemoveRepository)

	req := httptest.NewRequest(http.MethodDelete, "/api/teams/t-1/repositories/r-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if removedTeamID != "t-1" || removedRepoID != "r-1" {
		t.Errorf("expected t-1/r-1, got %s/%s", removedTeamID, removedRepoID)
	}
}
