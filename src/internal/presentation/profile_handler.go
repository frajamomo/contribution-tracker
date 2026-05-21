package presentation

import (
	"net/http"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

type ProfileHandler struct {
	userRepo application.UserRepository
}

func NewProfileHandler(userRepo application.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), authCtx.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, UserToDTO(*user))
}

func (h *ProfileHandler) SetPlatformUsername(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req PlatformUsernameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Platform == "" || req.Username == "" {
		writeError(w, http.StatusBadRequest, "platform and username required")
		return
	}

	platform := domain.GitPlatform{Name: req.Platform}

	if err := h.userRepo.UpdatePlatformUsername(r.Context(), authCtx.UserID, platform, req.Username); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update platform username")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
