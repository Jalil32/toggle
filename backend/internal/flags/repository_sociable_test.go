package flag_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	flag "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up the test database container once for all tests
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup: Start PostgreSQL container and run migrations
	_, err := testutil.SetupTestDatabase(ctx, "../../migrations")
	if err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	// Teardown: Clean up container
	if err := testutil.TeardownTestDatabase(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestRepository_Create_Sociable(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create tenant and project
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		project := testutil.CreateProject(t, tx, tenant.ID, "Test Project", "test-api-key-123")

		// Initialize repository with DB (not tx - it will use context)
		repo := flag.NewRepository(testutil.GetTestDB())

		// Test: Create a flag
		newFlag := &flag.Flag{
			TenantID:    tenant.ID,
			Name:        "new-feature",
			Description: "A new feature flag",
			Enabled:     false,
			Rules:       []flag.Rule{},
			RuleLogic:   "AND",
			ProjectID:   &project.ID,
		}

		err := repo.Create(ctx, newFlag)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, newFlag.ID, "ID should be generated")
		assert.NotZero(t, newFlag.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, newFlag.UpdatedAt, "UpdatedAt should be set")
	})
}

func TestRepository_GetByID_TenantIsolation(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create two tenants with their own projects and flags
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")

		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")
		project2 := testutil.CreateProject(t, tx, tenant2.ID, "Project 2", "api-key-2")

		repo := flag.NewRepository(testutil.GetTestDB())

		// Create flags for each tenant
		flag1 := &flag.Flag{
			TenantID:    tenant1.ID,
			Name:        "tenant1-flag",
			Description: "Flag for tenant 1",
			Enabled:     true,
			Rules:       []flag.Rule{},
			RuleLogic:   "AND",
			ProjectID:   &project1.ID,
		}
		require.NoError(t, repo.Create(ctx, flag1))

		flag2 := &flag.Flag{
			TenantID:    tenant2.ID,
			Name:        "tenant2-flag",
			Description: "Flag for tenant 2",
			Enabled:     false,
			Rules:       []flag.Rule{},
			RuleLogic:   "AND",
			ProjectID:   &project2.ID,
		}
		require.NoError(t, repo.Create(ctx, flag2))

		// Test: Tenant 1 can access their own flag
		retrieved, err := repo.GetByID(ctx, flag1.ID, tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, "tenant1-flag", retrieved.Name)

		// Test: Tenant 1 CANNOT access Tenant 2's flag (tenant isolation)
		retrieved, err = repo.GetByID(ctx, flag2.ID, tenant1.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should return no rows when accessing another tenant's flag")
		assert.Nil(t, retrieved)

		// Test: Tenant 2 can access their own flag
		retrieved, err = repo.GetByID(ctx, flag2.ID, tenant2.ID)
		require.NoError(t, err)
		assert.Equal(t, "tenant2-flag", retrieved.Name)
	})
}

func TestRepository_List_OnlyReturnsTenantData(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create two tenants with multiple flags each
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")

		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")
		project2 := testutil.CreateProject(t, tx, tenant2.ID, "Project 2", "api-key-2")

		repo := flag.NewRepository(testutil.GetTestDB())

		// Create 3 flags for tenant 1
		for i := 1; i <= 3; i++ {
			f := &flag.Flag{
				TenantID:    tenant1.ID,
				Name:        fmt.Sprintf("tenant1-flag-%d", i),
				Description: "Tenant 1 flag",
				Enabled:     true,
				Rules:       []flag.Rule{},
				RuleLogic:   "AND",
				ProjectID:   &project1.ID,
			}
			require.NoError(t, repo.Create(ctx, f))
		}

		// Create 2 flags for tenant 2
		for i := 1; i <= 2; i++ {
			f := &flag.Flag{
				TenantID:    tenant2.ID,
				Name:        fmt.Sprintf("tenant2-flag-%d", i),
				Description: "Tenant 2 flag",
				Enabled:     false,
				Rules:       []flag.Rule{},
				RuleLogic:   "AND",
				ProjectID:   &project2.ID,
			}
			require.NoError(t, repo.Create(ctx, f))
		}

		// Test: Tenant 1 should only see their 3 flags
		flags1, err := repo.List(ctx, tenant1.ID)
		require.NoError(t, err)
		assert.Len(t, flags1, 3, "Tenant 1 should see exactly 3 flags")
		for _, f := range flags1 {
			assert.Contains(t, f.Name, "tenant1", "All flags should belong to tenant 1")
		}

		// Test: Tenant 2 should only see their 2 flags
		flags2, err := repo.List(ctx, tenant2.ID)
		require.NoError(t, err)
		assert.Len(t, flags2, 2, "Tenant 2 should see exactly 2 flags")
		for _, f := range flags2 {
			assert.Contains(t, f.Name, "tenant2", "All flags should belong to tenant 2")
		}
	})
}

