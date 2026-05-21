package presentation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

func TestRequireAuth_MissingHeader(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{})

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidFormat(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{})

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{
		validateFn: func(token string) (*application.AuthContext, error) {
			return nil, application.NewUnauthorizedError("invalid")
		},
	})

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{
		validateFn: func(token string) (*application.AuthContext, error) {
			return &application.AuthContext{
				AccountID: "acc-1",
				UserID:    "u-1",
				Roles:     map[domain.Role]bool{domain.RoleTeamMember: true},
			}, nil
		},
	})

	var called bool
	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			t.Fatal("expected auth context")
		}
		if authCtx.UserID != "u-1" {
			t.Errorf("expected user ID u-1, got %s", authCtx.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireRole_HasRole(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := mw.RequireRole(domain.RoleAdmin)(inner)

	req := newAuthenticatedRequest(http.MethodGet, "/", nil, &application.AuthContext{
		Roles: map[domain.Role]bool{domain.RoleAdmin: true},
	})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	mw := NewAuthMiddleware(&mockAuthService{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	})

	handler := mw.RequireRole(domain.RoleAdmin)(inner)

	req := newAuthenticatedRequest(http.MethodGet, "/", nil, &application.AuthContext{
		Roles: map[domain.Role]bool{domain.RoleTeamMember: true},
	})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
