package persistence

import (
	"context"
	"fmt"

	"contribution-tracker/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxUserAccountRepo struct {
	pool *pgxpool.Pool
}

func NewPgxUserAccountRepo(pool *pgxpool.Pool) *PgxUserAccountRepo {
	return &PgxUserAccountRepo{pool: pool}
}

func (r *PgxUserAccountRepo) FindByUsername(ctx context.Context, username string) (*domain.UserAccount, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, username, password_hash, roles, user_id FROM user_accounts WHERE username = $1",
		username)
	return r.scanAccount(row)
}

func (r *PgxUserAccountRepo) FindByID(ctx context.Context, id string) (*domain.UserAccount, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, username, password_hash, roles, user_id FROM user_accounts WHERE id = $1",
		id)
	return r.scanAccount(row)
}

func (r *PgxUserAccountRepo) Save(ctx context.Context, account *domain.UserAccount) error {
	roleStrings := rolesToStrings(account.Roles)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_accounts (id, username, password_hash, roles, user_id)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET username=$2, password_hash=$3, roles=$4, user_id=$5`,
		account.ID, account.Username, account.PasswordHash, roleStrings, account.UserID)
	if err != nil {
		return fmt.Errorf("save account: %w", err)
	}
	return nil
}

func (r *PgxUserAccountRepo) FindAll(ctx context.Context) ([]domain.UserAccount, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, username, password_hash, roles, user_id FROM user_accounts ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("find all accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.UserAccount
	for rows.Next() {
		a, err := r.scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *a)
	}
	return accounts, nil
}

func (r *PgxUserAccountRepo) scanAccount(row pgx.Row) (*domain.UserAccount, error) {
	var a domain.UserAccount
	var roleStrings []string
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &roleStrings, &a.UserID)
	if err != nil {
		return nil, fmt.Errorf("scan account: %w", err)
	}
	a.Roles = stringsToRoles(roleStrings)
	return &a, nil
}

func (r *PgxUserAccountRepo) scanAccountFromRows(rows pgx.Rows) (*domain.UserAccount, error) {
	var a domain.UserAccount
	var roleStrings []string
	err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &roleStrings, &a.UserID)
	if err != nil {
		return nil, fmt.Errorf("scan account: %w", err)
	}
	a.Roles = stringsToRoles(roleStrings)
	return &a, nil
}

func rolesToStrings(roles map[domain.Role]bool) []string {
	var result []string
	for r := range roles {
		result = append(result, string(r))
	}
	return result
}

func stringsToRoles(ss []string) map[domain.Role]bool {
	roles := make(map[domain.Role]bool, len(ss))
	for _, s := range ss {
		roles[domain.Role(s)] = true
	}
	return roles
}
