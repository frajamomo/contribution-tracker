package presentation

import (
	"net/http"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	userRepo        application.UserRepository
	userAccountRepo application.UserAccountRepository
	teamRepo        application.TeamRepository
}

func NewAdminHandler(
	userRepo application.UserRepository,
	userAccountRepo application.UserAccountRepository,
	teamRepo application.TeamRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:        userRepo,
		userAccountRepo: userAccountRepo,
		teamRepo:        teamRepo,
	}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.FindAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	accounts, err := h.userAccountRepo.FindAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load accounts")
		return
	}

	teams, err := h.teamRepo.FindAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load teams")
		return
	}

	accountByUserID := make(map[string]*domain.UserAccount, len(accounts))
	for i := range accounts {
		accountByUserID[accounts[i].UserID] = &accounts[i]
	}

	var result []AdminUserDTO
	for _, u := range users {
		dto := AdminUserDTO{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Email:       u.Email,
		}

		if acct, ok := accountByUserID[u.ID]; ok {
			for _, r := range acct.RoleList() {
				dto.Roles = append(dto.Roles, string(r))
			}
		}

		for _, t := range teams {
			for _, mid := range t.MemberIDs {
				if mid == u.ID {
					dto.Teams = append(dto.Teams, TeamSummaryDTO{ID: t.ID, Name: t.Name})
					break
				}
			}
		}
		if dto.Teams == nil {
			dto.Teams = []TeamSummaryDTO{}
		}
		if dto.Roles == nil {
			dto.Roles = []string{}
		}
		result = append(result, dto)
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "username, displayName, and password are required")
		return
	}

	if len(req.Roles) == 0 {
		req.Roles = []string{string(domain.RoleTeamMember)}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	userID := "u-" + uuid.New().String()[:8]
	accountID := "a-" + uuid.New().String()[:8]

	user := &domain.User{
		ID:          userID,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Email:       req.Email,
	}

	if err := h.userRepo.Save(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	roles := make(map[domain.Role]bool, len(req.Roles))
	for _, r := range req.Roles {
		roles[domain.Role(r)] = true
	}

	account := &domain.UserAccount{
		ID:           accountID,
		Username:     req.Username,
		PasswordHash: string(hash),
		Roles:        roles,
		UserID:       userID,
	}

	if err := h.userAccountRepo.Save(r.Context(), account); err != nil {
		h.userRepo.Delete(r.Context(), userID)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	writeJSON(w, http.StatusCreated, AdminUserDTO{
		ID:          userID,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Roles:       req.Roles,
		Teams:       []TeamSummaryDTO{},
	})
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")

	if err := h.userAccountRepo.Delete(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}

	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req CreateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	teamID := "t-" + uuid.New().String()[:8]

	team := &domain.Team{
		ID:   teamID,
		Name: req.Name,
	}

	if err := h.teamRepo.Save(r.Context(), team); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create team")
		return
	}

	writeJSON(w, http.StatusCreated, TeamSummaryDTO{ID: teamID, Name: req.Name})
}

func (h *AdminHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	if err := h.teamRepo.Delete(r.Context(), teamID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete team")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req AddMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}

	if err := h.teamRepo.AddMember(r.Context(), teamID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *AdminHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := chi.URLParam(r, "userId")

	if err := h.teamRepo.RemoveMember(r.Context(), teamID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
