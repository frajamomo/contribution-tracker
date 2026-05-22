package presentation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

// --- Mock AuthServicePort ---

type mockAuthService struct {
	loginFn    func(ctx context.Context, username, password string) (*application.AuthToken, error)
	validateFn func(token string) (*application.AuthContext, error)
}

func (m *mockAuthService) Login(ctx context.Context, username, password string) (*application.AuthToken, error) {
	return m.loginFn(ctx, username, password)
}

func (m *mockAuthService) Validate(token string) (*application.AuthContext, error) {
	return m.validateFn(token)
}

// --- Mock ReportServicePort ---

type mockReportService struct {
	generateFn func(ctx context.Context, query application.ReportQuery, out chan<- application.ReportEvent)
}

func (m *mockReportService) GenerateReport(ctx context.Context, query application.ReportQuery, out chan<- application.ReportEvent) {
	m.generateFn(ctx, query, out)
}

// --- Mock BackupServicePort ---

type mockBackupService struct {
	exportFn  func(ctx context.Context) (*domain.BackupFile, error)
	restoreFn func(ctx context.Context, data *domain.BackupFile) error
}

func (m *mockBackupService) Export(ctx context.Context) (*domain.BackupFile, error) {
	return m.exportFn(ctx)
}

func (m *mockBackupService) Restore(ctx context.Context, data *domain.BackupFile) error {
	return m.restoreFn(ctx, data)
}

// --- Mock UserRepository ---

type mockUserRepo struct {
	findByIDFn              func(ctx context.Context, id string) (*domain.User, error)
	findByIDsFn             func(ctx context.Context, ids []string) ([]domain.User, error)
	findByAccountIDFn       func(ctx context.Context, accountID string) (*domain.User, error)
	updatePlatformUsernameFn func(ctx context.Context, userID string, platform domain.GitPlatform, username string) error
	findAllFn               func(ctx context.Context) ([]domain.User, error)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockUserRepo) FindByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ctx, ids)
	}
	return nil, nil
}

func (m *mockUserRepo) FindByAccountID(ctx context.Context, accountID string) (*domain.User, error) {
	return m.findByAccountIDFn(ctx, accountID)
}

func (m *mockUserRepo) UpdatePlatformUsername(ctx context.Context, userID string, platform domain.GitPlatform, username string) error {
	return m.updatePlatformUsernameFn(ctx, userID, platform, username)
}

func (m *mockUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	return m.findAllFn(ctx)
}

// --- Mock TeamRepository ---

type mockTeamRepo struct {
	findByIDFn       func(ctx context.Context, id string) (*domain.Team, error)
	findByMemberIDFn func(ctx context.Context, memberID string) ([]domain.Team, error)
	saveFn           func(ctx context.Context, team *domain.Team) error
	findAllFn        func(ctx context.Context) ([]domain.Team, error)
	addRepoFn        func(ctx context.Context, teamID, repoID string) error
	removeRepoFn     func(ctx context.Context, teamID, repoID string) error
}

func (m *mockTeamRepo) FindByID(ctx context.Context, id string) (*domain.Team, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockTeamRepo) FindByMemberID(ctx context.Context, memberID string) ([]domain.Team, error) {
	return m.findByMemberIDFn(ctx, memberID)
}

func (m *mockTeamRepo) Save(ctx context.Context, team *domain.Team) error {
	return m.saveFn(ctx, team)
}

func (m *mockTeamRepo) FindAll(ctx context.Context) ([]domain.Team, error) {
	return m.findAllFn(ctx)
}

func (m *mockTeamRepo) AddRepository(ctx context.Context, teamID, repoID string) error {
	return m.addRepoFn(ctx, teamID, repoID)
}

func (m *mockTeamRepo) RemoveRepository(ctx context.Context, teamID, repoID string) error {
	return m.removeRepoFn(ctx, teamID, repoID)
}

// --- Mock ConfigRepository ---

type mockConfigRepo struct {
	getFn     func(ctx context.Context, key string) (string, error)
	setFn     func(ctx context.Context, key, value string) error
	findAllFn func(ctx context.Context) (map[string]string, error)
}

func (m *mockConfigRepo) Get(ctx context.Context, key string) (string, error) {
	return m.getFn(ctx, key)
}

func (m *mockConfigRepo) Set(ctx context.Context, key, value string) error {
	return m.setFn(ctx, key, value)
}

func (m *mockConfigRepo) FindAll(ctx context.Context) (map[string]string, error) {
	return m.findAllFn(ctx)
}

// --- Mock RepositoryStore ---

type mockRepoStore struct {
	upsertFn    func(ctx context.Context, repo *domain.Repository) (*domain.Repository, error)
	findByIDsFn func(ctx context.Context, ids []string) ([]domain.Repository, error)
}

func (m *mockRepoStore) FindByID(ctx context.Context, id string) (*domain.Repository, error) {
	return nil, nil
}
func (m *mockRepoStore) FindByIDs(ctx context.Context, ids []string) ([]domain.Repository, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ctx, ids)
	}
	return nil, nil
}
func (m *mockRepoStore) Save(ctx context.Context, repo *domain.Repository) error { return nil }
func (m *mockRepoStore) Upsert(ctx context.Context, repo *domain.Repository) (*domain.Repository, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, repo)
	}
	return repo, nil
}
func (m *mockRepoStore) FindAll(ctx context.Context) ([]domain.Repository, error) {
	return nil, nil
}

// --- Helper to create an authenticated request ---

func newAuthenticatedRequest(method, url string, body *strings.Reader, authCtx *application.AuthContext) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, body)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(req.Context(), authContextKey, authCtx)
	return req.WithContext(ctx)
}
