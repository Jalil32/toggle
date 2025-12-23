# Technical Specification: Multi-Tenant Workspace Refactor

**Version:** 2.1 (Phase 1 Complete)
**Last Updated:** 2025-12-23

---

## ✅ Implementation Status

**Phase 1: COMPLETE** ✓ (2025-12-23)
- Database schema migrated to `tenants` with slug support
- All code updated to use `tenant_id` instead of `organization_id`
- Slug generation utility implemented
- Backward compatibility maintained in Gin context
- Build successful, migrations applied, onboarding flow tested

**Files Modified in Phase 1:**
- `migrations/20251218065959_feature_flag_init.sql` - Updated to use `tenants` table with slug
- `migrations/20251223030753_add_multi_tenant_support.sql` - Added tenant_members and tenant_invitations
- `internal/tenants/*` - Renamed from organizations, added slug support
- `internal/users/*` - Updated to use tenant_id, use tenantService
- `internal/projects/*` - Updated to use tenant_id
- `internal/middleware/auth.go` - Updated package name, use tenant_id (as org_id for compat)
- `internal/routes/routes.go` - Updated to use tenants package
- `internal/pkg/slugs/slugs.go` - New slug generation utility
- `go.mod` - Added github.com/gosimple/slug and github.com/google/uuid

**Next Phase:** Phase 2 - Multi-Tenant Context Middleware

---

## 0. Current State Analysis

### Architecture (Current - After Phase 1)

**Database Tables:**
- `tenants` - Workspace container (id, name, **slug**, created_at, updated_at) ✓ UPDATED
- `users` - User accounts (id, auth0_id, **tenant_id**, email, firstname, lastname, role, ...) ✓ UPDATED
- `projects` - Tenant-scoped (id, **tenant_id**, name, client_api_key, ...) ✓ UPDATED
- `flags` - Project-scoped (id, project_id, name, enabled, rules [JSONB], ...)
- `tenant_members` - Many-to-many users ↔ tenants (id, user_id, tenant_id, role, ...) ✓ NEW
- `tenant_invitations` - Invitation workflow (id, tenant_id, email, role, token, expires_at, ...) ✓ NEW

**Auth Flow (internal/middleware/auth.go):**
1. Auth0 JWT validated → `auth0_id` extracted
2. `userService.GetOrCreate()` called
3. On first login: Creates tenant `"{firstname} {lastname}'s Workspace"` with auto-generated slug + user as "owner"
4. Gin context set: `user_id`, `org_id` (maps to `tenant_id`), `role`, `auth0_id`
5. Handlers extract via `c.GetString("org_id")` (backward compat - still needs Phase 2 update)

**Current Limitation:**
- Users still belong to ONE tenant (hard-coded at creation)
- No workspace switching (requires Phase 2: X-Tenant-ID header + membership validation)
- `org_id` context key still used for backward compatibility

**Phase 1 Accomplishments:**
✓ Database schema fully migrated to tenant-based model
✓ Slug generation with collision handling
✓ All domains updated to use `tenant_id`
✓ Backward compatibility maintained
✓ Code compiles and runs successfully

**Remaining Issues (Phase 2+):**
- Need to implement `X-Tenant-ID` header-based tenant selection
- Need to validate tenant membership before allowing access
- Flags handler still needs org/project ownership validation (Phase 3)
- No transaction support yet (will add in user onboarding refactor)

**Tech Stack:**
- HTTP Framework: Gin
- Database: PostgreSQL with sqlx (raw SQL, parameterized queries)
- Auth: Auth0 with JWT middleware
- Logging: slog with tint handler
- Slug Generation: github.com/gosimple/slug ✓ NEW
- UUID Generation: github.com/google/uuid ✓ NEW
- No Redis (caching not implemented yet - optional for Phase 6)
- No transaction wrappers yet (planned for next phase)

---

## 1. Core Principles (To-Be)

1. **Multi-Workspace:** A user can belong to MULTIPLE tenants (workspaces/teams) with different roles
2. **Explicit Context:** Tenant context selected via `X-Tenant-ID` header (not inferred from user)
3. **Security First:** ALL queries MUST filter by `tenant_id` - zero tolerance for cross-tenant leakage
4. **Backward Compatible Naming:** Keep `organizations` table, extend it to support multi-tenancy (add slug)
5. **Atomic Operations:** Use sqlx transactions for multi-step operations (user onboarding, invite acceptance)

