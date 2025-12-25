package e2e_test

import (
	"context"
	"os"
	"testing"
	"time"

	flagspkg "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/projects"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jalil32/toggle/internal/users"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
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

// TestE2E_NewUserOnboardingJourney simulates a complete first-time user experience:
// 1. New Auth0 user signs in
// 2. System creates user record, default tenant, and membership
// 3. User creates their first project
// 4. User creates their first feature flag
// 5. User retrieves and toggles the flag
func TestE2E_NewUserOnboardingJourney(t *testing.T) {
	db := testutil.GetTestDB()

	// Initialize all services (simulating production setup)
	userRepo := users.NewRepository(db)
	tenantRepo := tenants.NewRepository(db)
	projectRepo := projects.NewRepository(db)
	flagRepo := flagspkg.NewRepository(db)
	uow := transaction.NewUnitOfWork(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, tenantRepo, uow, logger)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// === STEP 1: New user signs in via Auth0 ===
		auth0ID := "auth0|journey-user-123"
		firstname := "Alice"
		lastname := "Developer"

		startTime := time.Now()

		// System automatically creates user + tenant + membership
		user, err := userService.GetOrCreate(ctx, auth0ID, firstname, lastname)
		require.NoError(t, err, "User onboarding should succeed")
		require.NotNil(t, user)

		onboardingTime := time.Since(startTime)
		t.Logf("User onboarding completed in %v", onboardingTime)

		// Verify user has default tenant
		assert.NotEmpty(t, user.ID)
		assert.Equal(t, auth0ID, user.Auth0ID)
		assert.NotNil(t, user.LastActiveTenantID, "User should have default tenant")

		// === STEP 2: User retrieves their tenant info ===
		tenantMemberships, err := tenantRepo.ListUserTenants(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, tenantMemberships, 1, "New user should have exactly one tenant")

		membership := tenantMemberships[0]
		assert.Equal(t, "owner", membership.Role, "User should be owner of their default workspace")
		assert.Contains(t, membership.TenantName, "Alice Developer", "Tenant should be personalized")

		tenantID := membership.TenantID

		// === STEP 3: User creates their first project ===
		project, err := projectRepo.Create(ctx, tenantID, "My First Project")
		require.NoError(t, err, "Project creation should succeed")
		require.NotNil(t, project)
		assert.Equal(t, tenantID, project.TenantID)
		assert.NotEmpty(t, project.ClientAPIKey, "API key should be generated")
		assert.Len(t, project.ClientAPIKey, 64, "API key should be 64 characters")

		// === STEP 4: User creates their first feature flag ===
		firstFlag := &flagspkg.Flag{
			Name:        "welcome-banner",
			Description: "Show welcome banner to new users",
			Enabled:     false,
			ProjectID:   project.ID,
			Rules:       []flagspkg.Rule{},
		}

		err = flagRepo.Create(ctx, firstFlag)
		require.NoError(t, err, "Flag creation should succeed")
		assert.NotEmpty(t, firstFlag.ID)

		// === STEP 5: User retrieves their flag ===
		retrievedFlag, err := flagRepo.GetByID(ctx, firstFlag.ID, tenantID)
		require.NoError(t, err, "Flag retrieval should succeed")
		assert.Equal(t, "welcome-banner", retrievedFlag.Name)
		assert.False(t, retrievedFlag.Enabled, "Flag should start disabled")

		// === STEP 6: User toggles the flag ===
		retrievedFlag.Enabled = true
		err = flagRepo.Update(ctx, retrievedFlag, tenantID)
		require.NoError(t, err, "Flag update should succeed")

		// === STEP 7: Verify flag is enabled ===
		toggledFlag, err := flagRepo.GetByID(ctx, firstFlag.ID, tenantID)
		require.NoError(t, err)
		assert.True(t, toggledFlag.Enabled, "Flag should now be enabled")

		// === STEP 8: User lists all their flags ===
		allFlags, err := flagRepo.ListByProject(ctx, project.ID, tenantID)
		require.NoError(t, err)
		assert.Len(t, allFlags, 1, "User should have exactly one flag")

		totalTime := time.Since(startTime)
		t.Logf("Complete onboarding journey took %v", totalTime)

		// Performance assertion: entire journey should complete quickly
		assert.Less(t, totalTime.Milliseconds(), int64(500), "E2E journey should complete in <500ms")

		// Cleanup handled by transaction rollback
	})
}

