package persistence

import (
	"context"
	"fmt"

	"contribution-tracker/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTeamRepo struct {
	pool *pgxpool.Pool
}

func NewPgxTeamRepo(pool *pgxpool.Pool) *PgxTeamRepo {
	return &PgxTeamRepo{pool: pool}
}

func (r *PgxTeamRepo) FindByID(ctx context.Context, id string) (*domain.Team, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, name FROM teams WHERE id = $1", id)

	var t domain.Team
	if err := row.Scan(&t.ID, &t.Name); err != nil {
		return nil, fmt.Errorf("find team by id: %w", err)
	}

	if err := r.loadRelations(ctx, &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (r *PgxTeamRepo) FindByMemberID(ctx context.Context, memberID string) ([]domain.Team, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name FROM teams t
		 JOIN team_members tm ON t.id = tm.team_id
		 WHERE tm.user_id = $1`, memberID)
	if err != nil {
		return nil, fmt.Errorf("find teams by member: %w", err)
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		teams = append(teams, t)
	}

	for i := range teams {
		if err := r.loadRelations(ctx, &teams[i]); err != nil {
			return nil, err
		}
	}

	return teams, nil
}

func (r *PgxTeamRepo) Save(ctx context.Context, team *domain.Team) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"INSERT INTO teams (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = $2",
		team.ID, team.Name)
	if err != nil {
		return fmt.Errorf("upsert team: %w", err)
	}

	_, err = tx.Exec(ctx, "DELETE FROM team_leaders WHERE team_id = $1", team.ID)
	if err != nil {
		return fmt.Errorf("clear leaders: %w", err)
	}
	for _, leaderID := range team.LeaderIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO team_leaders (team_id, user_id) VALUES ($1, $2)", team.ID, leaderID)
		if err != nil {
			return fmt.Errorf("insert leader: %w", err)
		}
	}

	_, err = tx.Exec(ctx, "DELETE FROM team_members WHERE team_id = $1", team.ID)
	if err != nil {
		return fmt.Errorf("clear members: %w", err)
	}
	for _, memberID := range team.MemberIDs {
		_, err = tx.Exec(ctx,
			"INSERT INTO team_members (team_id, user_id) VALUES ($1, $2)", team.ID, memberID)
		if err != nil {
			return fmt.Errorf("insert member: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PgxTeamRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM teams WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) FindAll(ctx context.Context) ([]domain.Team, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name FROM teams ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("find all teams: %w", err)
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		teams = append(teams, t)
	}

	for i := range teams {
		if err := r.loadRelations(ctx, &teams[i]); err != nil {
			return nil, err
		}
	}

	return teams, nil
}

func (r *PgxTeamRepo) AddRepository(ctx context.Context, teamID, repoID string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO team_repositories (team_id, repo_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		teamID, repoID)
	if err != nil {
		return fmt.Errorf("add repository to team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) RemoveRepository(ctx context.Context, teamID, repoID string) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM team_repositories WHERE team_id = $1 AND repo_id = $2",
		teamID, repoID)
	if err != nil {
		return fmt.Errorf("remove repository from team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) AddMember(ctx context.Context, teamID, userID string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		teamID, userID)
	if err != nil {
		return fmt.Errorf("add member to team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	var isLeader bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM team_leaders WHERE team_id = $1 AND user_id = $2)",
		teamID, userID).Scan(&isLeader)
	if err != nil {
		return fmt.Errorf("check leader status: %w", err)
	}
	if isLeader {
		return fmt.Errorf("cannot remove a team leader from members; remove leadership first")
	}

	_, err = r.pool.Exec(ctx,
		"DELETE FROM team_members WHERE team_id = $1 AND user_id = $2",
		teamID, userID)
	if err != nil {
		return fmt.Errorf("remove member from team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) AddLeader(ctx context.Context, teamID, userID string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO team_leaders (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		teamID, userID)
	if err != nil {
		return fmt.Errorf("add leader to team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) RemoveLeader(ctx context.Context, teamID, userID string) error {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM team_leaders WHERE team_id = $1", teamID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count leaders: %w", err)
	}
	if count <= 1 {
		return fmt.Errorf("cannot remove the last leader of a team")
	}

	_, err = r.pool.Exec(ctx,
		"DELETE FROM team_leaders WHERE team_id = $1 AND user_id = $2",
		teamID, userID)
	if err != nil {
		return fmt.Errorf("remove leader from team: %w", err)
	}
	return nil
}

func (r *PgxTeamRepo) loadRelations(ctx context.Context, t *domain.Team) error {
	leaderIDs, err := r.loadLeaderIDs(ctx, t.ID)
	if err != nil {
		return err
	}
	t.LeaderIDs = leaderIDs

	memberIDs, err := r.loadMemberIDs(ctx, t.ID)
	if err != nil {
		return err
	}
	t.MemberIDs = memberIDs

	repoIDs, err := r.loadRepositoryIDs(ctx, t.ID)
	if err != nil {
		return err
	}
	t.RepositoryIDs = repoIDs
	return nil
}

func (r *PgxTeamRepo) loadLeaderIDs(ctx context.Context, teamID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT user_id FROM team_leaders WHERE team_id = $1", teamID)
	if err != nil {
		return nil, fmt.Errorf("load leader ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan leader id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PgxTeamRepo) loadMemberIDs(ctx context.Context, teamID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT user_id FROM team_members WHERE team_id = $1", teamID)
	if err != nil {
		return nil, fmt.Errorf("load member ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan member id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PgxTeamRepo) loadRepositoryIDs(ctx context.Context, teamID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT repo_id FROM team_repositories WHERE team_id = $1", teamID)
	if err != nil {
		return nil, fmt.Errorf("load repository ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan repo id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