---

## 2. Database Schema Migration

### Phase 0: Add Slug to Organizations (Backward Compatible)

```sql
-- Migration: Add slug support to existing organizations
ALTER TABLE organizations
  ADD COLUMN slug VARCHAR(255);

-- Backfill slugs for existing orgs (one-time data migration)
-- Example: "John Doe's Organization" → "john-does-organization"

ALTER TABLE organizations
  ALTER COLUMN slug SET NOT NULL,
  ADD CONSTRAINT organizations_slug_unique UNIQUE (slug);

CREATE INDEX idx_organizations_slug ON organizations(slug);
```

### Phase 1: Multi-Tenant Membership Tables

```sql
-- Migration: Create tenant membership system
CREATE TABLE tenant_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, tenant_id)
);

CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_tenant ON tenant_members(tenant_id);

-- Backfill existing users into tenant_members
INSERT INTO tenant_members (user_id, tenant_id, role)
SELECT id, organization_id, role
FROM users
WHERE organization_id IS NOT NULL;
```

### Phase 2: Invitation System

```sql
-- Migration: Create invitation workflow
CREATE TABLE tenant_invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'member',
  invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token VARCHAR(255) UNIQUE NOT NULL, -- Cryptographically secure random token
  expires_at TIMESTAMPTZ NOT NULL, -- Default 48 hours from creation
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, email) -- Prevent duplicate invites
);

CREATE INDEX idx_tenant_invitations_token ON tenant_invitations(token);
CREATE INDEX idx_tenant_invitations_email ON tenant_invitations(email);
```

### Phase 3: Cleanup Users Table (Breaking Change - Do Last)

```sql
-- Migration: Remove denormalized fields from users
-- ⚠️ ONLY run after tenant_members is fully adopted and tested
ALTER TABLE users DROP COLUMN organization_id;
ALTER TABLE users DROP COLUMN role;
```

**Important:** Keep `organization_id` and `role` on users table during transition for rollback safety.

---

## 3. Implementation Phases

### Phase 0: Critical Security Fixes (IMMEDIATE)

**Priority:** HIGH - Fixes cross-tenant data leakage

**Files to Update:**
- `internal/flags/handler.go`
- `internal/flags/service.go`
- `internal/flags/repository.go`

**Tasks:**
1. **Fix `List()` Handler:**
   - Extract `org_id` from Gin context
   - Pass to `service.ListByOrgID(ctx, orgID)` (new method)
   - Repository: JOIN flags → projects → check `organization_id`

2. **Fix `Create()` Handler:**
   - Validate `project_id` belongs to user's org BEFORE creating flag
   - Add `projects.GetByID(ctx, projectID)` check
   - Return 403 if project not owned by user's org

3. **Fix `Get()`, `Update()`, `Delete()`:**
   - Add ownership validation: flag → project → org check
   - Return 404 if not found, 403 if found but wrong org

**Security Test:**
- User A cannot list/create/modify flags in User B's projects

---

### Phase 1: Database Schema + Slug System

**Tasks:**

**1.1 - Add Slug Generation Utility**
```go
// internal/pkg/slugs/slugs.go
package slugs

import (
    "fmt"
    "strings"
    "github.com/google/uuid"
    "github.com/gosimple/slug"
)

// Generate creates a URL-safe slug from input string
// If slug exists, appends random suffix
func Generate(input string) string {
    base := slug.Make(input)
    // Validate uniqueness in caller, append UUID suffix if needed
    return base
}

// WithFallback ensures uniqueness by appending UUID
func WithFallback(input string) string {
    base := slug.Make(input)
    suffix := uuid.New().String()[:8]
    return fmt.Sprintf("%s-%s", base, suffix)
}
```

**1.2 - Execute Migrations**
- Run migration to add `slug` column to organizations
- Backfill existing orgs with generated slugs
- Run migration to create `tenant_members` table
- Backfill from `users.organization_id` and `users.role`

**1.3 - Update Organization Service**
```go
// internal/organizations/service.go

// Create now generates slug and uses transaction
func (s *Service) Create(ctx context.Context, name string) (*Organization, error) {
    slug := slugs.Generate(name)

    // Check slug uniqueness, retry with suffix if needed
    exists, err := s.repo.SlugExists(ctx, slug)
    if err != nil {
        return nil, err
    }
    if exists {
        slug = slugs.WithFallback(name)
    }

    return s.repo.Create(ctx, name, slug)
}
```

