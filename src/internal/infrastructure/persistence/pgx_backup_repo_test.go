package persistence

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"
)

func TestPgxBackupRepo_ExportAndRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxBackupRepo(pool)

	// Export includes seed data
	backup, err := repo.Export(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(backup.Users) < 1 {
		t.Errorf("expected at least 1 user from seed, got %d", len(backup.Users))
	}

	// Restore with entirely different data
	newBackup := &domain.BackupFile{
		Users: []domain.User{
			{ID: "u-restored", Username: "restored-user", DisplayName: "Restored"},
		},
		Accounts: []domain.UserAccount{
			{ID: "a-restored", Username: "restored-user", PasswordHash: "$2a$10$hash", Roles: map[domain.Role]bool{domain.RoleAdmin: true}, UserID: "u-restored"},
		},
		Config: map[string]string{"RESTORED_KEY": "restored-val"},
	}

	if err := repo.Restore(ctx, newBackup); err != nil {
		t.Fatalf("restore: %v", err)
	}

	// Verify restored data replaces seed data
	exported, err := repo.Export(ctx)
	if err != nil {
		t.Fatalf("export after restore: %v", err)
	}
	if len(exported.Users) != 1 {
		t.Errorf("expected 1 user after restore, got %d", len(exported.Users))
	}
	if exported.Users[0].Username != "restored-user" {
		t.Errorf("expected restored-user, got %s", exported.Users[0].Username)
	}
	if len(exported.Accounts) != 1 {
		t.Errorf("expected 1 account after restore, got %d", len(exported.Accounts))
	}
	if exported.Config["RESTORED_KEY"] != "restored-val" {
		t.Error("expected RESTORED_KEY config after restore")
	}
}

func TestPgxBackupRepo_RestoreTeamsWithRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxBackupRepo(pool)

	backup := &domain.BackupFile{
		Users: []domain.User{
			{ID: "u-1", Username: "leader", DisplayName: "Leader"},
			{ID: "u-2", Username: "dev", DisplayName: "Dev"},
		},
		Accounts: []domain.UserAccount{
			{ID: "a-1", Username: "leader", PasswordHash: "$2a$10$hash", Roles: map[domain.Role]bool{domain.RoleTeamLeader: true}, UserID: "u-1"},
			{ID: "a-2", Username: "dev", PasswordHash: "$2a$10$hash", Roles: map[domain.Role]bool{domain.RoleTeamMember: true}, UserID: "u-2"},
		},
		Repositories: []domain.Repository{
			{ID: "r-1", Name: "repo-a", FullName: "org/repo-a", Platform: domain.PlatformGitHub},
			{ID: "r-2", Name: "repo-b", FullName: "org/repo-b", Platform: domain.PlatformGitLab},
		},
		Teams: []domain.Team{
			{
				ID:            "t-1",
				Name:          "Backend",
				MemberIDs:     []string{"u-1", "u-2"},
				RepositoryIDs: []string{"r-1", "r-2"},
			},
		},
	}

	if err := repo.Restore(ctx, backup); err != nil {
		t.Fatalf("restore with teams referencing repos: %v", err)
	}

	exported, err := repo.Export(ctx)
	if err != nil {
		t.Fatalf("export after restore: %v", err)
	}

	if len(exported.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(exported.Users))
	}
	if len(exported.Repositories) != 2 {
		t.Errorf("expected 2 repos, got %d", len(exported.Repositories))
	}
	if len(exported.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(exported.Teams))
	}

	team := exported.Teams[0]
	if team.Name != "Backend" {
		t.Errorf("expected team Backend, got %s", team.Name)
	}
	if len(team.MemberIDs) != 2 {
		t.Errorf("expected 2 team members, got %d", len(team.MemberIDs))
	}
	if len(team.RepositoryIDs) != 2 {
		t.Errorf("expected 2 team repos, got %d", len(team.RepositoryIDs))
	}
}

