package application

import (
	"context"
	"time"

	"contribution-tracker/internal/domain"
)

// --- Service Ports (DIP — ADR-7) ---

type AuthServicePort interface {
	Login(ctx context.Context, username, password string) (*AuthToken, error)
	Validate(token string) (*AuthContext, error)
}

type ReportServicePort interface {
	GenerateReport(ctx context.Context, query ReportQuery, out chan<- ReportEvent)
}

type BackupServicePort interface {
	Export(ctx context.Context) (*domain.BackupFile, error)
	Restore(ctx context.Context, data *domain.BackupFile) error
}

// --- Persistence Ports ---

type UserAccountRepository interface {
	FindByUsername(ctx context.Context, username string) (*domain.UserAccount, error)
	FindByID(ctx context.Context, id string) (*domain.UserAccount, error)
	Save(ctx context.Context, account *domain.UserAccount) error
	Delete(ctx context.Context, userID string) error
	FindAll(ctx context.Context) ([]domain.UserAccount, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.User, error)
	FindByAccountID(ctx context.Context, accountID string) (*domain.User, error)
	UpdatePlatformUsername(ctx context.Context, userID string, platform domain.GitPlatform, username string) error
	FindAll(ctx context.Context) ([]domain.User, error)
	Save(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}

type TeamRepository interface {
	FindByID(ctx context.Context, id string) (*domain.Team, error)
	FindByMemberID(ctx context.Context, memberID string) ([]domain.Team, error)
	Save(ctx context.Context, team *domain.Team) error
	FindAll(ctx context.Context) ([]domain.Team, error)
	AddRepository(ctx context.Context, teamID, repoID string) error
	RemoveRepository(ctx context.Context, teamID, repoID string) error
	AddMember(ctx context.Context, teamID, userID string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	AddLeader(ctx context.Context, teamID, userID string) error
	RemoveLeader(ctx context.Context, teamID, userID string) error
	Delete(ctx context.Context, id string) error
}

type RepositoryStore interface {
	FindByID(ctx context.Context, id string) (*domain.Repository, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.Repository, error)
	Save(ctx context.Context, repo *domain.Repository) error
	Upsert(ctx context.Context, repo *domain.Repository) (*domain.Repository, error)
	FindAll(ctx context.Context) ([]domain.Repository, error)
}

type ConfigRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	FindAll(ctx context.Context) (map[string]string, error)
}

type BackupRepository interface {
	Export(ctx context.Context) (*domain.BackupFile, error)
	Restore(ctx context.Context, data *domain.BackupFile) error
}

// --- Fetcher Ports (Strategy — ADR-4, ISP — ADR-5) ---

type ActivityFetcher interface {
	GetSupportedPlatforms() []domain.GitPlatform
	GetSupportedTypes() []domain.ActivityType
	FetchForUser(ctx context.Context, username string, repo domain.Repository,
		since, until time.Time, types []domain.ActivityType) ([]domain.Activity, error)
	SearchActivities(ctx context.Context, username string, repos []domain.Repository,
		since, until time.Time, types []domain.ActivityType) ([]domain.Activity, error)
}

type RepoDiscoverer interface {
	DiscoverUserRepos(ctx context.Context, username string) ([]domain.Repository, error)
}

type FetcherFactory interface {
	Build(apiKey string) ActivityFetcher
}