**1.4 - Update User Onboarding (GetOrCreate)**
```go
// internal/users/service.go

func (s *Service) GetOrCreate(ctx context.Context, auth0ID, firstname, lastname string) (*User, error) {
    // Start transaction
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback() // Safe to call even after commit

    // Try to get existing user
    user, err := s.repo.GetByAuth0IDTx(ctx, tx, auth0ID)
    if err == nil {
        // User exists, check if has tenant membership
        hasMemberships, err := s.tenantMemberRepo.HasMembershipsTx(ctx, tx, user.ID)
        if err != nil {
            return nil, err
        }
        if hasMemberships {
            tx.Commit()
            return user, nil
        }
        // Fall through to create default tenant
    }

    // Create user if doesn't exist
    if errors.Is(err, sql.ErrNoRows) {
        user, err = s.repo.CreateTx(ctx, tx, auth0ID, email, firstname, lastname)
        if err != nil {
            return nil, fmt.Errorf("create user: %w", err)
        }
    }

    // Create default tenant workspace
    orgName := fmt.Sprintf("%s %s's Workspace", firstname, lastname)
    org, err := s.orgRepo.CreateTx(ctx, tx, orgName)
    if err != nil {
        return nil, fmt.Errorf("create organization: %w", err)
    }

    // Add user as owner of tenant
    err = s.tenantMemberRepo.CreateTx(ctx, tx, user.ID, org.ID, "owner")
    if err != nil {
        return nil, fmt.Errorf("create tenant membership: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit transaction: %w", err)
    }

    return user, nil
}
```

---

### Phase 2: Multi-Tenant Context Middleware

**Files to Create/Update:**
- `internal/middleware/tenant.go` (new)
- `internal/middleware/auth.go` (modify)
- `internal/pkg/context/tenant.go` (new - context helpers)

**Task 2.1 - Context Helpers**
```go
// internal/pkg/context/tenant.go
package context

import (
    "context"
    "errors"
)

type contextKey string

const (
    tenantIDKey contextKey = "tenant_id"
    userRoleKey contextKey = "user_role"
)

var ErrNoTenantContext = errors.New("tenant context not found")

func WithTenant(ctx context.Context, tenantID string, role string) context.Context {
    ctx = context.WithValue(ctx, tenantIDKey, tenantID)
    ctx = context.WithValue(ctx, userRoleKey, role)
    return ctx
}

func TenantID(ctx context.Context) (string, error) {
    val := ctx.Value(tenantIDKey)
    if val == nil {
        return "", ErrNoTenantContext
    }
    tenantID, ok := val.(string)
    if !ok {
        return "", ErrNoTenantContext
    }
    return tenantID, nil
}

func UserRole(ctx context.Context) string {
    val := ctx.Value(userRoleKey)
    if val == nil {
        return ""
    }
    role, _ := val.(string)
    return role
}
```

**Task 2.2 - Tenant Middleware**
```go
// internal/middleware/tenant.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    tenantCtx "toggle/internal/pkg/context"
)

type TenantMembershipService interface {
    // GetMembership returns (role, error)
    // Returns nil if user is not a member of tenant
    GetMembership(ctx context.Context, userID, tenantID string) (string, error)
}

func TenantMiddleware(memberService TenantMembershipService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract user_id from previous auth middleware
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
            c.Abort()
            return
        }

        // Extract tenant_id from header
        tenantID := c.GetHeader("X-Tenant-ID")
        if tenantID == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
            c.Abort()
            return
        }

        // Verify user has access to this tenant
        role, err := memberService.GetMembership(c.Request.Context(), userID.(string), tenantID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify tenant access"})
            c.Abort()
            return
        }

        if role == "" {
            c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this tenant"})
            c.Abort()
            return
        }

        // Inject tenant context into request context
        ctx := tenantCtx.WithTenant(c.Request.Context(), tenantID, role)
        c.Request = c.Request.WithContext(ctx)

        // Also set in Gin context for backward compatibility
        c.Set("tenant_id", tenantID)
        c.Set("user_role", role)

        c.Next()
    }
}
```

