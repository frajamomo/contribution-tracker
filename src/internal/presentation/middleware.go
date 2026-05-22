package presentation

import (
	"context"
	"net/http"
	"strings"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const authContextKey contextKey = "authContext"

type AuthMiddleware struct {
	authService application.AuthServicePort
}

func NewAuthMiddleware(authService application.AuthServicePort) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		authCtx, err := m.authService.Validate(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), authContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				writeError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			hasRole := false
			for _, role := range roles {
				if authCtx.Roles[role] {
					hasRole = true
					break
				}
			}

			if !hasRole {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireTeamLeaderOrAdmin(teamRepo application.TeamRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				writeError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			if authCtx.Roles[domain.RoleAdmin] {
				next.ServeHTTP(w, r)
				return
			}

			teamID := chi.URLParam(r, "teamId")
			if teamID == "" {
				writeError(w, http.StatusBadRequest, "team ID required")
				return
			}

			team, err := teamRepo.FindByID(r.Context(), teamID)
			if err != nil {
				writeError(w, http.StatusNotFound, "team not found")
				return
			}

			for _, lid := range team.LeaderIDs {
				if lid == authCtx.UserID {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeError(w, http.StatusForbidden, "you are not a leader of this team")
		})
	}
}

func GetAuthContext(r *http.Request) *application.AuthContext {
	if ctx, ok := r.Context().Value(authContextKey).(*application.AuthContext); ok {
		return ctx
	}
	return nil
}
