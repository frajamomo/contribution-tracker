package persistence

import (
	"context"
	"fmt"

	"contribution-tracker/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserRepo struct {
	pool *pgxpool.Pool
}

func NewPgxUserRepo(pool *pgxpool.Pool) *PgxUserRepo {
	return &PgxUserRepo{pool: pool}
}

func (r *PgxUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, username, display_name, email, avatar_url FROM users WHERE id = $1", id)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.AvatarURL); err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	platforms, err := r.loadPlatformUsernames(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	u.PlatformUsernames = platforms
	return &u, nil
}

func (r *PgxUserRepo) FindByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, username, display_name, email, avatar_url FROM users WHERE id = ANY($1)", ids)
	if err != nil {
		return nil, fmt.Errorf("find users by ids: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	for i := range users {
		platforms, err := r.loadPlatformUsernames(ctx, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].PlatformUsernames = platforms
	}

	return users, nil
}

func (r *PgxUserRepo) FindByAccountID(ctx context.Context, accountID string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT u.id, u.username, u.display_name, u.email, u.avatar_url
		 FROM users u JOIN user_accounts a ON u.id = a.user_id
		 WHERE a.id = $1`, accountID)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.AvatarURL); err != nil {
		return nil, fmt.Errorf("find user by account id: %w", err)
	}

	platforms, err := r.loadPlatformUsernames(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	u.PlatformUsernames = platforms
	return &u, nil
}

func (r *PgxUserRepo) UpdatePlatformUsername(ctx context.Context, userID string, platform domain.GitPlatform, username string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_platform_usernames (user_id, platform, username)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, platform) DO UPDATE SET username = $3`,
		userID, platform.Name, username)
	if err != nil {
		return fmt.Errorf("update platform username: %w", err)
	}
	return nil
}

func (r *PgxUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, username, display_name, email, avatar_url FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("find all users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}

	for i := range users {
		platforms, err := r.loadPlatformUsernames(ctx, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].PlatformUsernames = platforms
	}

	return users, nil
}

func (r *PgxUserRepo) Save(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, display_name, email, avatar_url)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET username=$2, display_name=$3, email=$4, avatar_url=$5`,
		user.ID, user.Username, user.DisplayName, user.Email, user.AvatarURL)
	if err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (r *PgxUserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *PgxUserRepo) loadPlatformUsernames(ctx context.Context, userID string) (map[domain.GitPlatform]string, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT platform, username FROM user_platform_usernames WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("load platform usernames: %w", err)
	}
	defer rows.Close()

	platforms := make(map[domain.GitPlatform]string)
	for rows.Next() {
		var platform, username string
		if err := rows.Scan(&platform, &username); err != nil {
			return nil, fmt.Errorf("scan platform username: %w", err)
		}
		platforms[domain.GitPlatform{Name: platform}] = username
	}
	return platforms, nil
}
