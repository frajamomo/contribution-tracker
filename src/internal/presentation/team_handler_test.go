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

	handler := NewTeamHandler(repo, &mockRepoStore{}, &mockUserRepo{})
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

	handler := NewTeamHandler(repo, &mockRepoStore{}, &mockUserRepo{})
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
	var addedTeamID string
	var upsertedRepo *domain.Repository

	teamRepo := &mockTeamRepo{
		addRepoFn: func(ctx context.Context, teamID, repoID string) error {
			addedTeamID = teamID
			return nil
		},
	}
	repoStore := &mockRepoStore{
		upsertFn: func(ctx context.Context, repo *domain.Repository) (*domain.Repository, error) {
			upsertedRepo = repo
			return repo, nil
		},
	}

	handler := NewTeamHandler(teamRepo, repoStore, &mockUserRepo{})
	body := strings.NewReader(`{"fullName":"myorg/myrepo","platform":"github","apiToken":"ghp_test123"}`)

	r := chi.NewRouter()
	r.Post("/api/teams/{teamId}/repositories", handler.AddRepository)

	req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/repositories", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if addedTeamID != "t-1" {
		t.Errorf("expected team t-1, got %s", addedTeamID)
	}
	if upsertedRepo == nil {
		t.Fatal("expected repo to be upserted")
	}
	if upsertedRepo.FullName != "myorg/myrepo" {
		t.Errorf("expected fullName myorg/myrepo, got %s", upsertedRepo.FullName)
	}
	if upsertedRepo.Platform.Name != "GITHUB" {
		t.Errorf("expected platform GITHUB, got %s", upsertedRepo.Platform.Name)
	}
	if upsertedRepo.APIToken != "ghp_test123" {
		t.Errorf("expected token ghp_test123, got %s", upsertedRepo.APIToken)
	}
}

func TestAddRepository_MissingFields(t *testing.T) {
	handler := NewTeamHandler(&mockTeamRepo{
		findByIDFn: func(ctx context.Context, id string) (*domain.Team, error) {
			return &domain.Team{ID: id}, nil
		},
	}, &mockRepoStore{}, &mockUserRepo{})

	cases := []struct {
		name string
		body string
	}{
		{"missing fullName", `{"fullName":"","platform":"github","apiToken":"tok"}`},
		{"missing platform", `{"fullName":"org/repo","platform":"","apiToken":"tok"}`},
		{"missing apiToken", `{"fullName":"org/repo","platform":"github","apiToken":""}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/teams/{teamId}/repositories", handler.AddRepository)

			req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/repositories", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestAddRepository_ReusesExistingToken(t *testing.T) {
	var upsertedRepo *domain.Repository
	handler := NewTeamHandler(
		&mockTeamRepo{
			findByIDFn: func(ctx context.Context, id string) (*domain.Team, error) {
				return &domain.Team{ID: id, RepositoryIDs: []string{"r-existing"}}, nil
			},
			addRepoFn: func(ctx context.Context, teamID, repoID string) error { return nil },
		},
		&mockRepoStore{
			findByIDsFn: func(ctx context.Context, ids []string) ([]domain.Repository, error) {
				return []domain.Repository{
					{ID: "r-existing", Platform: domain.PlatformGitHub, APIToken: "ghp_reused"},
				}, nil
			},
			upsertFn: func(ctx context.Context, repo *domain.Repository) (*domain.Repository, error) {
				upsertedRepo = repo
				return repo, nil
			},
		},
		&mockUserRepo{},
	)

	r := chi.NewRouter()
	r.Post("/api/teams/{teamId}/repositories", handler.AddRepository)

	body := strings.NewReader(`{"fullName":"org/new-repo","platform":"github","apiToken":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/repositories", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if upsertedRepo.APIToken != "ghp_reused" {
		t.Errorf("expected reused token ghp_reused, got %s", upsertedRepo.APIToken)
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

	handler := NewTeamHandler(repo, &mockRepoStore{}, &mockUserRepo{})

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
