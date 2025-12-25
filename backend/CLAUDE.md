# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Toggle is a multi-tenant feature flag management system built with Go. The backend uses Auth0 for authentication and PostgreSQL for data persistence. The architecture follows a strict domain-driven design with handler → service → repository layers.

## Common Commands

### Development
```bash
# Run the application
go run cmd/toggle/main.go

# Build the application
go build -o bin/toggle cmd/toggle/main.go

# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests for a specific package
go test ./internal/flags/...

# Run a single test
go test -run TestFlagService_Create ./internal/flags/

# Run all tests with verbose output
go test -v ./...

# Build and lint
go build ./... && golangci-lint run
```

### Database Migrations
Migrations are managed with Goose and located in `migrations/`. They run automatically in tests via testcontainers.

## Architecture

### Multi-Tenancy Model

The application implements **workspace-based multi-tenancy**:

- **Tenants** = Workspaces/Organizations with isolated data
- **Users** can belong to multiple tenants via `tenant_members` join table
- **Projects** are scoped to a single tenant
- **Flags** belong to projects and are transitively scoped to tenants

**Authentication Flow:**
1. Auth0 JWT validation extracts `sub` claim (Auth0 user ID)
2. First login auto-creates: User record → Default Tenant → Owner membership (atomic via UnitOfWork)
3. User selects active tenant via `X-Tenant-ID` header on subsequent requests
4. Tenant middleware validates membership and returns 403 if unauthorized

**Context Propagation:**
- `tenant_id`, `user_id`, `user_role`, `auth0_id` flow through `context.Context`
- Extract via `appContext.MustTenantID(ctx)` helpers in `internal/pkg/context/`
- Middleware injects values; services/repositories consume them

### Domain Structure

```
internal/
├── users/          # User identity and Auth0 sync
├── tenants/        # Workspace/organization management
├── projects/       # Feature flag projects (tenant-scoped)
├── flags/          # Feature flags with rollout rules
├── evaluation/     # Flag evaluation engine
├── middleware/     # Auth0 JWT + tenant scoping
├── routes/         # Central route registration
├── app/            # Server initialization
└── pkg/
    ├── context/    # Context helpers for tenant/user extraction
    ├── transaction/ # UnitOfWork pattern for atomic multi-step operations
    ├── errors/     # Domain error types
    ├── slugs/      # URL-safe slug generation with collision handling
    └── validation/ # Input validation utilities
```

### Three-Layer Pattern

Every domain follows this structure:

**Handler** (`handler.go`)
- Thin HTTP layer using Gin framework
- Extracts context (tenant_id, user_id) from middleware
- Binds JSON requests and validates input
- Delegates to service layer
- Maps errors to HTTP status codes (404, 403, 500)
- **Security:** Returns 404 for both "not found" AND "forbidden" to prevent ID enumeration

**Service** (`service.go`)
- Contains business logic and cross-resource validation
- Example: When creating a flag, validates that `project_id` belongs to the active `tenant_id`
- Orchestrates repository calls
- Uses UnitOfWork for multi-step atomic operations
- Logs important operations with structured logging

**Repository** (`repository.go`)
- Data access layer using sqlx
- **Critical:** ALL queries MUST include tenant scoping:
  - Direct: `WHERE tenant_id = $1` for tenant-owned resources (projects)
  - Relational: `INNER JOIN projects p ON p.id = f.project_id AND p.tenant_id = $1` for sub-resources (flags)
- Interface-based for testability
- Methods accept `context.Context` as first parameter
- Use `getDB(ctx)` helper to support transaction injection via context

**Executor Pattern** (in all repositories):
```go
func (r *postgresRepo) getDB(ctx context.Context) sqlx.ExtContext {
    if tx, ok := transaction.GetTx(ctx); ok {
        return tx  // Use transaction if present in context
    }
    return r.db  // Otherwise use connection pool
}
```

### UnitOfWork Pattern

For operations requiring atomicity across multiple repository calls:

```go
// Located in internal/pkg/transaction/uow.go
err := uow.RunInTransaction(ctx, func(txCtx context.Context) error {
    user, _ := userRepo.Create(txCtx, ...)
    tenant, _ := tenantRepo.Create(txCtx, ...)
    _ = tenantRepo.CreateMembership(txCtx, ...)
    return nil // Commits on success, rolls back on any error
})
```

The transaction is injected into context and automatically used by all repository calls within the closure.

## Testing Strategy

### Test Infrastructure

The codebase uses **"Sociable Testing"** with real PostgreSQL via testcontainers:

**Setup Pattern** (in `internal/testutil/`):
1. `TestMain` spins up a single PostgreSQL container for entire suite
2. Runs migrations once using Goose
3. Each test gets a transaction that auto-rolls back (keeps DB clean)

**Test Helper** (`testutil.WithTestDB`):
```go
func TestSomething(t *testing.T) {
    testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
        // Transaction automatically injected into context
        // All repository calls use this transaction
        // Auto-rollback after test completes
    })
}
```

**Fixture Helpers** (`internal/testutil/fixtures.go`):
- `CreateTenant(ctx, tx, name)` - Creates test tenant
- `CreateUser(ctx, tx, auth0ID)` - Creates test user
- `CreateProject(ctx, tx, tenantID, name)` - Creates test project
- `CreateFlag(ctx, tx, projectID, name)` - Creates test flag

