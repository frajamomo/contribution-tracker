package persistence

import (
	"context"
	"testing"
)

func TestPgxConfigRepo_SetAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxConfigRepo(pool)

	if err := repo.Set(ctx, "GITHUB_API_KEY", "test-key-123"); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := repo.Get(ctx, "GITHUB_API_KEY")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "test-key-123" {
		t.Errorf("expected test-key-123, got %s", val)
	}

	// Upsert
	if err := repo.Set(ctx, "GITHUB_API_KEY", "updated-key"); err != nil {
		t.Fatalf("update: %v", err)
	}
	val, _ = repo.Get(ctx, "GITHUB_API_KEY")
	if val != "updated-key" {
		t.Errorf("expected updated-key, got %s", val)
	}
}

func TestPgxConfigRepo_FindAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxConfigRepo(pool)

	repo.Set(ctx, "KEY_A", "val-a")
	repo.Set(ctx, "KEY_B", "val-b")

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 config entries, got %d", len(all))
	}
	if all["KEY_A"] != "val-a" {
		t.Errorf("expected val-a, got %s", all["KEY_A"])
	}
}
