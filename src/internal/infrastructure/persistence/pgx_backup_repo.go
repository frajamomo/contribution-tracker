package persistence

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"contribution-tracker/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxBackupRepo struct {
	pool *pgxpool.Pool
}

func NewPgxBackupRepo(pool *pgxpool.Pool) *PgxBackupRepo {
	return &PgxBackupRepo{pool: pool}
}

func (r *PgxBackupRepo) Export(ctx context.Context) (*domain.BackupFile, error) {
	userRepo := NewPgxUserRepo(r.pool)
	accountRepo := NewPgxUserAccountRepo(r.pool)
	teamRepo := NewPgxTeamRepo(r.pool)
	repoStore := NewPgxRepositoryStore(r.pool)
	configRepo := NewPgxConfigRepo(r.pool)

	users, err := userRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("export users: %w", err)
	}

	accounts, err := accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("export accounts: %w", err)
	}

	teams, err := teamRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("export teams: %w", err)
	}

	repos, err := repoStore.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("export repos: %w", err)
	}

	config, err := configRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("export config: %w", err)
	}

	for i := range repos {
		if repos[i].APIToken != "" {
			repos[i].APIToken = base64.StdEncoding.EncodeToString([]byte(repos[i].APIToken))
		}
	}

	return &domain.BackupFile{
		Metadata: domain.BackupMetadata{
			AppVersion: "1.0.0",
			ExportedAt: time.Now(),
		},
		Users:        users,
		Accounts:     accounts,
		Teams:        teams,
		Repositories: repos,
		Config:       config,
	}, nil
}

func (r *PgxBackupRepo) Restore(ctx context.Context, data *domain.BackupFile) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin restore tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tables := []string{
		"team_repositories", "team_members", "user_platform_usernames",
		"user_accounts", "teams", "repositories", "users", "app_config",
	}
	for _, table := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	for _, u := range data.Users {
		_, err := tx.Exec(ctx,
			"INSERT INTO users (id, username, display_name, email, avatar_url) VALUES ($1,$2,$3,$4,$5)",
			u.ID, u.Username, u.DisplayName, u.Email, u.AvatarURL)
		if err != nil {
			return fmt.Errorf("restore user %s: %w", u.ID, err)
		}
		for platform, username := range u.PlatformUsernames {
			_, err := tx.Exec(ctx,
				"INSERT INTO user_platform_usernames (user_id, platform, username) VALUES ($1,$2,$3)",
				u.ID, platform.Name, username)
			if err != nil {
				return fmt.Errorf("restore platform username: %w", err)
			}
		}
	}

	for _, a := range data.Accounts {
		roleStrings := rolesToStrings(a.Roles)
		_, err := tx.Exec(ctx,
			"INSERT INTO user_accounts (id, username, password_hash, roles, user_id) VALUES ($1,$2,$3,$4,$5)",
			a.ID, a.Username, a.PasswordHash, roleStrings, a.UserID)
		if err != nil {
			return fmt.Errorf("restore account %s: %w", a.ID, err)
		}
	}

	for _, t := range data.Teams {
		_, err := tx.Exec(ctx, "INSERT INTO teams (id, name) VALUES ($1,$2)", t.ID, t.Name)
		if err != nil {
			return fmt.Errorf("restore team %s: %w", t.ID, err)
		}
		for _, memberID := range t.MemberIDs {
			_, err := tx.Exec(ctx,
				"INSERT INTO team_members (team_id, user_id) VALUES ($1,$2)", t.ID, memberID)
			if err != nil {
				return fmt.Errorf("restore team member: %w", err)
			}
		}
		for _, repoID := range t.RepositoryIDs {
			_, err := tx.Exec(ctx,
				"INSERT INTO team_repositories (team_id, repo_id) VALUES ($1,$2)", t.ID, repoID)
			if err != nil {
				return fmt.Errorf("restore team repo: %w", err)
			}
		}
	}

	for _, repo := range data.Repositories {
		token := repo.APIToken
		if token != "" {
			if decoded, err := base64.StdEncoding.DecodeString(token); err == nil {
				token = string(decoded)
			}
		}
		_, err := tx.Exec(ctx,
			"INSERT INTO repositories (id, name, full_name, url, platform, api_token) VALUES ($1,$2,$3,$4,$5,$6)",
			repo.ID, repo.Name, repo.FullName, repo.URL, repo.Platform.Name, token)
		if err != nil {
			return fmt.Errorf("restore repo %s: %w", repo.ID, err)
		}
	}

	for key, value := range data.Config {
		_, err := tx.Exec(ctx,
			"INSERT INTO app_config (key, value) VALUES ($1,$2)", key, value)
		if err != nil {
			return fmt.Errorf("restore config %s: %w", key, err)
		}
	}

	return tx.Commit(ctx)
}
