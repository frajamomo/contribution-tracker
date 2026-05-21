package application

import (
	"context"
	"time"

	"contribution-tracker/internal/domain"
)

// --- Mock UserAccountRepository ---

type mockUserAccountRepo struct {
	accounts map[string]*domain.UserAccount
}

func newMockUserAccountRepo() *mockUserAccountRepo {
	return &mockUserAccountRepo{accounts: make(map[string]*domain.UserAccount)}
}

func (m *mockUserAccountRepo) FindByUsername(_ context.Context, username string) (*domain.UserAccount, error) {
	for _, a := range m.accounts {
		if a.Username == username {
			return a, nil
		}
	}
	return nil, NewNotFoundError("account not found")
}

func (m *mockUserAccountRepo) FindByID(_ context.Context, id string) (*domain.UserAccount, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, NewNotFoundError("account not found")
	}
	return a, nil
}

func (m *mockUserAccountRepo) Save(_ context.Context, account *domain.UserAccount) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockUserAccountRepo) FindAll(_ context.Context) ([]domain.UserAccount, error) {
	var result []domain.UserAccount
	for _, a := range m.accounts {
		result = append(result, *a)
	}
	return result, nil
}

// --- Mock UserRepository ---

type mockUserRepo struct {
	users          map[string]*domain.User
	byAccountID    map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:       make(map[string]*domain.User),
		byAccountID: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, NewNotFoundError("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) FindByIDs(_ context.Context, ids []string) ([]domain.User, error) {
	var result []domain.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, *u)
		}
	}
	return result, nil
}

func (m *mockUserRepo) FindByAccountID(_ context.Context, accountID string) (*domain.User, error) {
	u, ok := m.byAccountID[accountID]
	if !ok {
		return nil, NewNotFoundError("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) UpdatePlatformUsername(_ context.Context, userID string, platform domain.GitPlatform, username string) error {
	u, ok := m.users[userID]
	if !ok {
		return NewNotFoundError("user not found")
	}
	if u.PlatformUsernames == nil {
		u.PlatformUsernames = make(map[domain.GitPlatform]string)
	}
	u.PlatformUsernames[platform] = username
	return nil
}

func (m *mockUserRepo) FindAll(_ context.Context) ([]domain.User, error) {
	var result []domain.User
	for _, u := range m.users {
		result = append(result, *u)
	}
	return result, nil
}

// --- Mock TeamRepository ---

type mockTeamRepo struct {
	teams map[string]*domain.Team
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{teams: make(map[string]*domain.Team)}
}

func (m *mockTeamRepo) FindByID(_ context.Context, id string) (*domain.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return nil, NewNotFoundError("team not found")
	}
	return t, nil
}

func (m *mockTeamRepo) FindByMemberID(_ context.Context, memberID string) ([]domain.Team, error) {
	var result []domain.Team
	for _, t := range m.teams {
		for _, id := range t.MemberIDs {
			if id == memberID {
				result = append(result, *t)
				break
			}
		}
	}
	return result, nil
}

func (m *mockTeamRepo) Save(_ context.Context, team *domain.Team) error {
	m.teams[team.ID] = team
	return nil
}

func (m *mockTeamRepo) FindAll(_ context.Context) ([]domain.Team, error) {
	var result []domain.Team
	for _, t := range m.teams {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockTeamRepo) AddRepository(_ context.Context, teamID, repoID string) error {
	t, ok := m.teams[teamID]
	if !ok {
		return NewNotFoundError("team not found")
	}
	t.RepositoryIDs = append(t.RepositoryIDs, repoID)
	return nil
}

func (m *mockTeamRepo) RemoveRepository(_ context.Context, teamID, repoID string) error {
	t, ok := m.teams[teamID]
	if !ok {
		return NewNotFoundError("team not found")
	}
	var filtered []string
	for _, id := range t.RepositoryIDs {
		if id != repoID {
			filtered = append(filtered, id)
		}
	}
	t.RepositoryIDs = filtered
	return nil
}

// --- Mock RepositoryStore ---

type mockRepoStore struct {
	repos map[string]*domain.Repository
}

func newMockRepoStore() *mockRepoStore {
	return &mockRepoStore{repos: make(map[string]*domain.Repository)}
}

func (m *mockRepoStore) FindByID(_ context.Context, id string) (*domain.Repository, error) {
	r, ok := m.repos[id]
	if !ok {
		return nil, NewNotFoundError("repo not found")
	}
	return r, nil
}

func (m *mockRepoStore) FindByIDs(_ context.Context, ids []string) ([]domain.Repository, error) {
	var result []domain.Repository
	for _, id := range ids {
		if r, ok := m.repos[id]; ok {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRepoStore) Save(_ context.Context, repo *domain.Repository) error {
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockRepoStore) Upsert(_ context.Context, repo *domain.Repository) (*domain.Repository, error) {
	m.repos[repo.ID] = repo
	return repo, nil
}

func (m *mockRepoStore) FindAll(_ context.Context) ([]domain.Repository, error) {
	var result []domain.Repository
	for _, r := range m.repos {
		result = append(result, *r)
	}
	return result, nil
}

// --- Mock ConfigRepository ---

type mockConfigRepo struct {
	config map[string]string
}

func newMockConfigRepo() *mockConfigRepo {
	return &mockConfigRepo{config: make(map[string]string)}
}

func (m *mockConfigRepo) Get(_ context.Context, key string) (string, error) {
	v, ok := m.config[key]
	if !ok {
		return "", NewNotFoundError("config key not found")
	}
	return v, nil
}

func (m *mockConfigRepo) Set(_ context.Context, key, value string) error {
	m.config[key] = value
	return nil
}

func (m *mockConfigRepo) FindAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m.config {
		result[k] = v
	}
	return result, nil
}

// --- Mock BackupRepository ---

type mockBackupRepo struct {
	data *domain.BackupFile
}

func (m *mockBackupRepo) Export(_ context.Context) (*domain.BackupFile, error) {
	if m.data == nil {
		return &domain.BackupFile{}, nil
	}
	return m.data, nil
}

func (m *mockBackupRepo) Restore(_ context.Context, data *domain.BackupFile) error {
	m.data = data
	return nil
}

// --- Mock ActivityFetcher ---

type mockActivityFetcher struct {
	platforms  []domain.GitPlatform
	types      []domain.ActivityType
	activities []domain.Activity
}

func (m *mockActivityFetcher) GetSupportedPlatforms() []domain.GitPlatform { return m.platforms }
func (m *mockActivityFetcher) GetSupportedTypes() []domain.ActivityType    { return m.types }

func (m *mockActivityFetcher) FetchForUser(_ context.Context, _ string, _ domain.Repository,
	_, _ time.Time, _ []domain.ActivityType) ([]domain.Activity, error) {
	return m.activities, nil
}

func (m *mockActivityFetcher) SearchActivities(_ context.Context, _ string, _ []domain.Repository,
	_, _ time.Time, _ []domain.ActivityType) ([]domain.Activity, error) {
	return nil, nil
}

// --- Mock ActivityFetcher that also implements RepoDiscoverer ---

type mockFetcherWithDiscovery struct {
	mockActivityFetcher
	discoveredRepos []domain.Repository
}

func (m *mockFetcherWithDiscovery) DiscoverUserRepos(_ context.Context, _ string) ([]domain.Repository, error) {
	return m.discoveredRepos, nil
}

// --- Mock FetcherFactory ---

type mockFetcherFactory struct {
	fetcher ActivityFetcher
}

func (m *mockFetcherFactory) Build(_ string) ActivityFetcher {
	return m.fetcher
}
