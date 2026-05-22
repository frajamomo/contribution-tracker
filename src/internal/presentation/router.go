package presentation

import (
	"net/http"

	"contribution-tracker/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	authMiddleware *AuthMiddleware,
	authHandler *AuthHandler,
	reportHandler *ReportHandler,
	profileHandler *ProfileHandler,
	teamHandler *TeamHandler,
	backupHandler *BackupHandler,
	configHandler *ConfigHandler,
	adminHandler *AdminHandler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)

			r.Post("/reports/stream", reportHandler.StreamReport)
			r.Get("/profile", profileHandler.GetProfile)
			r.Put("/profile/platform-username", profileHandler.SetPlatformUsername)
			r.Get("/teams", teamHandler.ListTeams)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.RoleTeamLeader, domain.RoleAdmin))

				r.Post("/teams/{teamId}/repositories", teamHandler.AddRepository)
				r.Delete("/teams/{teamId}/repositories/{repoId}", teamHandler.RemoveRepository)
				r.Post("/teams/{teamId}/members", adminHandler.AddMember)
				r.Delete("/teams/{teamId}/members/{userId}", adminHandler.RemoveMember)
			})

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.RoleAdmin))

				r.Get("/admin/backup", backupHandler.Export)
				r.Post("/admin/restore", backupHandler.Restore)
				r.Get("/admin/config", configHandler.GetAll)
				r.Put("/admin/config", configHandler.Set)
				r.Get("/admin/users", adminHandler.ListUsers)
				r.Post("/admin/users", adminHandler.CreateUser)
				r.Delete("/admin/users/{userId}", adminHandler.DeleteUser)
				r.Post("/admin/teams", adminHandler.CreateTeam)
				r.Delete("/admin/teams/{teamId}", adminHandler.DeleteTeam)
			})
		})
	})

	r.Get("/*", frontendFileServer().ServeHTTP)

	return r
}
