package integration_test

import (
	"context"
	"os"
	"testing"

	flagspkg "github.com/jalil32/toggle/internal/flags"
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

// TestCascadeDelete_TenantDeletion_DeletesProjectsAndFlags tests that
// when a tenant is deleted, all associated projects and flags are also deleted
func TestCascadeDelete_TenantDeletion_DeletesProjectsAndFlags(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create tenant with projects and flags
		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		// Tenant 1 has 2 projects
		project1A := testutil.CreateProject(t, tx, tenant1.ID, "Project 1A", "api-key-1a")
		project1B := testutil.CreateProject(t, tx, tenant1.ID, "Project 1B", "api-key-1b")

		// Tenant 2 has 1 project (control group)
		project2A := testutil.CreateProject(t, tx, tenant2.ID, "Project 2A", "api-key-2a")

		// Create flags for tenant 1's projects
		flag1A1 := testutil.CreateFlag(t, tx, project1A.ID, "flag-1a-1", "Flag 1A1", false)
		flag1A2 := testutil.CreateFlag(t, tx, project1A.ID, "flag-1a-2", "Flag 1A2", true)
		flag1B1 := testutil.CreateFlag(t, tx, project1B.ID, "flag-1b-1", "Flag 1B1", false)

		// Create flag for tenant 2's project (control group)
		flag2A1 := testutil.CreateFlag(t, tx, project2A.ID, "flag-2a-1", "Flag 2A1", true)

		// Inject transaction into context
		ctx = transaction.InjectTx(ctx, tx)

		// Verify initial state
		projectRepo := projects.NewRepository(testutil.GetTestDB())
		flagRepo := flagspkg.NewRepository(testutil.GetTestDB())

		tenant1Projects, err := projectRepo.ListByTenantID(ctx, tenant1.ID)
		require.NoError(t, err)
		assert.Len(t, tenant1Projects, 2, "Tenant 1 should have 2 projects")

		// Test: Delete Tenant 1
		result, err := tx.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", tenant1.ID)
		require.NoError(t, err)
		rowsAffected, _ := result.RowsAffected()
		assert.Equal(t, int64(1), rowsAffected, "Should delete 1 tenant")

		// Assert: Tenant 1's projects are deleted
		tenant1ProjectsAfter, err := projectRepo.ListByTenantID(ctx, tenant1.ID)
		require.NoError(t, err)
		assert.Len(t, tenant1ProjectsAfter, 0, "Tenant 1 should have 0 projects after deletion")

		// Assert: Tenant 1's flags are deleted (cascade through projects)
		var flagCount int
		err = tx.GetContext(ctx, &flagCount, "SELECT COUNT(*) FROM flags WHERE id IN ($1, $2, $3)", flag1A1.ID, flag1A2.ID, flag1B1.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, flagCount, "All flags belonging to tenant 1's projects should be deleted")

		// Assert: Tenant 2's project and flags are unaffected
		tenant2ProjectsAfter, err := projectRepo.ListByTenantID(ctx, tenant2.ID)
		require.NoError(t, err)
		assert.Len(t, tenant2ProjectsAfter, 1, "Tenant 2 should still have 1 project")

		retrievedFlag2A1, err := flagRepo.GetByID(ctx, flag2A1.ID, tenant2.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedFlag2A1, "Tenant 2's flags should still exist")
	})
}

// TestCascadeDelete_ProjectDeletion_DeletesFlags tests that
// when a project is deleted, all associated flags are also deleted
func TestCascadeDelete_ProjectDeletion_DeletesFlags(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create tenant with multiple projects
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")

		projectA := testutil.CreateProject(t, tx, tenant.ID, "Project A", "api-key-a")
		projectB := testutil.CreateProject(t, tx, tenant.ID, "Project B", "api-key-b")

		// Project A has 3 flags
		flagA1 := testutil.CreateFlag(t, tx, projectA.ID, "flag-a-1", "Flag A1", false)
		flagA2 := testutil.CreateFlag(t, tx, projectA.ID, "flag-a-2", "Flag A2", true)
		flagA3 := testutil.CreateFlag(t, tx, projectA.ID, "flag-a-3", "Flag A3", false)

		// Project B has 1 flag (control group)
		flagB1 := testutil.CreateFlag(t, tx, projectB.ID, "flag-b-1", "Flag B1", true)

		ctx = transaction.InjectTx(ctx, tx)
		flagRepo := flagspkg.NewRepository(testutil.GetTestDB())

		// Verify initial state
		flagsA, err := flagRepo.ListByProject(ctx, projectA.ID, tenant.ID)
		require.NoError(t, err)
		assert.Len(t, flagsA, 3, "Project A should have 3 flags")

		// Test: Delete Project A
		result, err := tx.ExecContext(ctx, "DELETE FROM projects WHERE id = $1", projectA.ID)
		require.NoError(t, err)
		rowsAffected, _ := result.RowsAffected()
		assert.Equal(t, int64(1), rowsAffected, "Should delete 1 project")

		// Assert: Project A's flags are deleted
		var flagCount int
		err = tx.GetContext(ctx, &flagCount, "SELECT COUNT(*) FROM flags WHERE id IN ($1, $2, $3)", flagA1.ID, flagA2.ID, flagA3.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, flagCount, "All flags belonging to project A should be deleted")

		// Assert: Project B's flags are unaffected
		flagsB, err := flagRepo.ListByProject(ctx, projectB.ID, tenant.ID)
		require.NoError(t, err)
		assert.Len(t, flagsB, 1, "Project B should still have 1 flag")

		retrievedFlagB1, err := flagRepo.GetByID(ctx, flagB1.ID, tenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedFlagB1, "Project B's flag should still exist")
	})
}

