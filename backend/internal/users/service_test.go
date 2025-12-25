package users_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jalil32/toggle/internal/users"
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

// TestGetOrCreate_NewUser_CreatesUserTenantAndMembership tests the first-time user onboarding flow
// This verifies that when a new Auth0 user signs in:
// 1. A new User record is created
// 2. A default Tenant (workspace) is created
// 3. A TenantMember record is created with "owner" role
// 4. The user's last_active_tenant_id is set
//
// NOTE: This test does NOT use WithTestDB because the service uses UoW which creates
// its own transactions. We need to let the service commit, then verify in a separate read.
func TestGetOrCreate_NewUser_CreatesUserTenantAndMembership(t *testing.T) {
	// Setup: Initialize services
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	tenantRepo := tenants.NewRepository(db)
	uow := transaction.NewUnitOfWork(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, tenantRepo, uow, logger)

	// Use a separate transaction just for cleanup at the end
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Act: Call GetOrCreate with a new Auth0 user (service will create its own transaction)
		newAuth0ID := "auth0|new-user-123"
		user, err := userService.GetOrCreate(context.Background(), newAuth0ID, "John", "Doe")

		// Assert: User created successfully
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.NotEmpty(t, user.ID, "User ID should be generated")
		assert.Equal(t, newAuth0ID, user.Auth0ID)
		assert.NotNil(t, user.LastActiveTenantID, "LastActiveTenantID should be set")

		// Verify tenant was created (read in a fresh context)
		tenantMemberships, err := tenantRepo.ListUserTenants(context.Background(), user.ID)
		require.NoError(t, err)
		require.Len(t, tenantMemberships, 1, "User should have exactly one tenant membership")

		membership := tenantMemberships[0]
		assert.Equal(t, "owner", membership.Role, "User should be owner of their default workspace")
		assert.Equal(t, *user.LastActiveTenantID, membership.TenantID, "LastActiveTenantID should match the created tenant")
		assert.Contains(t, membership.TenantName, "John Doe's Workspace", "Tenant name should be personalized")

		// Verify tenant slug was generated
		assert.NotEmpty(t, membership.TenantSlug, "Tenant slug should be generated")

		// Cleanup: Delete the created data manually within this transaction
		// This ensures test isolation even though the service committed to the DB
		_, _ = tx.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = tx.Exec("DELETE FROM tenants WHERE id = $1", *user.LastActiveTenantID)
		_, _ = tx.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestGetOrCreate_ExistingUser_ReturnsExistingUser tests that calling GetOrCreate
// with an existing Auth0 ID returns the existing user without creating duplicates
func TestGetOrCreate_ExistingUser_ReturnsExistingUser(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	tenantRepo := tenants.NewRepository(db)
	uow := transaction.NewUnitOfWork(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, tenantRepo, uow, logger)

	// Create test data OUTSIDE of test transaction so service can see it
	// We'll wrap the whole test in a transaction for cleanup
	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Create test data using a committed transaction
		setupTx, _ := db.Beginx()
		existingUser := testutil.CreateUser(t, setupTx, "auth0|existing-123", "test@example.com", "Jane", "Smith")
		existingTenant := testutil.CreateTenant(t, setupTx, "Jane's Workspace", "jane-workspace")
		testutil.CreateTenantMember(t, setupTx, existingUser.ID, existingTenant.ID, "owner")
		testutil.SetUserLastActiveTenant(t, setupTx, existingUser.ID, existingTenant.ID)
		require.NoError(t, setupTx.Commit()) // COMMIT so service can see this data

		// Act: Call GetOrCreate with existing Auth0 ID
		user, err := userService.GetOrCreate(context.Background(), existingUser.Auth0ID, "Jane", "Smith")

		// Assert: Returns existing user
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID, "Should return the same user ID")
		assert.Equal(t, existingUser.Auth0ID, user.Auth0ID)

		// Verify no duplicate tenants were created
		tenantMemberships, err := tenantRepo.ListUserTenants(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Len(t, tenantMemberships, 1, "Should still have exactly one tenant")

		// Cleanup (cleanupTx will rollback, but we need to delete committed data first)
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", existingTenant.ID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestGetOrCreate_ExistingUserNoTenant_CreatesTenant tests the edge case where
// a user exists but has no tenant membership (orphaned user)
func TestGetOrCreate_ExistingUserNoTenant_CreatesTenant(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	tenantRepo := tenants.NewRepository(db)
	uow := transaction.NewUnitOfWork(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, tenantRepo, uow, logger)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Setup: Create a user without any tenant membership - COMMIT so service can see it
		setupTx, _ := db.Beginx()
		orphanedUser := testutil.CreateUser(t, setupTx, "auth0|orphaned-456", "orphan@example.com", "Bob", "Orphan")
		require.NoError(t, setupTx.Commit())

		// Act: Call GetOrCreate - should create a tenant for the orphaned user
		user, err := userService.GetOrCreate(context.Background(), orphanedUser.Auth0ID, "Bob", "Orphan")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, orphanedUser.ID, user.ID, "Should return the same user")
		assert.NotNil(t, user.LastActiveTenantID, "LastActiveTenantID should now be set")

		// Verify tenant was created for the orphaned user
		tenantMemberships, err := tenantRepo.ListUserTenants(context.Background(), user.ID)
		require.NoError(t, err)
		require.Len(t, tenantMemberships, 1, "Orphaned user should now have a tenant")
		assert.Equal(t, "owner", tenantMemberships[0].Role)

		// Cleanup committed data
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id = $1", user.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", *user.LastActiveTenantID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
	})
}

// TestGetOrCreate_SlugConflict_GeneratesUniqueSlug tests that when two users
// have names that would generate the same slug, unique slugs are created
func TestGetOrCreate_SlugConflict_GeneratesUniqueSlug(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	tenantRepo := tenants.NewRepository(db)
	uow := transaction.NewUnitOfWork(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, tenantRepo, uow, logger)

	testutil.WithTestDB(t, func(ctx context.Context, cleanupTx *sqlx.Tx) {
		// Act: Create first user with name "Test User"
		user1, err := userService.GetOrCreate(context.Background(), "auth0|test-user-1", "Test", "User")
		require.NoError(t, err)

		memberships1, err := tenantRepo.ListUserTenants(context.Background(), user1.ID)
		require.NoError(t, err)
		require.Len(t, memberships1, 1)
		slug1 := memberships1[0].TenantSlug

		// Act: Create second user with same name "Test User"
		user2, err := userService.GetOrCreate(context.Background(), "auth0|test-user-2", "Test", "User")
		require.NoError(t, err)

		memberships2, err := tenantRepo.ListUserTenants(context.Background(), user2.ID)
		require.NoError(t, err)
		require.Len(t, memberships2, 1)
		slug2 := memberships2[0].TenantSlug

		// Assert: Slugs must be different
		assert.NotEqual(t, slug1, slug2, "Slugs for tenants with same name should be unique")
		assert.NotEmpty(t, slug1)
		assert.NotEmpty(t, slug2)

		// Cleanup committed data
		_, _ = db.Exec("DELETE FROM tenant_members WHERE user_id IN ($1, $2)", user1.ID, user2.ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id IN ($1, $2)", *user1.LastActiveTenantID, *user2.LastActiveTenantID)
		_, _ = db.Exec("DELETE FROM users WHERE id IN ($1, $2)", user1.ID, user2.ID)
	})
}
