package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jalil32/toggle/internal/middleware"
	pkgcontext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, err := testutil.SetupTestDatabase(ctx, "../../migrations")
	if err != nil {
		panic(err)
	}

	code := m.Run()

	if err := testutil.TeardownTestDatabase(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

// setupTestRouter creates a Gin router with tenant middleware for testing
func setupTestRouter(tenantRepo tenants.Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Apply tenant middleware
	router.Use(middleware.Tenant(tenantRepo, logger))

	// Test endpoint that returns tenant info from context
	router.GET("/test", func(c *gin.Context) {
		tenantID, err := pkgcontext.TenantID(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no tenant in context"})
			return
		}

		role := pkgcontext.UserRole(c.Request.Context())

		c.JSON(http.StatusOK, gin.H{
			"tenant_id": tenantID,
			"role":      role,
		})
	})

	return router
}

// TestTenantMiddleware_ValidTenantID_Success tests that a valid X-Tenant-ID header
// for a tenant the user belongs to succeeds and injects tenant context
func TestTenantMiddleware_ValidTenantID_Success(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create user, tenant, and membership - COMMIT so middleware can see it
		setupTx, err := db.Beginx()
		require.NoError(t, err)
		user := testutil.CreateUser(t, setupTx, "auth0|test-user", "test@example.com", "Test", "User")
		tenant := testutil.CreateTenant(t, setupTx, "Test Tenant", "test-tenant")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenant.ID, "admin")
		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Create request with user context (simulating auth middleware)
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", tenant.ID)

		// Inject user context (what auth middleware would do)
		reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, "", "", user.Auth0ID)
		req = req.WithContext(reqCtx)

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert: Success with tenant context injected
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response contains tenant info
		assert.Contains(t, w.Body.String(), tenant.ID)
		assert.Contains(t, w.Body.String(), "admin")

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", tenant.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestTenantMiddleware_MissingHeader_Returns400 tests that missing X-Tenant-ID
// header returns 400 Bad Request
func TestTenantMiddleware_MissingHeader_Returns400(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create a user
		setupTx, err := db.Beginx()
		require.NoError(t, err)
		user := testutil.CreateUser(t, setupTx, "auth0|test-user", "test@example.com", "Test", "User")
		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Create request WITHOUT X-Tenant-ID header
		req := httptest.NewRequest("GET", "/test", nil)
		// NO header: req.Header.Set("X-Tenant-ID", ...)

		// Inject user context
		reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, "", "", user.Auth0ID)
		req = req.WithContext(reqCtx)

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert: 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "X-Tenant-ID header required")

		// Cleanup
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestTenantMiddleware_UnauthorizedTenant_Returns403 tests that when a user
// tries to access a tenant they don't belong to, they get 403 Forbidden
func TestTenantMiddleware_UnauthorizedTenant_Returns403(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create two users and two tenants
		setupTx, err := db.Beginx()
		require.NoError(t, err)

		// User 1 with Tenant A
		user1 := testutil.CreateUser(t, setupTx, "auth0|user-1", "user1@example.com", "User", "One")
		tenantA := testutil.CreateTenant(t, setupTx, "Tenant A", "tenant-a")
		testutil.CreateTenantMember(t, setupTx, user1.ID, tenantA.ID, "owner")

		// User 2 with Tenant B
		user2 := testutil.CreateUser(t, setupTx, "auth0|user-2", "user2@example.com", "User", "Two")
		tenantB := testutil.CreateTenant(t, setupTx, "Tenant B", "tenant-b")
		testutil.CreateTenantMember(t, setupTx, user2.ID, tenantB.ID, "owner")

		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Create request: User 1 tries to access Tenant B (unauthorized)
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", tenantB.ID) // User1 trying to access TenantB

		// Inject User 1's context
		reqCtx := pkgcontext.WithAuth(req.Context(), user1.ID, "", "", user1.Auth0ID)
		req = req.WithContext(reqCtx)

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert: 403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Access denied")

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id IN ($1, $2)", user1.ID, user2.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id IN ($1, $2)", tenantA.ID, tenantB.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id IN ($1, $2)", user1.ID, user2.ID)
	})
}