// TestCascadeDelete_TenantDeletion_DeletesMemberships tests that
// when a tenant is deleted, all tenant memberships are also deleted
func TestCascadeDelete_TenantDeletion_DeletesMemberships(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create users and tenants with memberships
		user1 := testutil.CreateUser(t, tx, "User One", "user1@example.com")
		user2 := testutil.CreateUser(t, tx, "User Two", "user2@example.com")

		tenant1 := testutil.CreateTenant(t, tx, "Tenant 1", "tenant-1")
		tenant2 := testutil.CreateTenant(t, tx, "Tenant 2", "tenant-2")

		// Create memberships
		testutil.CreateTenantMember(t, tx, user1.ID, tenant1.ID, "owner")
		testutil.CreateTenantMember(t, tx, user2.ID, tenant1.ID, "member")
		testutil.CreateTenantMember(t, tx, user1.ID, tenant2.ID, "admin") // Control group

		ctx = transaction.InjectTx(ctx, tx)

		// Verify initial state
		var membershipCount int
		err := tx.GetContext(ctx, &membershipCount, "SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1", tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, membershipCount, "Tenant 1 should have 2 memberships")

		// Test: Delete Tenant 1
		result, err := tx.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", tenant1.ID)
		require.NoError(t, err)
		rowsAffected, _ := result.RowsAffected()
		assert.Equal(t, int64(1), rowsAffected, "Should delete 1 tenant")

		// Assert: Tenant 1's memberships are deleted
		err = tx.GetContext(ctx, &membershipCount, "SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1", tenant1.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, membershipCount, "Tenant 1 should have 0 memberships after deletion")

		// Assert: Tenant 2's memberships are unaffected
		err = tx.GetContext(ctx, &membershipCount, "SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1", tenant2.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, membershipCount, "Tenant 2 should still have 1 membership")
	})
}

// TestCascadeDelete_MultiLevel_VerifyIntegrity tests the full cascade:
// Tenant → Projects → Flags in a complex multi-tenant scenario
func TestCascadeDelete_MultiLevel_VerifyIntegrity(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Setup: Create complex multi-tenant structure
		tenant1 := testutil.CreateTenant(t, tx, "Company A", "company-a")
		tenant2 := testutil.CreateTenant(t, tx, "Company B", "company-b")
		tenant3 := testutil.CreateTenant(t, tx, "Company C", "company-c")

		// Each tenant has 2 projects
		p1A := testutil.CreateProject(t, tx, tenant1.ID, "Proj 1A", "key-1a")
		p1B := testutil.CreateProject(t, tx, tenant1.ID, "Proj 1B", "key-1b")
		p2A := testutil.CreateProject(t, tx, tenant2.ID, "Proj 2A", "key-2a")
		p2B := testutil.CreateProject(t, tx, tenant2.ID, "Proj 2B", "key-2b")
		p3A := testutil.CreateProject(t, tx, tenant3.ID, "Proj 3A", "key-3a")
		p3B := testutil.CreateProject(t, tx, tenant3.ID, "Proj 3B", "key-3b")

		// Each project has 2 flags
		testutil.CreateFlag(t, tx, p1A.ID, "f1a1", "Flag 1A1", true)
		testutil.CreateFlag(t, tx, p1A.ID, "f1a2", "Flag 1A2", false)
		testutil.CreateFlag(t, tx, p1B.ID, "f1b1", "Flag 1B1", true)
		testutil.CreateFlag(t, tx, p1B.ID, "f1b2", "Flag 1B2", true)

		testutil.CreateFlag(t, tx, p2A.ID, "f2a1", "Flag 2A1", false)
		testutil.CreateFlag(t, tx, p2A.ID, "f2a2", "Flag 2A2", true)
		testutil.CreateFlag(t, tx, p2B.ID, "f2b1", "Flag 2B1", false)
		testutil.CreateFlag(t, tx, p2B.ID, "f2b2", "Flag 2B2", false)

		testutil.CreateFlag(t, tx, p3A.ID, "f3a1", "Flag 3A1", true)
		testutil.CreateFlag(t, tx, p3A.ID, "f3a2", "Flag 3A2", true)
		testutil.CreateFlag(t, tx, p3B.ID, "f3b1", "Flag 3B1", false)
		testutil.CreateFlag(t, tx, p3B.ID, "f3b2", "Flag 3B2", true)

		ctx = transaction.InjectTx(ctx, tx)

		// Verify initial state: 3 tenants, 6 projects, 12 flags
		var count int
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM tenants")
		assert.Equal(t, 3, count, "Should have 3 tenants")

		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM projects")
		assert.Equal(t, 6, count, "Should have 6 projects")

		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM flags")
		assert.Equal(t, 12, count, "Should have 12 flags")

		// Test: Delete Tenant 2 (should delete 2 projects and 4 flags)
		_, _ = tx.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", tenant2.ID)

		// Assert: Tenant count reduced
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM tenants")
		assert.Equal(t, 2, count, "Should have 2 tenants remaining")

		// Assert: Projects count reduced by 2
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM projects")
		assert.Equal(t, 4, count, "Should have 4 projects remaining")

		// Assert: Flags count reduced by 4
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM flags")
		assert.Equal(t, 8, count, "Should have 8 flags remaining")

		// Assert: Only tenant 1 and 3 data remains
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM projects WHERE tenant_id IN ($1, $2)", tenant1.ID, tenant3.ID)
		assert.Equal(t, 4, count, "Should have all projects for tenant 1 and 3")

		// Verify data integrity: all remaining flags belong to remaining projects
		_ = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM flags WHERE project_id IN ($1, $2, $3, $4)", p1A.ID, p1B.ID, p3A.ID, p3B.ID)
		assert.Equal(t, 8, count, "All remaining flags should belong to remaining projects")
	})
}
