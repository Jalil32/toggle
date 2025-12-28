package users_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jalil32/toggle/internal/pkg/transaction"
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

// TestGetUser_Success tests successfully retrieving a user by ID
func TestGetUser_Success(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, logger)

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Create test user
		user := testutil.CreateUser(t, tx, "Test User", "test@example.com")

		// Inject transaction into context
		ctx = transaction.InjectTx(ctx, tx)

		// Act: Get user by ID
		retrieved, err := userService.GetUser(ctx, user.ID)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.EmailVerified, retrieved.EmailVerified)
	})
}

// TestGetUser_NotFound tests error handling when user doesn't exist
func TestGetUser_NotFound(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, logger)

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		ctx = transaction.InjectTx(ctx, tx)

		// Act: Try to get non-existent user
		fakeID := "00000000-0000-0000-0000-999999999999"
		user, err := userService.GetUser(ctx, fakeID)

		// Assert
		require.Error(t, err)
		assert.Nil(t, user)
	})
}

// TestUpdateLastActiveTenant tests updating a user's last active tenant
func TestUpdateLastActiveTenant(t *testing.T) {
	db := testutil.GetTestDB()
	userRepo := users.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userService := users.NewService(userRepo, logger)

	testutil.WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
		// Create test data
		user := testutil.CreateUser(t, tx, "Test User", "test@example.com")
		tenant := testutil.CreateTenant(t, tx, "Test Tenant", "test-tenant")

		ctx = transaction.InjectTx(ctx, tx)

		// Act: Update last active tenant
		err := userService.UpdateLastActiveTenant(ctx, user.ID, tenant.ID)

		// Assert
		require.NoError(t, err)

		// Verify the update
		updated, err := userService.GetUser(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, updated.LastActiveTenantID)
		assert.Equal(t, tenant.ID, *updated.LastActiveTenantID)
	})
}
