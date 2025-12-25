package projects_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/projects"
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

// TestRepository_ListByTenantID_OnlyReturnsTenantProjects tests that
// ListByTenantID only returns projects belonging to the specified tenant
func TestRepository_ListByTenantID_OnlyReturnsTenantProjects(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create two tenants with projects
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		// Tenant 1 has 3 projects
		project1A := testutil.CreateProject(t, tx, tenant1.ID, "Project 1A", "api-key-1a")
		project1B := testutil.CreateProject(t, tx, tenant1.ID, "Project 1B", "api-key-1b")
		project1C := testutil.CreateProject(t, tx, tenant1.ID, "Project 1C", "api-key-1c")

		// Tenant 2 has 2 projects
		project2A := testutil.CreateProject(t, tx, tenant2.ID, "Project 2A", "api-key-2a")
		project2B := testutil.CreateProject(t, tx, tenant2.ID, "Project 2B", "api-key-2b")

		// Initialize repository
		repo := projects.NewRepository(testutil.GetTestDB())

		// Inject transaction into context
		ctx = transaction.InjectTx(ctx, tx)

		// Test: List projects for Tenant 1
		tenant1Projects, err := repo.ListByTenantID(ctx, tenant1.ID)
		require.NoError(t, err)

		// Assert: Tenant 1 sees only their 3 projects
		require.Len(t, tenant1Projects, 3, "Tenant 1 should see exactly 3 projects")
		projectNames := make([]string, len(tenant1Projects))
		for i, p := range tenant1Projects {
			projectNames[i] = p.Name
		}
		assert.Contains(t, projectNames, "Project 1A")
		assert.Contains(t, projectNames, "Project 1B")
		assert.Contains(t, projectNames, "Project 1C")
		assert.NotContains(t, projectNames, "Project 2A")
		assert.NotContains(t, projectNames, "Project 2B")

		// Test: List projects for Tenant 2
		tenant2Projects, err := repo.ListByTenantID(ctx, tenant2.ID)
		require.NoError(t, err)

		// Assert: Tenant 2 sees only their 2 projects
		require.Len(t, tenant2Projects, 2, "Tenant 2 should see exactly 2 projects")
		projectNames2 := make([]string, len(tenant2Projects))
		for i, p := range tenant2Projects {
			projectNames2[i] = p.Name
		}
		assert.Contains(t, projectNames2, "Project 2A")
		assert.Contains(t, projectNames2, "Project 2B")
		assert.NotContains(t, projectNames2, "Project 1A")

		// Verify project IDs match
		projectIDs1 := []string{project1A.ID, project1B.ID, project1C.ID}
		for _, p := range tenant1Projects {
			assert.Contains(t, projectIDs1, p.ID)
		}

		projectIDs2 := []string{project2A.ID, project2B.ID}
		for _, p := range tenant2Projects {
			assert.Contains(t, projectIDs2, p.ID)
		}
	})
}

// TestRepository_GetByID_EnforcesTenantBoundary tests that GetByID
// prevents cross-tenant access to projects
func TestRepository_GetByID_EnforcesTenantBoundary(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create two tenants with projects
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")
		project2 := testutil.CreateProject(t, tx, tenant2.ID, "Project 2", "api-key-2")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Test: Tenant 1 can access their own project
		retrieved, err := repo.GetByID(ctx, project1.ID, tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, project1.ID, retrieved.ID)
		assert.Equal(t, "Project 1", retrieved.Name)

		// Test: Tenant 1 CANNOT access Tenant 2's project
		retrieved, err = repo.GetByID(ctx, project2.ID, tenant1.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should return no rows when accessing another tenant's project")
		assert.Nil(t, retrieved)

		// Test: Tenant 2 can access their own project
		retrieved, err = repo.GetByID(ctx, project2.ID, tenant2.ID)
		require.NoError(t, err)
		assert.Equal(t, project2.ID, retrieved.ID)
		assert.Equal(t, "Project 2", retrieved.Name)

		// Test: Tenant 2 CANNOT access Tenant 1's project
		retrieved, err = repo.GetByID(ctx, project1.ID, tenant2.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, retrieved)
	})
}

// TestRepository_Delete_EnforcesTenantBoundary tests that Delete
// prevents cross-tenant deletion
func TestRepository_Delete_EnforcesTenantBoundary(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")
		project2 := testutil.CreateProject(t, tx, tenant2.ID, "Project 2", "api-key-2")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Test: Tenant 2 CANNOT delete Tenant 1's project
		err := repo.Delete(ctx, project1.ID, tenant2.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should fail when tenant tries to delete another tenant's project")

		// Verify project still exists
		retrieved, err := repo.GetByID(ctx, project1.ID, tenant1.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved, "Project should still exist")

		// Test: Tenant 1 CAN delete their own project
		err = repo.Delete(ctx, project1.ID, tenant1.ID)
		require.NoError(t, err)

		// Verify project is deleted
		retrieved, err = repo.GetByID(ctx, project1.ID, tenant1.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, retrieved)

		// Verify Tenant 2's project is unaffected
		retrieved, err = repo.GetByID(ctx, project2.ID, tenant2.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

// TestRepository_Create_GeneratesUniqueAPIKey tests that each project
// gets a unique API key
func TestRepository_Create_GeneratesUniqueAPIKey(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Create multiple projects
		project1, err := repo.Create(ctx, tenant.ID, "Project 1")
		require.NoError(t, err)

		project2, err := repo.Create(ctx, tenant.ID, "Project 2")
		require.NoError(t, err)

		project3, err := repo.Create(ctx, tenant.ID, "Project 3")
		require.NoError(t, err)

		// Assert: All API keys are unique
		apiKeys := []string{project1.ClientAPIKey, project2.ClientAPIKey, project3.ClientAPIKey}
		uniqueKeys := make(map[string]bool)
		for _, key := range apiKeys {
			assert.NotEmpty(t, key, "API key should be generated")
			assert.Len(t, key, 64, "API key should be 64 characters (32 bytes hex-encoded)")
			uniqueKeys[key] = true
		}
		assert.Len(t, uniqueKeys, 3, "All API keys should be unique")
	})
}

// TestRepository_ListByTenantID_EmptyTenant tests that listing projects
// for a tenant with no projects returns an empty slice, not an error
func TestRepository_ListByTenantID_EmptyTenant(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create tenant with NO projects
		tenant := testutil.CreateTenant(t, tx, "Empty Tenant", "empty-tenant")

		repo := projects.NewRepository(testutil.GetTestDB())
		ctx = transaction.InjectTx(ctx, tx)

		// Test: List projects for empty tenant
		projects, err := repo.ListByTenantID(ctx, tenant.ID)

		// Assert: No error, empty slice
		require.NoError(t, err)
		assert.NotNil(t, projects, "Should return empty slice, not nil")
		assert.Len(t, projects, 0, "Should have zero projects")
	})
}
