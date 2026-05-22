package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
)

func newAdminHandler(userRepo *mockUserRepo, accountRepo *mockUserAccountRepo, teamRepo *mockTeamRepo) *AdminHandler {
	return NewAdminHandler(userRepo, accountRepo, teamRepo)
}

func TestListUsers(t *testing.T) {
	userRepo := &mockUserRepo{
		findAllFn: func(ctx context.Context) ([]domain.User, error) {
			return []domain.User{
				{ID: "u-1", Username: "alice", DisplayName: "Alice"},
				{ID: "u-2", Username: "bob", DisplayName: "Bob"},
			}, nil
		},
	}
	accountRepo := &mockUserAccountRepo{
		findAllFn: func(ctx context.Context) ([]domain.UserAccount, error) {
			return []domain.UserAccount{
				{ID: "a-1", UserID: "u-1", Roles: map[domain.Role]bool{domain.RoleAdmin: true}},
				{ID: "a-2", UserID: "u-2", Roles: map[domain.Role]bool{domain.RoleTeamMember: true}},
			}, nil
		},
	}
	teamRepo := &mockTeamRepo{
		findAllFn: func(ctx context.Context) ([]domain.Team, error) {
			return []domain.Team{
				{ID: "t-1", Name: "Backend", MemberIDs: []string{"u-1", "u-2"}},
			}, nil
		},
	}

	h := newAdminHandler(userRepo, accountRepo, teamRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result []AdminUserDTO
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 2 {
		t.Fatalf("expected 2 users, got %d", len(result))
	}
	if result[0].Username != "alice" {
		t.Errorf("expected alice, got %s", result[0].Username)
	}
	if len(result[0].Teams) != 1 || result[0].Teams[0].Name != "Backend" {
		t.Errorf("expected alice in Backend team, got %v", result[0].Teams)
	}
	if len(result[0].Roles) != 1 || result[0].Roles[0] != string(domain.RoleAdmin) {
		t.Errorf("expected admin role, got %v", result[0].Roles)
	}
}

func TestCreateUser(t *testing.T) {
	var savedUser *domain.User
	var savedAccount *domain.UserAccount

	userRepo := &mockUserRepo{
		saveFn: func(ctx context.Context, user *domain.User) error {
			savedUser = user
			return nil
		},
	}
	accountRepo := &mockUserAccountRepo{
		saveFn: func(ctx context.Context, account *domain.UserAccount) error {
			savedAccount = account
			return nil
		},
	}
	teamRepo := &mockTeamRepo{}

	h := newAdminHandler(userRepo, accountRepo, teamRepo)
	body := `{"username":"dave","displayName":"Dave D","email":"dave@example.com","password":"secret123","roles":["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if savedUser == nil {
		t.Fatal("user was not saved")
	}
	if savedUser.Username != "dave" {
		t.Errorf("expected username dave, got %s", savedUser.Username)
	}
	if savedAccount == nil {
		t.Fatal("account was not saved")
	}
	if savedAccount.PasswordHash == "" || savedAccount.PasswordHash == "secret123" {
		t.Error("password should be hashed")
	}

	var result AdminUserDTO
	json.NewDecoder(w.Body).Decode(&result)
	if result.Username != "dave" {
		t.Errorf("expected dave in response, got %s", result.Username)
	}
}

func TestCreateUserMissingFields(t *testing.T) {
	h := newAdminHandler(&mockUserRepo{}, &mockUserAccountRepo{}, &mockTeamRepo{})

	body := `{"username":"","password":"","displayName":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	var deletedAccountUserID, deletedUserID string

	userRepo := &mockUserRepo{
		deleteFn: func(ctx context.Context, id string) error {
			deletedUserID = id
			return nil
		},
	}
	accountRepo := &mockUserAccountRepo{
		deleteFn: func(ctx context.Context, userID string) error {
			deletedAccountUserID = userID
			return nil
		},
	}

	teamRepo := &mockTeamRepo{
		findByMemberIDFn: func(ctx context.Context, memberID string) ([]domain.Team, error) {
			return nil, nil
		},
	}
	h := newAdminHandler(userRepo, accountRepo, teamRepo)

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{userId}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if deletedAccountUserID != "u-1" {
		t.Errorf("expected account deleted for u-1, got %s", deletedAccountUserID)
	}
	if deletedUserID != "u-1" {
		t.Errorf("expected user deleted u-1, got %s", deletedUserID)
	}
}

func TestAddMember(t *testing.T) {
	var addedTeamID, addedUserID string

	teamRepo := &mockTeamRepo{
		addMemberFn: func(ctx context.Context, teamID, userID string) error {
			addedTeamID = teamID
			addedUserID = userID
			return nil
		},
	}

	h := newAdminHandler(&mockUserRepo{}, &mockUserAccountRepo{}, teamRepo)

	r := chi.NewRouter()
	r.Post("/api/teams/{teamId}/members", h.AddMember)

	body := `{"userId":"u-2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/teams/t-1/members", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if addedTeamID != "t-1" || addedUserID != "u-2" {
		t.Errorf("expected t-1/u-2, got %s/%s", addedTeamID, addedUserID)
	}
}

func TestRemoveMember(t *testing.T) {
	var removedTeamID, removedUserID string

	teamRepo := &mockTeamRepo{
		removeMemberFn: func(ctx context.Context, teamID, userID string) error {
			removedTeamID = teamID
			removedUserID = userID
			return nil
		},
	}

	h := newAdminHandler(&mockUserRepo{}, &mockUserAccountRepo{}, teamRepo)

	r := chi.NewRouter()
	r.Delete("/api/teams/{teamId}/members/{userId}", h.RemoveMember)

	req := httptest.NewRequest(http.MethodDelete, "/api/teams/t-1/members/u-2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if removedTeamID != "t-1" || removedUserID != "u-2" {
		t.Errorf("expected t-1/u-2, got %s/%s", removedTeamID, removedUserID)
	}
}

func TestCreateTeam(t *testing.T) {
	var savedTeam *domain.Team

	teamRepo := &mockTeamRepo{
		saveFn: func(ctx context.Context, team *domain.Team) error {
			savedTeam = team
			return nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Username: "leader"}, nil
		},
	}
	accountRepo := &mockUserAccountRepo{
		findAllFn: func(ctx context.Context) ([]domain.UserAccount, error) {
			return []domain.UserAccount{
				{ID: "a-1", UserID: "u-leader", Roles: map[domain.Role]bool{domain.RoleTeamMember: true}},
			}, nil
		},
		saveFn: func(ctx context.Context, account *domain.UserAccount) error { return nil },
	}

	h := newAdminHandler(userRepo, accountRepo, teamRepo)
	body := `{"name":"Frontend","leaderId":"u-leader"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/teams", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateTeam(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	if savedTeam == nil {
		t.Fatal("team was not saved")
	}
	if !strings.HasPrefix(savedTeam.ID, "t-") {
		t.Errorf("expected team ID to start with t-, got %s", savedTeam.ID)
	}
	if savedTeam.Name != "Frontend" {
		t.Errorf("expected name Frontend, got %s", savedTeam.Name)
	}
	if len(savedTeam.LeaderIDs) != 1 || savedTeam.LeaderIDs[0] != "u-leader" {
		t.Errorf("expected leader u-leader, got %v", savedTeam.LeaderIDs)
	}
	if len(savedTeam.MemberIDs) != 1 || savedTeam.MemberIDs[0] != "u-leader" {
		t.Errorf("expected leader auto-added as member, got %v", savedTeam.MemberIDs)
	}

	var result TeamSummaryDTO
	json.NewDecoder(w.Body).Decode(&result)
	if result.Name != "Frontend" {
		t.Errorf("expected Frontend in response, got %s", result.Name)
	}
}

func TestCreateTeamMissingLeader(t *testing.T) {
	h := newAdminHandler(&mockUserRepo{}, &mockUserAccountRepo{}, &mockTeamRepo{})

	body := `{"name":"Frontend"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/teams", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateTeam(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteTeam(t *testing.T) {
	var deletedTeamID string

	teamRepo := &mockTeamRepo{
		findByIDFn: func(ctx context.Context, id string) (*domain.Team, error) {
			return &domain.Team{ID: id, Name: "Backend", LeaderIDs: []string{"u-leader"}}, nil
		},
		deleteFn: func(ctx context.Context, id string) error {
			deletedTeamID = id
			return nil
		},
		findAllFn: func(ctx context.Context) ([]domain.Team, error) {
			return nil, nil
		},
	}

	h := newAdminHandler(&mockUserRepo{}, &mockUserAccountRepo{
		findAllFn: func(ctx context.Context) ([]domain.UserAccount, error) {
			return nil, nil
		},
	}, teamRepo)

	r := chi.NewRouter()
	r.Delete("/api/admin/teams/{teamId}", h.DeleteTeam)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/teams/t-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if deletedTeamID != "t-1" {
		t.Errorf("expected team t-1 deleted, got %s", deletedTeamID)
	}
}