**Task 2.3 - Update Route Registration**
```go
// internal/routes/routes.go

func SetupRoutes(r *gin.Engine, db *sqlx.DB, logger *slog.Logger, authMiddleware, tenantMiddleware gin.HandlerFunc) {
    api := r.Group("/api/v1")

    // Public routes
    api.GET("/health", healthHandler)

    // Protected routes (require auth + tenant context)
    protected := api.Group("")
    protected.Use(authMiddleware)
    protected.Use(tenantMiddleware) // NEW: Validates tenant access

    // Register domain handlers
    flagHandler.RegisterRoutes(protected)
    projectHandler.RegisterRoutes(protected)
    // ...
}
```

---

### Phase 3: Repository & Service Tenant Scoping

**Goal:** Ensure ALL database queries are scoped by tenant_id from context

**Task 3.1 - Standardize Context Usage**

Update ALL repository methods to accept `context.Context` as first parameter:

```go
// Before (flags/repository.go)
func (r *Repository) List() ([]*Flag, error)

// After
func (r *Repository) List(ctx context.Context, tenantID string) ([]*Flag, error)
```

**Task 3.2 - Update SQL Queries**

Add tenant filtering to all queries via JOIN or direct column:

```go
// flags/repository.go

func (r *Repository) List(ctx context.Context, tenantID string) ([]*Flag, error) {
    query := `
        SELECT f.*
        FROM flags f
        INNER JOIN projects p ON f.project_id = p.id
        WHERE p.organization_id = $1
        ORDER BY f.created_at DESC
    `

    var flags []*Flag
    err := r.db.SelectContext(ctx, &flags, query, tenantID)
    return flags, err
}

func (r *Repository) GetByID(ctx context.Context, flagID, tenantID string) (*Flag, error) {
    query := `
        SELECT f.*
        FROM flags f
        INNER JOIN projects p ON f.project_id = p.id
        WHERE f.id = $1 AND p.organization_id = $2
    `

    var flag Flag
    err := r.db.GetContext(ctx, &flag, query, flagID, tenantID)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrFlagNotFound
    }
    return &flag, err
}
```

**Task 3.3 - Update Handlers to Extract Tenant**

```go
// flags/handler.go

func (h *Handler) List(c *gin.Context) {
    tenantID, err := tenantCtx.TenantID(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context missing"})
        return
    }

    flags, err := h.service.List(c.Request.Context(), tenantID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, flags)
}
```

**Task 3.4 - Add Tenant Validation Utilities**

```go
// internal/pkg/validation/tenant.go

// ValidateProjectOwnership checks if project belongs to tenant
func ValidateProjectOwnership(ctx context.Context, db *sqlx.DB, projectID, tenantID string) error {
    var orgID string
    err := db.GetContext(ctx, &orgID,
        "SELECT organization_id FROM projects WHERE id = $1", projectID)

    if errors.Is(err, sql.ErrNoRows) {
        return errors.New("project not found")
    }
    if err != nil {
        return err
    }

    if orgID != tenantID {
        return errors.New("project does not belong to tenant")
    }

    return nil
}
```

---

### Phase 4: Invitation System

**Task 4.1 - Create Invitations Domain**

```go
// internal/invitations/repository.go
type Repository struct {
    db *sqlx.DB
}

func (r *Repository) Create(ctx context.Context, tenantID, email, role, invitedBy, token string, expiresAt time.Time) error {
    query := `
        INSERT INTO tenant_invitations (tenant_id, email, role, invited_by, token, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (tenant_id, email) DO UPDATE
        SET token = $5, expires_at = $6, created_at = NOW()
    `
    _, err := r.db.ExecContext(ctx, query, tenantID, email, role, invitedBy, token, expiresAt)
    return err
}

func (r *Repository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
    query := `SELECT * FROM tenant_invitations WHERE token = $1 AND accepted_at IS NULL`
    var inv Invitation
    err := r.db.GetContext(ctx, &inv, query, token)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrInvitationNotFound
    }
    return &inv, err
}

func (r *Repository) MarkAccepted(ctx context.Context, id string) error {
    query := `UPDATE tenant_invitations SET accepted_at = NOW() WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}
```

```go
// internal/invitations/service.go
type Service struct {
    repo              *Repository
    tenantMemberRepo  *tenantmembers.Repository
    db                *sqlx.DB
}

