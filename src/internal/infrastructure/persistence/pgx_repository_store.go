package persistence

import (
	"context"
	"fmt"

	"contribution-tracker/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxRepositoryStore struct {
	pool *pgxpool.Pool
}

func NewPgxRepositoryStore(pool *pgxpool.Pool) *PgxRepositoryStore {
	return &PgxRepositoryStore{pool: pool}
}

func (r *PgxRepositoryStore) FindByID(ctx context.Context, id string) (*domain.Repository, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, name, full_name, url, platform, api_token FROM repositories WHERE id = $1", id)

	var repo domain.Repository
	var platform string
	if err := row.Scan(&repo.ID, &repo.Name, &repo.FullName, &repo.URL, &platform, &repo.APIToken); err != nil {
		return nil, fmt.Errorf("find repo by id: %w", err)
	}
	repo.Platform = domain.GitPlatform{Name: platform}
	return &repo, nil
}

func (r *PgxRepositoryStore) FindByIDs(ctx context.Context, ids []string) ([]domain.Repository, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, name, full_name, url, platform, api_token FROM repositories WHERE id = ANY($1)", ids)
	if err != nil {
		return nil, fmt.Errorf("find repos by ids: %w", err)
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		var platform string
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.FullName, &repo.URL, &platform, &repo.APIToken); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		repo.Platform = domain.GitPlatform{Name: platform}
		repos = append(repos, repo)
	}
	return repos, nil
}

func (r *PgxRepositoryStore) Save(ctx context.Context, repo *domain.Repository) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO repositories (id, name, full_name, url, platform, api_token)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE SET name=$2, full_name=$3, url=$4, platform=$5, api_token=$6`,
		repo.ID, repo.Name, repo.FullName, repo.URL, repo.Platform.Name, repo.APIToken)
	if err != nil {
		return fmt.Errorf("save repo: %w", err)
	}
	return nil
}

func (r *PgxRepositoryStore) Upsert(ctx context.Context, repo *domain.Repository) (*domain.Repository, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO repositories (id, name, full_name, url, platform, api_token)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (full_name, platform) DO UPDATE SET name=$2, url=$4, api_token=$6
		 RETURNING id, name, full_name, url, platform, api_token`,
		repo.ID, repo.Name, repo.FullName, repo.URL, repo.Platform.Name, repo.APIToken)

	var result domain.Repository
	var platform string
	if err := row.Scan(&result.ID, &result.Name, &result.FullName, &result.URL, &platform, &result.APIToken); err != nil {
		return nil, fmt.Errorf("upsert repo: %w", err)
	}
	result.Platform = domain.GitPlatform{Name: platform}
	return &result, nil
}

func (r *PgxRepositoryStore) FindAll(ctx context.Context) ([]domain.Repository, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, name, full_name, url, platform, api_token FROM repositories ORDER BY full_name")
	if err != nil {
		return nil, fmt.Errorf("find all repos: %w", err)
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		var platform string
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.FullName, &repo.URL, &platform, &repo.APIToken); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		repo.Platform = domain.GitPlatform{Name: platform}
		repos = append(repos, repo)
	}
	return repos, nil
}
