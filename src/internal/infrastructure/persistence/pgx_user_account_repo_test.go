package persistence

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"
)

func TestPgxUserAccountRepo_SaveAndFind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)",
		"u-test-acct", "acctuser", "Account Test User")

	repo := NewPgxUserAccountRepo(pool)

	account := &domain.UserAccount{
		ID:           "a-test-acct",
		Username:     "acctuser",
		PasswordHash: "$2a$10$hash",
		Roles:        map[domain.Role]bool{domain.RoleTeamMember: true, domain.RoleTeamLeader: true},
		UserID:       "u-test-acct",
	}

	if err := repo.Save(ctx, account); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := repo.FindByUsername(ctx, "acctuser")
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if found.ID != "a-test-acct" {
		t.Errorf("expected a-test-acct, got %s", found.ID)
	}
	if !found.HasRole(domain.RoleTeamMember) {
		t.Error("expected TEAM_MEMBER role")
	}
	if !found.HasRole(domain.RoleTeamLeader) {
		t.Error("expected TEAM_LEADER role")
	}

	foundByID, err := repo.FindByID(ctx, "a-test-acct")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if foundByID.Username != "acctuser" {
		t.Errorf("expected acctuser, got %s", foundByID.Username)
	}
}
