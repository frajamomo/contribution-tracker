package presentation

import (
	"net/http"
	"strings"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TeamHandler struct {
	teamRepo  application.TeamRepository
	repoStore application.RepositoryStore
	userRepo  application.UserRepository
}

func NewTeamHandler(teamRepo application.TeamRepository, repoStore application.RepositoryStore, userRepo application.UserRepository) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo, repoStore: repoStore, userRepo: userRepo}
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var teams []domain.Team
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

	dtos := make([]TeamDTO, len(teams))
	for i, t := range teams {
		var repos []RepoDTO
		if len(t.RepositoryIDs) > 0 {
			repoEntities, repoErr := h.repoStore.FindByIDs(r.Context(), t.RepositoryIDs)
			if repoErr == nil {
				repos = make([]RepoDTO, len(repoEntities))
				for j, re := range repoEntities {
					repos[j] = RepoDTO{ID: re.ID, FullName: re.FullName, Platform: re.Platform.Name}
				}
			}
		}
		if repos == nil {
			repos = []RepoDTO{}
		}

		var members []MemberDTO
		if len(t.MemberIDs) > 0 {
			users, userErr := h.userRepo.FindByIDs(r.Context(), t.MemberIDs)
			if userErr == nil {
				members = make([]MemberDTO, len(users))
				for j, u := range users {
					members[j] = MemberDTO{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName}
				}
			}
		}
		if members == nil {
			members = []MemberDTO{}
		}

		dtos[i] = TeamDTO{
			ID:            t.ID,
			Name:          t.Name,
			MemberIDs:     t.MemberIDs,
			Members:       members,
			RepositoryIDs: t.RepositoryIDs,
			Repositories:  repos,
		}
	}

	writeJSON(w, http.StatusOK, dtos)
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

	if req.FullName == "" || req.Platform == "" {
		writeError(w, http.StatusBadRequest, "fullName and platform are required")
		return
	}

	platform := domain.GitPlatform{Name: strings.ToUpper(req.Platform)}
	apiToken := req.APIToken

	if apiToken == "" {
		team, err := h.teamRepo.FindByID(r.Context(), teamID)
		if err == nil && len(team.RepositoryIDs) > 0 {
			existing, err := h.repoStore.FindByIDs(r.Context(), team.RepositoryIDs)
			if err == nil {
				for _, er := range existing {
					if er.Platform == platform && er.APIToken != "" {
						apiToken = er.APIToken
						break
					}
				}
			}
		}
		if apiToken == "" {
			writeError(w, http.StatusBadRequest, "apiToken is required (no existing token found for this platform)")
			return
		}
	}

	nameParts := strings.Split(req.FullName, "/")
	repoName := req.FullName
	if len(nameParts) > 1 {
		repoName = nameParts[len(nameParts)-1]
	}

	repo := &domain.Repository{
		ID:       "r-" + uuid.New().String()[:8],
		Name:     repoName,
		FullName: req.FullName,
		Platform: platform,
		APIToken: apiToken,
	}

	saved, err := h.repoStore.Upsert(r.Context(), repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save repository")
		return
	}

	if err := h.teamRepo.AddRepository(r.Context(), teamID, saved.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add repository to team")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "added", "repoId": saved.ID})
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
