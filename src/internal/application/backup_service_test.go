package application

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"
)

func TestBackupService_Export(t *testing.T) {
	repo := &mockBackupRepo{
		data: &domain.BackupFile{
			Users: []domain.User{{ID: "u-1", Username: "alice"}},
		},
	}
	svc := NewBackupService(repo)

	backup, err := svc.Export(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backup.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(backup.Users))
	}
	if backup.Users[0].Username != "alice" {
		t.Errorf("expected alice, got %s", backup.Users[0].Username)
	}
}

func TestBackupService_Restore(t *testing.T) {
	repo := &mockBackupRepo{}
	svc := NewBackupService(repo)

	backup := &domain.BackupFile{
		Users: []domain.User{{ID: "u-1", Username: "bob"}},
	}

	if err := svc.Restore(context.Background(), backup); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exported, _ := repo.Export(context.Background())
	if len(exported.Users) != 1 || exported.Users[0].Username != "bob" {
		t.Error("expected restored data to be accessible via export")
	}
}
