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

func TestGetProfile_Success(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID:          "u-1",
				Username:    "alice",
				DisplayName: "Alice",
				AvatarURL:   "https://example.com/alice.png",
			}, nil
		},
	}

	handler := NewProfileHandler(repo)
	req := newAuthenticatedRequest(http.MethodGet, "/api/profile", nil, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})
	rr := httptest.NewRecorder()

	handler.GetProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var user UserDTO
	json.NewDecoder(rr.Body).Decode(&user)
	if user.Username != "alice" {
		t.Errorf("expected alice, got %s", user.Username)
	}
}

func TestGetProfile_NotAuthenticated(t *testing.T) {
	handler := NewProfileHandler(&mockUserRepo{})
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	rr := httptest.NewRecorder()

	handler.GetProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestSetPlatformUsername_Success(t *testing.T) {
	var updatedPlatform, updatedUsername string
	repo := &mockUserRepo{
		updatePlatformUsernameFn: func(ctx context.Context, userID string, platform domain.GitPlatform, username string) error {
			updatedPlatform = platform.Name
			updatedUsername = username
			return nil
		},
	}

	handler := NewProfileHandler(repo)
	body := strings.NewReader(`{"platform":"GITHUB","username":"alice-gh"}`)
	req := newAuthenticatedRequest(http.MethodPut, "/api/profile/platform-username", body, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})
	rr := httptest.NewRecorder()

	handler.SetPlatformUsername(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if updatedPlatform != "GITHUB" || updatedUsername != "alice-gh" {
		t.Errorf("expected GITHUB/alice-gh, got %s/%s", updatedPlatform, updatedUsername)
	}
}

func TestSetPlatformUsername_MissingFields(t *testing.T) {
	handler := NewProfileHandler(&mockUserRepo{})
	body := strings.NewReader(`{"platform":"GITHUB"}`)
	req := newAuthenticatedRequest(http.MethodPut, "/api/profile/platform-username", body, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})
	rr := httptest.NewRecorder()

	handler.SetPlatformUsername(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
