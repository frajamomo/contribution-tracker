package presentation

import (
	"context"
	"net/http"
	"strings"

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
					dto.Teams = append(dto.Teams, TeamSummaryDTO{ID: t.ID, Name: t.Name, LeaderIDs: t.LeaderIDs})
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

	teams, err := h.teamRepo.FindByMemberID(r.Context(), userID)
	if err == nil {
		for _, t := range teams {
			for _, lid := range t.LeaderIDs {
				if lid == userID && len(t.LeaderIDs) == 1 {
					writeError(w, http.StatusBadRequest, "user is the only leader of team "+t.Name+"; reassign leadership first")
					return
				}
			}
		}
	}

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

	if req.Name == "" || req.LeaderID == "" {
		writeError(w, http.StatusBadRequest, "name and leaderId are required")
		return
	}

	leader, err := h.userRepo.FindByID(r.Context(), req.LeaderID)
	if err != nil || leader == nil {
		writeError(w, http.StatusBadRequest, "leader user not found")
		return
	}

	teamID := "t-" + uuid.New().String()[:8]

	team := &domain.Team{
		ID:        teamID,
		Name:      req.Name,
		LeaderIDs: []string{req.LeaderID},
		MemberIDs: []string{req.LeaderID},
	}

	if err := h.teamRepo.Save(r.Context(), team); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create team")
		return
	}

	h.ensureLeaderRole(r.Context(), req.LeaderID)

	writeJSON(w, http.StatusCreated, TeamSummaryDTO{ID: teamID, Name: req.Name, LeaderIDs: []string{req.LeaderID}})
}

func (h *AdminHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	team, err := h.teamRepo.FindByID(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	formerLeaderIDs := team.LeaderIDs

	if err := h.teamRepo.Delete(r.Context(), teamID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete team")
		return
	}

	for _, leaderID := range formerLeaderIDs {
		h.removeLeaderRoleIfOrphaned(r.Context(), leaderID)
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
		if strings.Contains(err.Error(), "leader") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *AdminHandler) AddLeader(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req AddLeaderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}

	team, err := h.teamRepo.FindByID(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}

	isMember := false
	for _, mid := range team.MemberIDs {
		if mid == req.UserID {
			isMember = true
			break
		}
	}
	if !isMember {
		writeError(w, http.StatusBadRequest, "user must be a member of the team first")
		return
	}

	if err := h.teamRepo.AddLeader(r.Context(), teamID, req.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add leader")
		return
	}

	h.ensureLeaderRole(r.Context(), req.UserID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *AdminHandler) RemoveLeader(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := chi.URLParam(r, "userId")

	if err := h.teamRepo.RemoveLeader(r.Context(), teamID, userID); err != nil {
		if strings.Contains(err.Error(), "last leader") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove leader")
		return
	}

	h.removeLeaderRoleIfOrphaned(r.Context(), userID)

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *AdminHandler) ensureLeaderRole(ctx context.Context, userID string) {
	accounts, err := h.userAccountRepo.FindAll(ctx)
	if err != nil {
		return
	}
	for i := range accounts {
		if accounts[i].UserID == userID {
			if !accounts[i].Roles[domain.RoleTeamLeader] {
				accounts[i].Roles[domain.RoleTeamLeader] = true
				h.userAccountRepo.Save(ctx, &accounts[i])
			}
			return
		}
	}
}

func (h *AdminHandler) removeLeaderRoleIfOrphaned(ctx context.Context, userID string) {
	teams, err := h.teamRepo.FindAll(ctx)
	if err != nil {
		return
	}
	for _, t := range teams {
		for _, lid := range t.LeaderIDs {
			if lid == userID {
				return
			}
		}
	}
	accounts, err := h.userAccountRepo.FindAll(ctx)
	if err != nil {
		return
	}
	for i := range accounts {
		if accounts[i].UserID == userID && accounts[i].Roles[domain.RoleTeamLeader] {
			delete(accounts[i].Roles, domain.RoleTeamLeader)
			h.userAccountRepo.Save(ctx, &accounts[i])
			return
		}
	}
}
