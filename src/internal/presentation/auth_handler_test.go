package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

func TestLogin_Success(t *testing.T) {
	authSvc := &mockAuthService{
		loginFn: func(ctx context.Context, username, password string) (*application.AuthToken, error) {
			return &application.AuthToken{
				Value:     "jwt-token",
				AccountID: "acc-1",
				Roles:     map[domain.Role]bool{domain.RoleTeamMember: true},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByAccountIDFn: func(ctx context.Context, accountID string) (*domain.User, error) {
			return &domain.User{ID: "u-1", Username: "alice", DisplayName: "Alice"}, nil
		},
	}

	handler := NewAuthHandler(authSvc, userRepo)
	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp LoginResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Token != "jwt-token" {
		t.Errorf("expected token jwt-token, got %s", resp.Token)
	}
	if resp.User.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.User.Username)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	authSvc := &mockAuthService{
		loginFn: func(ctx context.Context, username, password string) (*application.AuthToken, error) {
			return nil, application.NewUnauthorizedError("invalid credentials")
		},
	}
	userRepo := &mockUserRepo{}

	handler := NewAuthHandler(authSvc, userRepo)
	body := strings.NewReader(`{"username":"alice","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLogin_MissingFields(t *testing.T) {
	handler := NewAuthHandler(&mockAuthService{}, &mockUserRepo{})
	body := strings.NewReader(`{"username":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	handler := NewAuthHandler(&mockAuthService{}, &mockUserRepo{})
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