func TestPgxBackupRepo_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxBackupRepo(pool)

	original := &domain.BackupFile{
		Users: []domain.User{
			{ID: "u-a", Username: "alice", DisplayName: "Alice", Email: "alice@test.com", PlatformUsernames: map[domain.GitPlatform]string{domain.PlatformGitHub: "alice-gh"}},
			{ID: "u-b", Username: "bob", DisplayName: "Bob", Email: "bob@test.com"},
		},
		Accounts: []domain.UserAccount{
			{ID: "a-a", Username: "alice", PasswordHash: "$2a$10$hash1", Roles: map[domain.Role]bool{domain.RoleAdmin: true}, UserID: "u-a"},
			{ID: "a-b", Username: "bob", PasswordHash: "$2a$10$hash2", Roles: map[domain.Role]bool{domain.RoleTeamMember: true}, UserID: "u-b"},
		},
		Repositories: []domain.Repository{
			{ID: "r-x", Name: "frontend", FullName: "org/frontend", URL: "https://github.com/org/frontend", Platform: domain.PlatformGitHub, APIToken: "ghp_secret123"},
		},
		Teams: []domain.Team{
			{ID: "t-x", Name: "Frontend", MemberIDs: []string{"u-a", "u-b"}, RepositoryIDs: []string{"r-x"}},
		},
		Config: map[string]string{"theme": "dark"},
	}

	if err := repo.Restore(ctx, original); err != nil {
		t.Fatalf("restore: %v", err)
	}

	exported, err := repo.Export(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Users
	if len(exported.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(exported.Users))
	}
	usersByID := map[string]domain.User{}
	for _, u := range exported.Users {
		usersByID[u.ID] = u
	}
	alice := usersByID["u-a"]
	if alice.Username != "alice" || alice.Email != "alice@test.com" {
		t.Errorf("alice mismatch: %+v", alice)
	}
	if alice.PlatformUsernames[domain.PlatformGitHub] != "alice-gh" {
		t.Errorf("expected alice github username alice-gh, got %s", alice.PlatformUsernames[domain.PlatformGitHub])
	}

	// Accounts
	if len(exported.Accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(exported.Accounts))
	}

	// Repos — API token should be base64-encoded in export
	if len(exported.Repositories) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(exported.Repositories))
	}
	if exported.Repositories[0].FullName != "org/frontend" {
		t.Errorf("expected org/frontend, got %s", exported.Repositories[0].FullName)
	}
	if exported.Repositories[0].APIToken == "" {
		t.Error("expected non-empty API token in export")
	}

	// Teams with members and repos
	if len(exported.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(exported.Teams))
	}
	if exported.Teams[0].Name != "Frontend" {
		t.Errorf("expected Frontend, got %s", exported.Teams[0].Name)
	}
	if len(exported.Teams[0].MemberIDs) != 2 {
		t.Errorf("expected 2 members, got %d", len(exported.Teams[0].MemberIDs))
	}
	if len(exported.Teams[0].RepositoryIDs) != 1 {
		t.Errorf("expected 1 repo in team, got %d", len(exported.Teams[0].RepositoryIDs))
	}

	// Config
	if exported.Config["theme"] != "dark" {
		t.Errorf("expected config theme=dark, got %s", exported.Config["theme"])
	}

	// Restore again from the export to verify the cycle is stable
	if err := repo.Restore(ctx, exported); err != nil {
		t.Fatalf("second restore (from export): %v", err)
	}

	reExported, err := repo.Export(ctx)
	if err != nil {
		t.Fatalf("second export: %v", err)
	}
	if len(reExported.Users) != 2 || len(reExported.Teams) != 1 || len(reExported.Repositories) != 1 {
		t.Errorf("round-trip unstable: users=%d teams=%d repos=%d",
			len(reExported.Users), len(reExported.Teams), len(reExported.Repositories))
	}
}
