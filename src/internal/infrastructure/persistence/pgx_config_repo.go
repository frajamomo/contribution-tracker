package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxConfigRepo struct {
	pool *pgxpool.Pool
}

func NewPgxConfigRepo(pool *pgxpool.Pool) *PgxConfigRepo {
	return &PgxConfigRepo{pool: pool}
}

func (r *PgxConfigRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx, "SELECT value FROM app_config WHERE key = $1", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get config %s: %w", key, err)
	}
	return value, nil
}

func (r *PgxConfigRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO app_config (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2",
		key, value)
	if err != nil {
		return fmt.Errorf("set config %s: %w", key, err)
	}
	return nil
}

func (r *PgxConfigRepo) FindAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, "SELECT key, value FROM app_config ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("find all config: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		result[key] = value
	}
	return result, nil
}
