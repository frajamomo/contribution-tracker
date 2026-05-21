package persistence

import (
	"context"
	"testing"

	"contribution-tracker/internal/domain"
)

func TestPgxTeamRepo_SaveAndFind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxTeamRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-t1", "tuser1", "Test User 1")
	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-t2", "tuser2", "Test User 2")

	team := &domain.Team{
		ID:        "t-test-1",
		Name:      "Test Team",
		MemberIDs: []string{"u-test-t1", "u-test-t2"},
	}

	if err := repo.Save(ctx, team); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := repo.FindByID(ctx, "t-test-1")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Name != "Test Team" {
		t.Errorf("expected Test Team, got %s", found.Name)
	}
	if len(found.MemberIDs) != 2 {
		t.Errorf("expected 2 members, got %d", len(found.MemberIDs))
	}
}

func TestPgxTeamRepo_AddAndRemoveRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxTeamRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-ar1", "aruser1", "AR User")
	pool.Exec(ctx, "INSERT INTO teams (id, name) VALUES ($1,$2)", "t-test-ar", "AR Team")
	pool.Exec(ctx, "INSERT INTO team_members (team_id, user_id) VALUES ($1,$2)", "t-test-ar", "u-test-ar1")
	pool.Exec(ctx, "INSERT INTO repositories (id, name, full_name, url, platform) VALUES ($1,$2,$3,$4,$5)",
		"r-test-ar1", "ar-repo", "testorg/ar-repo", "https://github.com/testorg/ar-repo", "GITHUB")

	if err := repo.AddRepository(ctx, "t-test-ar", "r-test-ar1"); err != nil {
		t.Fatalf("add repository: %v", err)
	}

	team, err := repo.FindByID(ctx, "t-test-ar")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(team.RepositoryIDs) != 1 {
		t.Errorf("expected 1 repo, got %d", len(team.RepositoryIDs))
	}

	if err := repo.RemoveRepository(ctx, "t-test-ar", "r-test-ar1"); err != nil {
		t.Fatalf("remove repository: %v", err)
	}

	team, _ = repo.FindByID(ctx, "t-test-ar")
	if len(team.RepositoryIDs) != 0 {
		t.Errorf("expected 0 repos, got %d", len(team.RepositoryIDs))
	}
}

func TestPgxTeamRepo_FindByMemberID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	ctx := context.Background()
	repo := NewPgxTeamRepo(pool)

	pool.Exec(ctx, "INSERT INTO users (id, username, display_name) VALUES ($1,$2,$3)", "u-test-fm1", "fmuser1", "FM User")
	pool.Exec(ctx, "INSERT INTO teams (id, name) VALUES ($1,$2)", "t-test-fm", "FM Team")
	pool.Exec(ctx, "INSERT INTO team_members (team_id, user_id) VALUES ($1,$2)", "t-test-fm", "u-test-fm1")

	teams, err := repo.FindByMemberID(ctx, "u-test-fm1")
	if err != nil {
		t.Fatalf("find by member: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	if teams[0].Name != "FM Team" {
		t.Errorf("expected FM Team, got %s", teams[0].Name)
	}
}
