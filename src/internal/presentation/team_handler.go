package presentation

import (
	"net/http"

	"contribution-tracker/internal/application"

	"github.com/go-chi/chi/v5"
)

type TeamHandler struct {
	teamRepo application.TeamRepository
}

func NewTeamHandler(teamRepo application.TeamRepository) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo}
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var teams interface{}
	var err error

	if authCtx.IsAdmin() {
		teams, err = h.teamRepo.FindAll(r.Context())
	} else {
		teams, err = h.teamRepo.FindByMemberID(r.Context(), authCtx.UserID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load teams")
		return
	}

	writeJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) AddRepository(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID required")
		return
	}

	var req AddRepoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RepoID == "" {
		writeError(w, http.StatusBadRequest, "repoId required")
		return
	}

	if err := h.teamRepo.AddRepository(r.Context(), teamID, req.RepoID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add repository")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *TeamHandler) RemoveRepository(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	repoID := chi.URLParam(r, "repoId")

	if teamID == "" || repoID == "" {
		writeError(w, http.StatusBadRequest, "team ID and repo ID required")
		return
	}

	if err := h.teamRepo.RemoveRepository(r.Context(), teamID, repoID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove repository")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