func (s *Service) Create(ctx context.Context, tenantID, email, role, invitedBy string) (string, error) {
    // Generate secure random token
    token := generateSecureToken() // Use crypto/rand
    expiresAt := time.Now().Add(48 * time.Hour)

    err := s.repo.Create(ctx, tenantID, email, role, invitedBy, token, expiresAt)
    return token, err
}

func (s *Service) Accept(ctx context.Context, token string, userID string) error {
    // Start transaction
    tx, err := s.db.BeginTxx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Get invitation
    inv, err := s.repo.GetByTokenTx(ctx, tx, token)
    if err != nil {
        return err
    }

    // Validate expiration
    if time.Now().After(inv.ExpiresAt) {
        return ErrInvitationExpired
    }

    // Create tenant membership
    err = s.tenantMemberRepo.CreateTx(ctx, tx, userID, inv.TenantID, inv.Role)
    if err != nil {
        return err
    }

    // Mark invitation as accepted
    err = s.repo.MarkAcceptedTx(ctx, tx, inv.ID)
    if err != nil {
        return err
    }

    return tx.Commit()
}

func generateSecureToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

**Task 4.2 - Add Invitation Endpoints**

```go
// internal/invitations/handler.go

// POST /api/v1/tenants/:tenant_id/invitations
func (h *Handler) Create(c *gin.Context) {
    tenantID := c.Param("tenant_id")
    userID := c.GetString("user_id")

    var req struct {
        Email string `json:"email" binding:"required,email"`
        Role  string `json:"role" binding:"required,oneof=member admin"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    token, err := h.service.Create(c.Request.Context(), tenantID, req.Email, req.Role, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // TODO: Send email with invitation link containing token
    inviteURL := fmt.Sprintf("https://app.toggle.dev/invite?token=%s", token)

    c.JSON(http.StatusCreated, gin.H{
        "token": token,
        "invite_url": inviteURL,
    })
}

