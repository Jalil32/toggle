package testutil_test

import (
	"context"
	"os"
	"testing"

	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jmoiron/sqlx"
)

// TestMain sets up the test database container once for all tests in this package.
// This is an example of how to use the testutil package in your test files.
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

// Example test showing how to use WithTestDB for transactional isolation
func TestExample_WithTestDB(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Create test data using tx directly (for fixtures)
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")
		user := testutil.CreateUser(t, tx, "auth0|123", "test@example.com", "John", "Doe")
		testutil.CreateTenantMember(t, tx, user.ID, tenant.ID, "owner")

		// Use ctx for repository calls (ctx contains the transaction)
		// repo := NewRepository(testutil.GetTestDB())
		// repo.Create(ctx, item)

		// Everything will be rolled back automatically
	})
}
