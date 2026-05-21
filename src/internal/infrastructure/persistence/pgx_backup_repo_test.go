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