func TestRepository_Update_EnforcesTenantBoundary(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")

		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		repo := flag.NewRepository(testutil.GetTestDB())

		// Create a flag for tenant 1
		originalFlag := &flag.Flag{
			TenantID:    tenant1.ID,
			Name:        "original-flag",
			Description: "Original",
			Enabled:     false,
			Rules:       []flag.Rule{},
			RuleLogic:   "AND",
			ProjectID:   &project1.ID,
		}
		require.NoError(t, repo.Create(ctx, originalFlag))

		// Test: Tenant 1 can update their own flag
		originalFlag.Name = "updated-flag"
		originalFlag.Enabled = true
		err := repo.Update(ctx, originalFlag, tenant1.ID)
		require.NoError(t, err)

		// Verify update succeeded
		retrieved, err := repo.GetByID(ctx, originalFlag.ID, tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-flag", retrieved.Name)
		assert.True(t, retrieved.Enabled)

		// Test: Tenant 2 CANNOT update Tenant 1's flag
		originalFlag.Name = "malicious-update"
		err = repo.Update(ctx, originalFlag, tenant2.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should fail when tenant tries to update another tenant's flag")

		// Verify flag was NOT updated by tenant 2
		retrieved, err = repo.GetByID(ctx, originalFlag.ID, tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-flag", retrieved.Name, "Name should still be from tenant 1's update")
	})
}

func TestRepository_Delete_EnforcesTenantBoundary(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		project1 := testutil.CreateProject(t, tx, tenant1.ID, "Project 1", "api-key-1")

		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		repo := flag.NewRepository(testutil.GetTestDB())

		// Create a flag for tenant 1
		testFlag := &flag.Flag{
			TenantID:    tenant1.ID,
			Name:        "test-flag",
			Description: "Test flag",
			Enabled:     true,
			Rules:       []flag.Rule{},
			RuleLogic:   "AND",
			ProjectID:   &project1.ID,
		}
		require.NoError(t, repo.Create(ctx, testFlag))

		// Test: Tenant 2 CANNOT delete Tenant 1's flag
		err := repo.Delete(ctx, testFlag.ID, tenant2.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows, "Should fail when tenant tries to delete another tenant's flag")

		// Verify flag still exists
		retrieved, err := repo.GetByID(ctx, testFlag.ID, tenant1.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved, "Flag should still exist")

		// Test: Tenant 1 CAN delete their own flag
		err = repo.Delete(ctx, testFlag.ID, tenant1.ID)
		require.NoError(t, err)

		// Verify flag is deleted
		retrieved, err = repo.GetByID(ctx, testFlag.ID, tenant1.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, retrieved)
	})
}

// Note: TestRepository_ZeroContext_Protection is intentionally not included
// In a real UoW pattern, repositories should always receive a context with either
// a transaction (from UoW) or use the DB directly. Testing "zero context" isn't
// meaningful in our transactional test setup since test data lives in a transaction.