### Test Types

1. **Unit Tests** (`*_test.go`): Mock-based handler tests focusing on HTTP behavior
2. **Sociable Tests** (`*_sociable_test.go`): Service + Repository + Real DB tested together
3. **E2E Tests** (`internal/e2e/`): Complete user journeys (signup → create project → create flag)
4. **Security Tests** (`internal/security/`): Cross-tenant access attempts and data leakage prevention

### Running Tests

Each domain has a `TestMain` that sets up the database:

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    _, err := testutil.SetupTestDatabase(ctx, "../../migrations")
    if err != nil {
        log.Fatalf("Failed to setup test database: %v", err)
    }
    code := m.Run()
    testutil.TeardownTestDatabase(ctx)
    os.Exit(code)
}
```

## Security Requirements

### Tenant Isolation (Zero-Tolerance Policy)

**Every repository query MUST enforce tenant scoping:**
- Direct resources: `WHERE tenant_id = $1`
- Transitive resources: `INNER JOIN` to parent with `WHERE parent.tenant_id = $1`
- UPDATE/DELETE operations: Include tenant_id in WHERE clause to prevent ID guessing

**Cross-Resource Validation:**
- Before creating a flag, verify the `project_id` belongs to the active `tenant_id`
- Services must validate ownership chains before delegating to repositories

**Error Handling for Security:**
- Return **404 Not Found** for both "doesn't exist" AND "forbidden" cases
- Prevents attackers from enumerating valid IDs
- Only return 403 for membership validation failures (tenant switching)

**Context Requirements:**
- Tenant-scoped routes REQUIRE `X-Tenant-ID` header
- Middleware validates user has membership in that tenant
- Missing tenant context should cause repository methods to fail explicitly

## Key Routes Structure

```
/api/v1
├── /health                    # Public health check
├── [Auth Middleware]
│   ├── /me                    # User-level (no tenant required)
│   │   ├── GET /tenants       # List user's workspaces
│   │   └── PUT /active-tenant # Switch active workspace
│   └── [Tenant Middleware]    # Requires X-Tenant-ID header
│       ├── POST /tenants      # Create new workspace
│       ├── /projects          # Tenant-scoped projects
│       ├── /flags             # Tenant-scoped feature flags
│       └── /flags/:id/evaluate # Flag evaluation
```

## Configuration

Environment variables are loaded via `godotenv` from `.env` file:
- `DATABASE_URL` - PostgreSQL connection string
- `AUTH0_DOMAIN` - Auth0 tenant domain
- `AUTH0_AUDIENCE` - Auth0 API audience
- `PORT` - Server port (default 8080)
- `SKIP_AUTH` - Set to "true" for local development without Auth0

Configuration is structured in `config/env.go`.

## Development Guidelines

### When Adding Features

1. **Read existing code first** - Understand patterns before modifying
2. **Follow the layer pattern** - Handler → Service → Repository
3. **Enforce tenant scoping** - ALL repository queries must filter by tenant_id
4. **Use interfaces** - Services depend on Repository interfaces, not concrete types
5. **Write sociable tests** - Test service + repository + DB together using testutil helpers
6. **Validate cross-resource ownership** - Check parent resources belong to active tenant
7. **Map errors for security** - Return 404 instead of 403 for resource access violations

### When Writing Tests

1. Use `testutil.WithTestDB(t, func(ctx, tx) {...})` for database tests
2. Create fixtures with `testutil.CreateX()` helpers
3. Each test runs in isolated transaction (auto-rollback)
4. Test security: Verify cross-tenant access returns 404
5. Test data isolation: Ensure queries only return tenant-scoped data

### Transaction Usage

- **Single-resource operations**: Repository methods handle this automatically
- **Multi-resource operations**: Use `uow.RunInTransaction()` to ensure atomicity
- **Tests**: Transaction automatically provided by `WithTestDB` helper

## Issue Tracking

This project uses **bd** (beads) for issue tracking. See `AGENTS.md` for the complete workflow, but key commands:

```bash
bd ready                                # Find available work
bd show <id>                            # View issue details
bd update <id> --status in_progress     # Claim work
bd close <id>                           # Complete work
bd sync                                 # Sync with git
```

**Session Completion Checklist:**
1. Create issues for remaining work
2. Run quality gates (tests, linters, builds) if code changed
3. Update issue status
4. **MANDATORY:** Push to remote with `git push` (work is NOT complete until pushed)
5. Verify with `git status` showing "up to date with origin"

## Project Status

**Current Phase:** Phase 3 - Security Audit & Data Scoping (IN PROGRESS)

**Completed:**
- Multi-tenant database schema with migrations
- Auth0 integration with automatic user onboarding
- Tenant switching via X-Tenant-ID header
- Service/Repository tenant scoping
- Sociable testing infrastructure with testcontainers

**In Progress:**
- Cross-tenant attack scenario tests
- Repository security hardening (ensuring ALL queries are scoped)
- Data leakage prevention validation

**Reference Documents:**
- `spec.md` - Full technical specification and phase roadmap
- `test_spec.md` - Testing strategy and patterns
- `AGENTS.md` - Issue tracking workflow with bd (beads)