// TestE2E_MultiTenantUserJourney simulates a user who belongs to multiple tenants:
// 1. User is member of 3 different tenants with different roles
// 2. User switches between tenants
// 3. User creates resources in each tenant
// 4. Verify complete data isolation between tenants
func TestE2E_MultiTenantUserJourney(t *testing.T) {
	db := testutil.GetTestDB()

	tenantRepo := tenants.NewRepository(db)
	projectRepo := projects.NewRepository(db)
	flagRepo := flagspkg.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// === SETUP: Create user and 3 tenants with different roles ===
		user := testutil.CreateUser(t, cleanupTx, "auth0|multi-tenant-user", "bob@example.com", "Bob", "MultiTenant")

		// Tenant A: Bob is owner
		tenantA := testutil.CreateTenant(t, cleanupTx, "Company A", "company-a")
		testutil.CreateTenantMember(t, cleanupTx, user.ID, tenantA.ID, "owner")

		// Tenant B: Bob is admin
		tenantB := testutil.CreateTenant(t, cleanupTx, "Company B", "company-b")
		testutil.CreateTenantMember(t, cleanupTx, user.ID, tenantB.ID, "admin")

		// Tenant C: Bob is member
		tenantC := testutil.CreateTenant(t, cleanupTx, "Company C", "company-c")
		testutil.CreateTenantMember(t, cleanupTx, user.ID, tenantC.ID, "member")

		// Inject transaction into context for all operations
		ctx = transaction.InjectTx(ctx, cleanupTx)

		// === STEP 1: User retrieves all their tenant memberships ===
		memberships, err := tenantRepo.ListUserTenants(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, memberships, 3, "User should belong to 3 tenants")

		// Verify roles
		roleMap := make(map[string]string)
		for _, m := range memberships {
			roleMap[m.TenantID] = m.Role
		}
		assert.Equal(t, "owner", roleMap[tenantA.ID])
		assert.Equal(t, "admin", roleMap[tenantB.ID])
		assert.Equal(t, "member", roleMap[tenantC.ID])

		// === STEP 2: Work in Tenant A (as owner) ===
		projectA1 := testutil.CreateProject(t, cleanupTx, tenantA.ID, "Project A1", "api-key-a1")
		projectA2 := testutil.CreateProject(t, cleanupTx, tenantA.ID, "Project A2", "api-key-a2")

		flagA1 := testutil.CreateFlag(t, cleanupTx, projectA1.ID, "flag-a1", "Flag A1", true)
		flagA2 := testutil.CreateFlag(t, cleanupTx, projectA2.ID, "flag-a2", "Flag A2", false)

		// List all Tenant A projects
		projectsA, err := projectRepo.ListByTenantID(ctx, tenantA.ID)
		require.NoError(t, err)
		assert.Len(t, projectsA, 2, "Tenant A should have 2 projects")

		// === STEP 3: Switch to Tenant B (as admin) ===
		projectB1 := testutil.CreateProject(t, cleanupTx, tenantB.ID, "Project B1", "api-key-b1")
		flagB1 := testutil.CreateFlag(t, cleanupTx, projectB1.ID, "flag-b1", "Flag B1", true)

		projectsB, err := projectRepo.ListByTenantID(ctx, tenantB.ID)
		require.NoError(t, err)
		assert.Len(t, projectsB, 1, "Tenant B should have 1 project")

		// === STEP 4: Switch to Tenant C (as member) ===
		projectC1 := testutil.CreateProject(t, cleanupTx, tenantC.ID, "Project C1", "api-key-c1")
		flagC1 := testutil.CreateFlag(t, cleanupTx, projectC1.ID, "flag-c1", "Flag C1", false)

		projectsC, err := projectRepo.ListByTenantID(ctx, tenantC.ID)
		require.NoError(t, err)
		assert.Len(t, projectsC, 1, "Tenant C should have 1 project")

		// === STEP 5: Verify complete data isolation ===

		// Tenant A can only see Tenant A's flags
		flagsA, err := flagRepo.List(ctx, tenantA.ID)
		require.NoError(t, err)
		assert.Len(t, flagsA, 2, "Tenant A should see exactly 2 flags")
		for _, flag := range flagsA {
			assert.Contains(t, []string{flagA1.ID, flagA2.ID}, flag.ID)
		}

		// Tenant B can only see Tenant B's flags
		flagsB, err := flagRepo.List(ctx, tenantB.ID)
		require.NoError(t, err)
		assert.Len(t, flagsB, 1, "Tenant B should see exactly 1 flag")
		assert.Equal(t, flagB1.ID, flagsB[0].ID)

		// Tenant C can only see Tenant C's flags
		flagsC, err := flagRepo.List(ctx, tenantC.ID)
		require.NoError(t, err)
		assert.Len(t, flagsC, 1, "Tenant C should see exactly 1 flag")
		assert.Equal(t, flagC1.ID, flagsC[0].ID)

		// === STEP 6: Verify cross-tenant access is prevented ===

		// Tenant A cannot access Tenant B's project
		_, err = projectRepo.GetByID(ctx, projectB1.ID, tenantA.ID)
		assert.Error(t, err, "Cross-tenant project access should fail")

		// Tenant B cannot access Tenant C's flag
		_, err = flagRepo.GetByID(ctx, flagC1.ID, tenantB.ID)
		assert.Error(t, err, "Cross-tenant flag access should fail")

		t.Log("Complete multi-tenant isolation verified across 3 tenants")

		// Cleanup handled by transaction rollback
	})
}

