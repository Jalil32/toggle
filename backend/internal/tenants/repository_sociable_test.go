package tenants_test

import (
	"context"
	"os"
	"testing"

	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

// TestRepository_Create_DuplicateSlug_Fails tests that creating a tenant
// with a duplicate slug fails with a unique constraint violation
func TestRepository_Create_DuplicateSlug_Fails(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = context.WithValue(ctx, "tx", tx)

		// Create first tenant with slug "acme-corp"
		tenant1, err := repo.Create(ctx, "Acme Corp", "acme-corp")
		require.NoError(t, err)
		require.NotNil(t, tenant1)
		assert.Equal(t, "acme-corp", tenant1.Slug)

		// Attempt to create second tenant with same slug
		tenant2, err := repo.Create(ctx, "Acme Corporation", "acme-corp")

		// Assert: Should fail with unique constraint violation
		require.Error(t, err)
		assert.Nil(t, tenant2)

		// Verify it's a PostgreSQL unique violation error
		pqErr, ok := err.(*pq.Error)
		require.True(t, ok, "Error should be a PostgreSQL error")
		assert.Equal(t, pq.ErrorCode("23505"), pqErr.Code, "Should be unique_violation error")
	})
}

// TestRepository_SlugExists_ReturnsCorrectly tests that SlugExists
// accurately reports whether a slug is already taken
func TestRepository_SlugExists_ReturnsCorrectly(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = context.WithValue(ctx, "tx", tx)

		// Check non-existent slug
		exists, err := repo.SlugExists(ctx, "nonexistent-slug")
		require.NoError(t, err)
		assert.False(t, exists, "Non-existent slug should return false")

		// Create a tenant
		tenant := testutil.CreateTenant(t, tx, "Tech Startup", "tech-startup")

		// Check existing slug
		exists, err = repo.SlugExists(ctx, "tech-startup")
		require.NoError(t, err)
		assert.True(t, exists, "Existing slug should return true")

		// Check different slug
		exists, err = repo.SlugExists(ctx, "different-slug")
		require.NoError(t, err)
		assert.False(t, exists, "Different slug should return false")

		// Verify tenant was created
		assert.NotEmpty(t, tenant.ID)
	})
}

// TestRepository_GetBySlug_ReturnsCorrectTenant tests that GetBySlug
// retrieves the correct tenant by slug
func TestRepository_GetBySlug_ReturnsCorrectTenant(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = context.WithValue(ctx, "tx", tx)

		// Create multiple tenants
		tenant1 := testutil.CreateTenant(t, tx, "Company A", "company-a")
		tenant2 := testutil.CreateTenant(t, tx, "Company B", "company-b")
		tenant3 := testutil.CreateTenant(t, tx, "Company C", "company-c")

		// Test: Retrieve tenant by slug
		retrieved, err := repo.GetBySlug(ctx, "company-b")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Assert: Correct tenant retrieved
		assert.Equal(t, tenant2.ID, retrieved.ID)
		assert.Equal(t, "Company B", retrieved.Name)
		assert.Equal(t, "company-b", retrieved.Slug)

		// Verify other tenants exist but weren't returned
		assert.NotEqual(t, tenant1.ID, retrieved.ID)
		assert.NotEqual(t, tenant3.ID, retrieved.ID)
	})
}

// TestRepository_GetBySlug_NonExistent_ReturnsError tests that GetBySlug
// returns an error when slug doesn't exist
func TestRepository_GetBySlug_NonExistent_ReturnsError(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = context.WithValue(ctx, "tx", tx)

		// Test: Retrieve non-existent slug
		retrieved, err := repo.GetBySlug(ctx, "nonexistent-slug")

		// Assert: Should return error
		require.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

// TestRepository_CaseSensitiveSlug tests that slugs are case-sensitive
// (e.g., "Acme-Corp" and "acme-corp" are different)
func TestRepository_CaseSensitiveSlug(t *testing.T) {
	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		repo := tenants.NewRepository(testutil.GetTestDB())
		ctx = context.WithValue(ctx, "tx", tx)

		// Create tenant with lowercase slug
		tenant1, err := repo.Create(ctx, "Acme Corp", "acme-corp")
		require.NoError(t, err)
		assert.Equal(t, "acme-corp", tenant1.Slug)

		// Create tenant with uppercase slug (should succeed if case-sensitive)
		tenant2, err := repo.Create(ctx, "ACME Corp", "ACME-CORP")
		require.NoError(t, err, "Different case slugs should be allowed")
		assert.Equal(t, "ACME-CORP", tenant2.Slug)

		// Verify both exist
		exists1, err := repo.SlugExists(ctx, "acme-corp")
		require.NoError(t, err)
		assert.True(t, exists1)

		exists2, err := repo.SlugExists(ctx, "ACME-CORP")
		require.NoError(t, err)
		assert.True(t, exists2)
	})
}
