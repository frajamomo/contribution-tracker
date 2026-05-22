# Contribution Tracker

A web application that aggregates and visualizes software contributions (commits, pull requests, issues, code reviews) from multiple git platforms for team reporting and analysis.

## Features

- **Multi-platform aggregation** — fetch contributions from GitHub and GitLab with a pluggable architecture for additional platforms
- **Role-based access control** — three roles (Team Member, Team Leader, Admin) with scoped visibility and permissions
- **Real-time streaming reports** — Server-Sent Events deliver per-member results progressively as data is fetched
- **Interactive report cards** — collapsible user cards with activity type filtering via clickable stat pills
- **Team management** — team leaders can add/remove repositories with automatic API token reuse across same-platform repos
- **Per-platform usernames** — users configure their GitHub/GitLab identity via a profile page
- **Backup & restore** — admin can export/import operational data (API tokens are base64-encoded in exports)
- **Weighted scoring** — configurable activity weights for quantitative contribution assessment

## Architecture

The project follows **Hexagonal Architecture** (Ports & Adapters) with four layers:

| Layer | Package | Responsibility |
|-------|---------|---------------|
| Domain | `internal/domain` | Entities, value objects, interfaces — zero external dependencies |
| Application | `internal/application` | Service ports, use cases, DTOs, events, fetcher registry |
| Infrastructure | `internal/infrastructure` | PostgreSQL adapters, GitHub/GitLab API clients and fetchers |
| Presentation | `internal/presentation` | HTTP handlers, JWT middleware, SSE writer, static frontend |

Key design decisions are documented as 13 ADRs in [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md).

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.25, Chi router |
| Database | PostgreSQL 16 |
| Frontend | Static SPA (HTML/CSS/JS), nginx |
| Auth | JWT (golang-jwt/v5), bcrypt |
| Persistence | pgx/v5 connection pool |
| Containers | Podman, podman-compose |
| Testing | testcontainers-go (integration), httptest (unit) |

## Quick Start

### Prerequisites

- [Podman](https://podman.io/) and [podman-compose](https://github.com/containers/podman-compose)
- Go 1.25+ (for local development)

### Run with containers

```bash
git clone https://github.com/frajamomo/contribution-tracker.git
cd contribution-tracker
make docker-build
make docker-up
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Demo Accounts

All demo accounts use password: **`secret`**

| Username | Display Name | Roles |
|----------|-------------|-------|
| `alice` | Alice Johnson | Team Member |
| `bob` | Bob Smith | Team Member |
| `carol` | Carol Davis | Team Member, Team Leader |
| `admin` | Administrator | Admin |

The demo includes an **Engineering** team with alice, bob, and carol as members.

## Development

### Makefile targets

```bash
make build              # Build Go binary
make test               # Run unit tests
make test-unit          # Unit tests (domain, application, infrastructure, presentation)
make test-integration   # PostgreSQL integration tests (requires Podman/Docker)
make test-all           # Run all tests
make run                # Build and run locally
make docker-build       # Build Podman containers
make docker-up          # Start containerized environment
make docker-down        # Stop containerized environment
make docker-logs        # Stream container logs
make migrate            # Run database migrations
make clean              # Clean build artifacts
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | Full PostgreSQL connection string (overrides individual DB_* vars) |
| `DB_USER` | `ctuser` | Database user |
| `DB_PASSWORD` | `ctpass` | Database password |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_NAME` | `contribution_tracker` | Database name |
| `JWT_SECRET` | `dev-secret-change-me` | JWT signing secret (**change in production**) |
| `PORT` | `8080` | API server port |
| `GITHUB_BASE_URL` | `https://api.github.com` | GitHub API base URL |
| `GITLAB_BASE_URL` | `https://gitlab.com` | GitLab API base URL |

## API Routes

```
POST   /api/auth/login                          — public
POST   /api/reports/stream                       — authenticated (SSE)
GET    /api/profile                              — authenticated
PUT    /api/profile/platform-username            — authenticated
GET    /api/teams                                — authenticated
POST   /api/teams/{teamId}/repositories          — Team Leader | Admin
DELETE /api/teams/{teamId}/repositories/{repoId} — Team Leader | Admin
GET    /api/admin/backup                         — Admin
POST   /api/admin/restore                        — Admin
GET    /api/admin/config                         — Admin
PUT    /api/admin/config                         — Admin
```

## Project Structure

```
src/
├── cmd/api/main.go                             — composition root
├── migrations/                                 — numbered SQL migrations
├── internal/
│   ├── domain/                                 — entities, value objects, interfaces
│   ├── application/                            — ports, services, DTOs, events, registry
│   ├── infrastructure/
│   │   ├── persistence/                        — pgx repository adapters, migrator
│   │   ├── github/                             — GitHub API client, fetcher, factory
│   │   └── gitlab/                             — GitLab API client, fetcher, factory
│   └── presentation/                           — handlers, middleware, DTOs, SSE, router
│       └── frontend/                           — index.html, styles.css, app.js
├── go.mod
└── go.sum
doc/
├── ARCHITECTURE.md                             — ADRs and design rationale
├── design_domain.puml                          — domain model specification
├── design_application.puml                     — port interfaces and services
├── design_infrastructure.puml                  — adapter specifications
├── design_presentation.puml                    — handler specifications
├── sequence_report.puml                        — SSE streaming flow
├── deployment.puml                             — container topology
└── mockup.html                                 — UI reference mockup
docker/
├── podman-compose.yml                          — service definitions
├── Dockerfile.api                              — multi-stage Go build
├── Dockerfile.frontend                         — nginx with static assets
├── nginx.conf                                  — reverse proxy + SSE support
└── .env                                        — default environment variables
Makefile                                        — build, test, docker targets
```

## Design Documentation

- [ARCHITECTURE.md](doc/ARCHITECTURE.md) — 13 Architecture Decision Records
- [Domain model](doc/design_domain.puml) — entities, value objects, interfaces (PlantUML)
- [Application layer](doc/design_application.puml) — ports, services, DTOs (PlantUML)
- [Infrastructure layer](doc/design_infrastructure.puml) — adapters, API clients (PlantUML)
- [Presentation layer](doc/design_presentation.puml) — handlers, route map (PlantUML)
- [Report sequence](doc/sequence_report.puml) — SSE streaming with RBAC scoping (PlantUML)
- [Deployment](doc/deployment.puml) — container topology (PlantUML)
- [UI mockup](doc/mockup.html) — frontend reference

## License

This project is for internal use. See repository for details.