// TestE2E_LoadTest_CreateManyTenants verifies performance doesn't degrade
// when creating multiple tenants, projects, and flags
func TestE2E_LoadTest_CreateManyTenants(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	db := testutil.GetTestDB()

	projectRepo := projects.NewRepository(db)
	flagRepo := flagspkg.NewRepository(db)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		ctx = transaction.InjectTx(ctx, cleanupTx)

		numTenants := 50
		projectsPerTenant := 3
		flagsPerProject := 5

		startTime := time.Now()

		createdTenantIDs := make([]string, 0, numTenants)
		createdProjectIDs := make([]string, 0, numTenants*projectsPerTenant)

		// === Create 50 tenants, each with 3 projects and 5 flags per project ===
		for i := 0; i < numTenants; i++ {
			// Create tenant
			tenant := testutil.CreateTenant(t, cleanupTx,
				"Load Test Tenant "+string(rune('A'+i%26)),
				"load-tenant-"+string(rune('a'+i)))
			createdTenantIDs = append(createdTenantIDs, tenant.ID)

			// Create projects for this tenant
			for j := 0; j < projectsPerTenant; j++ {
				project := testutil.CreateProject(t, cleanupTx, tenant.ID,
					"Project "+string(rune('A'+j)),
					"api-key-"+tenant.ID+"-"+string(rune('a'+j)))
				createdProjectIDs = append(createdProjectIDs, project.ID)

				// Create flags for this project
				for k := 0; k < flagsPerProject; k++ {
					flag := testutil.CreateFlag(t, cleanupTx, project.ID,
						"flag-"+string(rune('a'+k)),
						"Flag "+string(rune('A'+k)),
						k%2 == 0)
					_ = flag
				}
			}
		}

		creationTime := time.Since(startTime)
		totalResources := numTenants + (numTenants * projectsPerTenant) + (numTenants * projectsPerTenant * flagsPerProject)

		t.Logf("Created %d resources (50 tenants, 150 projects, 750 flags) in %v",
			totalResources, creationTime)
		t.Logf("Average time per resource: %v", creationTime/time.Duration(totalResources))

		// === Verify query performance on populated database ===
		queryStart := time.Now()

		// Test 1: List projects for first tenant
		firstTenantProjects, err := projectRepo.ListByTenantID(ctx, createdTenantIDs[0])
		require.NoError(t, err)
		assert.Len(t, firstTenantProjects, projectsPerTenant)

		queryTime := time.Since(queryStart)
		t.Logf("Query time for listing projects: %v", queryTime)
		assert.Less(t, queryTime.Milliseconds(), int64(100), "Query should complete in <100ms")

		// Test 2: List flags for first project
		queryStart = time.Now()
		firstProjectFlags, err := flagRepo.ListByProject(ctx, createdProjectIDs[0], createdTenantIDs[0])
		require.NoError(t, err)
		assert.Len(t, firstProjectFlags, flagsPerProject)

		flagQueryTime := time.Since(queryStart)
		t.Logf("Query time for listing flags: %v", flagQueryTime)
		assert.Less(t, flagQueryTime.Milliseconds(), int64(100), "Flag query should complete in <100ms")

		// Test 3: Verify tenant isolation at scale
		// Even with 50 tenants, we should only see our tenant's data
		allTenantAFlags, err := flagRepo.List(ctx, createdTenantIDs[0])
		require.NoError(t, err)
		assert.Len(t, allTenantAFlags, projectsPerTenant*flagsPerProject,
			"Should see exactly flags from our tenant's projects")

		// === Performance Assertions ===
		avgCreationTime := creationTime / time.Duration(totalResources)
		t.Logf("Performance summary:")
		t.Logf("  - Total creation time: %v", creationTime)
		t.Logf("  - Average per resource: %v", avgCreationTime)
		t.Logf("  - Query performance: %v (projects), %v (flags)", queryTime, flagQueryTime)

		// Reasonable performance expectations
		assert.Less(t, avgCreationTime.Milliseconds(), int64(10),
			"Average resource creation should be <10ms")
		assert.Less(t, creationTime.Seconds(), float64(10),
			"Creating 800 resources should take <10 seconds")

		t.Log("Load test passed: System performs well with 50 tenants and 800 total resources")

		// Cleanup handled by transaction rollback
	})
}