// TestTenantMiddleware_TenantSwitching_OverridesAuthContext tests that
// X-Tenant-ID header correctly overrides the tenant from auth middleware
func TestTenantMiddleware_TenantSwitching_OverridesAuthContext(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create one user with memberships in TWO tenants
		setupTx, err := db.Beginx()
		require.NoError(t, err)

		user := testutil.CreateUser(t, setupTx, "auth0|multi-tenant-user", "user@example.com", "Multi", "User")

		tenantA := testutil.CreateTenant(t, setupTx, "Tenant A", "tenant-a")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenantA.ID, "owner")

		tenantB := testutil.CreateTenant(t, setupTx, "Tenant B", "tenant-b")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenantB.ID, "member") // Different role

		testutil.SetUserLastActiveTenant(t, setupTx, user.ID, tenantA.ID) // Last active is A

		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Create request: Auth middleware set TenantA, but X-Tenant-ID requests TenantB
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", tenantB.ID) // Switch to Tenant B

		// Inject user context with TenantA (what auth middleware would set based on last_active)
		reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, tenantA.ID, "owner", user.Auth0ID)
		req = req.WithContext(reqCtx)

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert: Success and tenant context is OVERRIDDEN to Tenant B
		require.Equal(t, http.StatusOK, w.Code)

		// The response should show Tenant B (not A) and role "member" (not "owner")
		assert.Contains(t, w.Body.String(), tenantB.ID)
		assert.Contains(t, w.Body.String(), "member")
		assert.NotContains(t, w.Body.String(), tenantA.ID)

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id IN ($1, $2)", tenantA.ID, tenantB.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestTenantMiddleware_MultipleRoles_ReturnsCorrectRole tests that when a user
// has different roles in different tenants, the correct role is injected
func TestTenantMiddleware_MultipleRoles_ReturnsCorrectRole(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: One user with different roles in different tenants
		setupTx, err := db.Beginx()
		require.NoError(t, err)

		user := testutil.CreateUser(t, setupTx, "auth0|role-test-user", "user@example.com", "Role", "User")

		tenantOwner := testutil.CreateTenant(t, setupTx, "Owner Tenant", "owner-tenant")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenantOwner.ID, "owner")

		tenantMember := testutil.CreateTenant(t, setupTx, "Member Tenant", "member-tenant")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenantMember.ID, "member")

		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Test 1: Access tenant where user is "owner"
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Tenant-ID", tenantOwner.ID)
		ctx1 := pkgcontext.WithAuth(req1.Context(), user.ID, "", "", user.Auth0ID)
		req1 = req1.WithContext(ctx1)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Contains(t, w1.Body.String(), "owner")

		// Test 2: Access tenant where user is "member"
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Tenant-ID", tenantMember.ID)
		ctx2 := pkgcontext.WithAuth(req2.Context(), user.ID, "", "", user.Auth0ID)
		req2 = req2.WithContext(ctx2)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Contains(t, w2.Body.String(), "member")
		assert.NotContains(t, w2.Body.String(), "owner")

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id IN ($1, $2)", tenantOwner.ID, tenantMember.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestTenantMiddleware_InvalidTenantID_Returns403 tests that an invalid/non-existent
// tenant ID in the header returns 403 (user has no membership to non-existent tenant)
func TestTenantMiddleware_InvalidTenantID_Returns403(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create a user
		setupTx, err := db.Beginx()
		require.NoError(t, err)
		user := testutil.CreateUser(t, setupTx, "auth0|test-user", "test@example.com", "Test", "User")
		require.NoError(t, setupTx.Commit())

		// Setup router
		router := setupTestRouter(tenantRepo)

		// Create request with non-existent tenant ID
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000000") // Non-existent UUID

		// Inject user context
		reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, "", "", user.Auth0ID)
		req = req.WithContext(reqCtx)

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert: 403 Forbidden (user not a member of this tenant)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Access denied")

		// Cleanup
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}