// POST /api/v1/invitations/:token/accept
func (h *Handler) Accept(c *gin.Context) {
    token := c.Param("token")
    userID := c.GetString("user_id")

    err := h.service.Accept(c.Request.Context(), token, userID)
    if err != nil {
        if errors.Is(err, ErrInvitationExpired) {
            c.JSON(http.StatusGone, gin.H{"error": "Invitation expired"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted"})
}
```

---

### Phase 5: API Refactoring

**Task 5.1 - Add Tenant Listing Endpoint**

```go
// internal/tenantmembers/service.go

func (s *Service) ListUserTenants(ctx context.Context, userID string) ([]*TenantMembership, error) {
    query := `
        SELECT
            tm.tenant_id,
            tm.role,
            o.name as tenant_name,
            o.slug as tenant_slug
        FROM tenant_members tm
        INNER JOIN organizations o ON tm.tenant_id = o.id
        WHERE tm.user_id = $1
        ORDER BY tm.created_at ASC
    `

    var memberships []*TenantMembership
    err := s.db.SelectContext(ctx, &memberships, query, userID)
    return memberships, err
}
```

```go
// internal/tenantmembers/handler.go

// GET /api/v1/tenants (or /api/v1/me/tenants)
func (h *Handler) ListMyTenants(c *gin.Context) {
    userID := c.GetString("user_id")

    tenants, err := h.service.ListUserTenants(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, tenants)
}
```

**Task 5.2 - Update Organization Routes**

Current: `GET /api/v1/organization` (singular - assumes single org)
New: `GET /api/v1/tenants/:tenant_id` (explicit tenant selection)

OR keep backward compatible:
- `GET /api/v1/organization` - Returns current tenant (from X-Tenant-ID header)
- `GET /api/v1/tenants` - Lists all user's tenants

---

## 4. Security & Data Integrity Requirements

### Non-Negotiable Rules

1. **Tenant Filtering:** Every query MUST include tenant_id filter via WHERE clause or JOIN
2. **Header Validation:** `X-Tenant-ID` header MUST be present on all protected routes
3. **Membership Verification:** Middleware MUST verify user belongs to requested tenant
4. **403 on Unauthorized:** Return 403 (not 404) if user tries to access tenant they don't belong to
5. **Transaction Atomicity:** Multi-step operations (user onboarding, invite acceptance) MUST use sqlx transactions
6. **Slug Uniqueness:** Organization slugs MUST be unique (database constraint + application check)

### Testing Requirements

**Security Tests (Critical):**
```go
// Test: User cannot access another tenant's data
func TestCrossTenantIsolation(t *testing.T) {
    // Create User A in Tenant A
    // Create User B in Tenant B
    // User A tries to access Tenant B with X-Tenant-ID: tenant_b_id
    // Expect: 403 Forbidden
}

// Test: Missing X-Tenant-ID header
func TestMissingTenantHeader(t *testing.T) {
    // Make request without X-Tenant-ID header
    // Expect: 400 Bad Request
}

// Test: Invitation acceptance is atomic
func TestInvitationAtomicity(t *testing.T) {
    // Simulate failure after creating membership but before marking accepted
    // Expect: Rollback, no partial state
}
```

---

## 5. Optional Enhancements (Future Phases)

### Redis Caching (Phase 6 - Optional)

**Goal:** Reduce database load for membership verification

```go
// internal/cache/redis.go
type MembershipCache struct {
    client *redis.Client
}

func (c *MembershipCache) GetMembership(userID, tenantID string) (string, error) {
    key := fmt.Sprintf("user:mems:%s:%s", userID, tenantID)
    return c.client.Get(ctx, key).Result()
}

func (c *MembershipCache) SetMembership(userID, tenantID, role string, ttl time.Duration) error {
    key := fmt.Sprintf("user:mems:%s:%s", userID, tenantID)
    return c.client.Set(ctx, key, role, ttl).Err()
}

func (c *MembershipCache) Invalidate(userID string) error {
    pattern := fmt.Sprintf("user:mems:%s:*", userID)
    // Delete all keys matching pattern
}
```

**Update Tenant Middleware:**
```go
// Check cache first, fallback to database
role, err := cache.GetMembership(userID, tenantID)
if err == redis.Nil {
    // Cache miss - query database
    role, err = memberService.GetMembership(ctx, userID, tenantID)
    if err == nil && role != "" {
        cache.SetMembership(userID, tenantID, role, 15*time.Minute)
    }
}
```

**Invalidation Points:**
- When user joins/leaves tenant → invalidate user's cache
- When user role changes → invalidate user's cache

---

## 6. Migration Checklist

### Pre-Migration
- [x] Review current database schema ✓
- [x] Backup production database (N/A - pre-production) ✓
- [x] Test migrations on staging environment (tested locally) ✓
- [ ] Fix critical flags handler security issues (Phase 0) - **Deferred to Phase 3**

### Phase 1: Schema + Slug System ✅ COMPLETE (2025-12-23)
- [x] Rename organizations table to tenants ✓
- [x] Add slug column to tenants table ✓
- [x] Update all foreign keys from organization_id to tenant_id ✓
- [x] Create tenant_members table ✓
- [x] Create tenant_invitations table ✓
- [x] Implement slug generation utility (internal/pkg/slugs) ✓
- [x] Update tenants service with slug generation and collision handling ✓
- [x] Rename internal/organizations to internal/tenants ✓
- [x] Update all domain models (users, projects, flags) to use tenant_id ✓
- [x] Update all repositories to use tenant_id in queries ✓
- [x] Update all services to use tenant_id ✓
- [x] Update routes to use tenants package ✓
- [x] Update middleware package name and function ✓
- [x] Test: Build successful ✓
- [x] Test: Migrations apply cleanly ✓
- [x] Test: User creation creates default tenant with slug ✓
- [x] Test: Project creation uses tenant_id ✓
- [ ] Add transaction support to user onboarding - **Deferred to Phase 2**

### Phase 2: Middleware
- [ ] Create context helpers (internal/pkg/context/tenant.go)
- [ ] Implement tenant membership service
- [ ] Create tenant middleware
- [ ] Update route registration to use tenant middleware
- [ ] Test: 403 when accessing unauthorized tenant
- [ ] Test: 400 when X-Tenant-ID header missing

### Phase 3: Repository Scoping
- [ ] Update all repository methods to accept context
- [ ] Add tenant_id filtering to all SQL queries
- [ ] Update all handlers to extract tenant from context
- [ ] Test: Cross-tenant data leakage prevented
- [ ] Test: Query performance with tenant filtering

### Phase 4: Invitations
- [ ] Create invitations domain (repository, service, handler)
- [ ] Implement token generation (crypto/rand)
- [ ] Add invitation endpoints
- [ ] Implement email sending (optional)
- [ ] Test: Invitation acceptance creates membership atomically
- [ ] Test: Expired invitations rejected

### Phase 5: API Updates
- [ ] Add tenant listing endpoint (GET /api/v1/tenants)
- [ ] Update frontend to call tenant listing on login
- [ ] Implement tenant switcher UI component
- [ ] Update all API calls to include X-Tenant-ID header

### Phase 6: Cleanup (Breaking Change)
- [ ] Remove organization_id from users table
- [ ] Remove role from users table
- [ ] Update all references to use tenant_members
- [ ] Migration complete!

---

## 7. Acceptance Criteria

**Phase 1 - Complete:**
1. ✅ Tenant slugs are unique and URL-safe
2. ✅ User onboarding creates default tenant with slug
3. ✅ Database schema uses tenant_id throughout
4. ✅ All domain models updated to use tenant_id
5. ✅ Backward compatibility maintained (org_id in context)
6. ✅ Build successful, no compilation errors
7. ✅ Migrations apply cleanly from scratch

**Phase 2 - Pending:**
1. ⏳ X-Tenant-ID header validation in middleware
2. ⏳ Tenant membership verification before access
3. ⏳ 403 returned when accessing unauthorized tenant
4. ⏳ 400 returned when X-Tenant-ID header missing
5. ⏳ Context.Context propagated through all layers

**Phase 3 - Pending:**
1. ⏳ All queries include tenant_id filtering
2. ⏳ Cross-tenant data leakage prevented (tested via security suite)
3. ⏳ Flags handler has proper ownership validation

**Phase 4 - Pending:**
1. ⏳ Invitation acceptance is atomic (transaction-wrapped)
2. ⏳ User can belong to multiple tenants with different roles

**Phase 5 - Pending:**
1. ⏳ User can switch between tenants via UI (sends different X-Tenant-ID)
2. ⏳ Tenant listing endpoint available

**Performance (Phase 6 - Optional):**
- Membership verification < 50ms (without Redis)
- Optional: < 5ms with Redis cache

**Current Backward Compatibility:**
- ✅ Handlers still use c.GetString("org_id")
- ✅ Middleware sets org_id mapping to tenant_id
- ✅ Database schema allows clean rollback via goose down

---

## 8. Open Questions & Decisions

### Terminology
**Decision:** Keep `organizations` table name, use "tenant" in code/docs for clarity
**Rationale:** Avoids database rename migration, less risky

### Default Tenant Name
**Current:** "{Firstname} {Lastname}'s Organization"
**Proposed:** "{Firstname} {Lastname}'s Workspace"
**Decision:** Update in Phase 1

### Slug Collision Strategy
**Option A:** Append random suffix (e.g., `acme-corp-a3b2c4d1`)
**Option B:** Append sequential number (e.g., `acme-corp-2`)
**Decision:** Option A (UUID suffix) for security and simplicity

### Public vs Private Invitations
**Current Spec:** Email-based invitations (anyone with link can accept)
**Alternative:** User must be logged in AND email matches
**Decision:** Logged-in user only, email validation on acceptance

### Redis Dependency
**Decision:** Optional, implement in Phase 6 after core multi-tenancy works
**Rationale:** Don't block on infrastructure setup, measure need first

---

## Next Steps

✅ ~~1. **Review this spec** with team for alignment~~ (Complete)
✅ ~~2. **Start Phase 1:** Database migrations + slug system~~ (Complete)
✅ ~~3. **Test incrementally:** Each phase should be deployable and testable independently~~ (Complete)

**Current Priority:**
1. **Start Phase 2:** Multi-Tenant Context Middleware
   - Create context helpers (internal/pkg/context/tenant.go)
   - Implement tenant membership service (internal/tenantmembers)
   - Create tenant middleware to validate X-Tenant-ID header
   - Update route registration to use tenant middleware

2. **Start Phase 3:** Repository & Service Scoping
   - Update all handlers to extract tenant from request context
   - Add tenant_id filtering to flags queries (security fix)
   - Test cross-tenant data leakage prevention

3. **Monitor performance:** Measure query times with tenant filtering

---

**Document Version Control:**
- v1.0: Initial spec (original)
- v2.0: Updated to reflect current codebase architecture (2025-12-23)
- v2.1: Phase 1 complete - updated with implementation status (2025-12-23)
