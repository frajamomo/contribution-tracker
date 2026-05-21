package application

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

func setupAuthService() (*AuthService, *mockUserAccountRepo, *mockUserRepo) {
	accountRepo := newMockUserAccountRepo()
	userRepo := newMockUserRepo()
	teamRepo := newMockTeamRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)

	account := &domain.UserAccount{
		ID:           "a-1",
		Username:     "alice",
		PasswordHash: string(hash),
		Roles:        map[domain.Role]bool{domain.RoleTeamMember: true},
		UserID:       "u-1",
	}
	accountRepo.accounts[account.ID] = account

	user := &domain.User{ID: "u-1", Username: "alice", DisplayName: "Alice"}
	userRepo.users[user.ID] = user
	userRepo.byAccountID[account.ID] = user

	svc := NewAuthService(accountRepo, userRepo, teamRepo, []byte("test-jwt-secret"))
	return svc, accountRepo, userRepo
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, _, _ := setupAuthService()

	token, err := svc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.Value == "" {
		t.Error("expected non-empty token")
	}
	if token.AccountID != "a-1" {
		t.Errorf("expected account ID a-1, got %s", token.AccountID)
	}
	if !token.Roles[domain.RoleTeamMember] {
		t.Error("expected TEAM_MEMBER role in token")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, _, _ := setupAuthService()

	_, err := svc.Login(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc, _, _ := setupAuthService()

	_, err := svc.Login(context.Background(), "nobody", "secret")
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestAuthService_Validate_Success(t *testing.T) {
	svc, _, _ := setupAuthService()

	token, err := svc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	authCtx, err := svc.Validate(token.Value)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if authCtx.AccountID != "a-1" {
		t.Errorf("expected account a-1, got %s", authCtx.AccountID)
	}
	if authCtx.UserID != "u-1" {
		t.Errorf("expected user u-1, got %s", authCtx.UserID)
	}
	if !authCtx.IsTeamMember() {
		t.Error("expected IsTeamMember to be true")
	}
}

func TestAuthService_Validate_InvalidToken(t *testing.T) {
	svc, _, _ := setupAuthService()

	_, err := svc.Validate("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
