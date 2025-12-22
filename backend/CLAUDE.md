# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Toggle is a feature flag management system backend built with Go, Gin, PostgreSQL, and Auth0. It provides multi-tenant organization support with feature flag evaluation and management capabilities.

## Build & Development Commands

### Build
```bash
go build -o ./build/toggle ./cmd/toggle
```

### Run with hot-reload (Air)
```bash
air
# Uses .air.toml config, builds to ./build/toggle, logs errors to tmp/build-errors.log
```

### Run directly
```bash
go run ./cmd/toggle
```

### Linting
```bash
./bin/golangci-lint run
# Or if installed globally:
golangci-lint run
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/flags

# Run a single test
go test ./internal/flags -run TestFunctionName

# Run tests with verbose output
go test -v ./...
```

### Database Setup
```bash
# Start PostgreSQL and pgAdmin with Docker Compose
docker compose -f deployments/compose.yml up -d

# Run migrations (using goose)
goose -dir migrations postgres "postgres://admin:root@localhost:5432/test_db?sslmode=disable" up

# Create new migration
goose -dir migrations create migration_name sql
```

## Architecture

### Project Structure
```
cmd/toggle/         - Application entry point
config/             - Environment-based configuration
internal/
  app/              - Server setup, DB connection, custom logger
  middleware/       - Auth0 JWT authentication middleware
  routes/           - Route registration and dependency injection
  flags/            - Feature flag CRUD operations
  organizations/    - Multi-tenant organization management
  projects/         - Project scoping for flags
  users/            - User management with Auth0 sync
  evaluation/       - Feature flag evaluation engine (in development)
migrations/         - SQL schema migrations (goose)
deployments/        - Docker Compose configuration
```

### Layered Architecture Pattern

Each domain follows a consistent 3-layer pattern (Handler → Service → Repository):

1. **Handler** (`handler.go`): HTTP request/response handling, input validation, error mapping
2. **Service** (`service.go`): Business logic, validation, error handling with domain errors
3. **Repository** (`repository.go`): Database operations using sqlx

Example flow for flags domain:
- `flags/handler.go` - Gin handlers, binds JSON, returns HTTP responses
- `flags/service.go` - Flag validation, business rules
- `flags/repository.go` - PostgreSQL queries with sqlx

### Dependency Injection

All dependencies are wired in `internal/routes/routes.go`:
1. Repositories are created with DB connection
2. Services are created with repository dependencies
3. Handlers are created with service dependencies
4. Routes are registered via `RegisterRoutes()` method on each handler

### Authentication Flow

Auth middleware (`internal/middleware/auth.go`) runs on all `/api/v1` protected routes:

1. **Production mode** (`SKIP_AUTH=false`):
   - Validates Auth0 JWT token from Authorization header
   - Extracts `auth0_id` from token subject claim
   - Calls `userService.GetOrCreate()` to sync user with database
   - Creates user and organization if first login
   - Sets context values: `user_id`, `org_id`, `role`, `auth0_id`

2. **Development mode** (`SKIP_AUTH=true`):
   - Bypasses token validation
   - Creates/uses dev user with fixed credentials
   - Sets same context values for testing

Handlers extract user context via `c.Get("user_id")`, `c.Get("org_id")`, etc.

### Multi-Tenancy Model

Data hierarchy: **Organization → Users, Projects → Flags**

- Each Auth0 user gets an organization on first login (1:1 mapping currently)
- Users belong to one organization, have a role (member/admin)
- Projects are scoped to organizations via `organization_id`
- Flags are scoped to projects via `project_id`
- Cascading deletes enforced at database level

Key files:
- `migrations/20251218065959_feature_flag_init.sql` - Schema with foreign keys
- `internal/users/service.go:GetOrCreate()` - User/org creation logic
- `internal/middleware/auth.go` - Sets org context for requests

### Database Schema

- **organizations**: Multi-tenant isolation
- **users**: Auth0 ID mapping, org membership, roles
- **projects**: Org-scoped projects with client API keys
- **flags**: Feature flags with JSONB rules, project-scoped

Key patterns:
- UUID primary keys with `gen_random_uuid()`
- Foreign keys with `ON DELETE CASCADE`
- Indexes on foreign keys and auth0_id
- JSONB for flexible rule storage
- Timestamps: `created_at`, `updated_at`

### Configuration

Environment variables loaded via `config/env.go` (uses godotenv autoload):
- Router: `GIN_MODE` (debug/release)
- Backend: `BACKEND_PORT`
- Postgres: `POSTGRES_*` (user, password, name, host, port, sslmode)
- Auth0: `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`, `SKIP_AUTH`
- Goose: `GOOSE_*` (for migrations)

See `.env.template` for all variables. Copy to `.env` for local development.

### Testing Conventions

Tests use:
- `sqlmock` for repository layer (see `flags/repository_test.go`)
- Standard Go testing package
- Test files: `*_test.go` alongside source files
- Excluded from Air builds via `.air.toml`

### Logging

Custom structured logging using `slog` with `tint` handler:
- Logger initialized in `cmd/toggle/main.go`
- Custom Gin middleware in `internal/app/logger.go` for consistent log format
- All services receive logger via dependency injection
