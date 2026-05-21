package presentation

import (
	"net/http"

	"contribution-tracker/internal/application"
)

type AuthHandler struct {
	authService application.AuthServicePort
	userRepo    application.UserRepository
}

func NewAuthHandler(authService application.AuthServicePort, userRepo application.UserRepository) *AuthHandler {
	return &AuthHandler{authService: authService, userRepo: userRepo}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	token, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	user, err := h.userRepo.FindByAccountID(r.Context(), token.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	roles := make([]string, 0, len(token.Roles))
	for role := range token.Roles {
		roles = append(roles, string(role))
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: token.Value,
		Roles: roles,
		User:  UserToDTO(*user),
	})
}
