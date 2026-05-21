package persistence

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"
)

func TestPgxUserRepo_FindByIDWithPlatformUsernames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxUserRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name, email) VALUES ($1,$2,$3,$4)",
		"u-test-plat", "platuser", "Platform User", "plat@example.com")
	pool.Exec(ctx, "INSERT INTO user_platform_usernames (user_id, platform, username) VALUES ($1,$2,$3)",
		"u-test-plat", "GITHUB", "platuser-gh")

	user, err := repo.FindByID(ctx, "u-test-plat")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}

	if user.Username != "platuser" {
		t.Errorf("expected platuser, got %s", user.Username)
	}
	if user.GetPlatformUsername(domain.PlatformGitHub) != "platuser-gh" {
		t.Errorf("expected platuser-gh, got %s", user.GetPlatformUsername(domain.PlatformGitHub))
	}
}

func TestPgxUserRepo_FindByIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxUserRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-ids1", "idsuser1", "IDS User 1")
	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-ids2", "idsuser2", "IDS User 2")

	users, err := repo.FindByIDs(ctx, []string{"u-test-ids1", "u-test-ids2"})
	if err != nil {
		t.Fatalf("find by ids: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestPgxUserRepo_UpdatePlatformUsername(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxUserRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-upd", "upduser", "Update User")

	if err := repo.UpdatePlatformUsername(ctx, "u-test-upd", domain.PlatformGitHub, "upduser-github"); err != nil {
		t.Fatalf("update platform username: %v", err)
	}

	user, err := repo.FindByID(ctx, "u-test-upd")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if user.GetPlatformUsername(domain.PlatformGitHub) != "upduser-github" {
		t.Errorf("expected upduser-github, got %s", user.GetPlatformUsername(domain.PlatformGitHub))
	}

	if err := repo.UpdatePlatformUsername(ctx, "u-test-upd", domain.PlatformGitHub, "upduser-gh-new"); err != nil {
		t.Fatalf("update platform username again: %v", err)
	}
	user, _ = repo.FindByID(ctx, "u-test-upd")
	if user.GetPlatformUsername(domain.PlatformGitHub) != "upduser-gh-new" {
		t.Errorf("expected upduser-gh-new, got %s", user.GetPlatformUsername(domain.PlatformGitHub))
	}
}
