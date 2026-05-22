package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
	"contribution-tracker/internal/infrastructure/github"
	"contribution-tracker/internal/infrastructure/gitlab"
	"contribution-tracker/internal/infrastructure/persistence"
	"contribution-tracker/internal/presentation"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			envOrDefault("DB_USER", "ctuser"),
			envOrDefault("DB_PASSWORD", "ctpass"),
			envOrDefault("DB_HOST", "localhost"),
			envOrDefault("DB_PORT", "5432"),
			envOrDefault("DB_NAME", "contribution_tracker"),
		)
	}

	pool, err := persistence.NewPool(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrationsDir := envOrDefault("MIGRATIONS_DIR", "migrations")
	if err := persistence.RunMigrations(ctx, pool, migrationsDir); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	userAccountRepo := persistence.NewPgxUserAccountRepo(pool)
	userRepo := persistence.NewPgxUserRepo(pool)
	teamRepo := persistence.NewPgxTeamRepo(pool)
	repoStore := persistence.NewPgxRepositoryStore(pool)
	configRepo := persistence.NewPgxConfigRepo(pool)
	backupRepo := persistence.NewPgxBackupRepo(pool)

	jwtSecret := []byte(envOrDefault("JWT_SECRET", "dev-secret-change-me"))

	registry := application.NewActivityFetcherRegistry()
	ghBaseURL := envOrDefault("GITHUB_BASE_URL", "https://api.github.com")
	glBaseURL := envOrDefault("GITLAB_BASE_URL", "https://gitlab.com")
	registry.Register(domain.PlatformGitHub, github.NewFetcherFactory(ghBaseURL))
	registry.Register(domain.PlatformGitLab, gitlab.NewFetcherFactory(glBaseURL))

	authService := application.NewAuthService(userAccountRepo, userRepo, teamRepo, jwtSecret)
	reportService := application.NewReportService(userRepo, teamRepo, repoStore, registry)
	backupService := application.NewBackupService(backupRepo)

	authMiddleware := presentation.NewAuthMiddleware(authService)
	authHandler := presentation.NewAuthHandler(authService, userRepo)
	reportHandler := presentation.NewReportHandler(reportService)
	profileHandler := presentation.NewProfileHandler(userRepo)
	teamHandler := presentation.NewTeamHandler(teamRepo, repoStore, userRepo)
	backupHandler := presentation.NewBackupHandler(backupService)
	configHandler := presentation.NewConfigHandler(configRepo)

	router := presentation.NewRouter(
		authMiddleware,
		authHandler,
		reportHandler,
		profileHandler,
		teamHandler,
		backupHandler,
		configHandler,
	)

	port := envOrDefault("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
