package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB     *sqlx.DB
	setupOnce  sync.Once
	container  *postgres.PostgresContainer
	setupError error
)

// SetupTestDatabase initializes a PostgreSQL testcontainer for the entire test suite.
// Call this from TestMain to set up the database once for all tests.
func SetupTestDatabase(ctx context.Context, migrationsDir string) (*sqlx.DB, error) {
	setupOnce.Do(func() {
		// Start PostgreSQL container
		container, setupError = postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase("toggle_test"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		if setupError != nil {
			setupError = fmt.Errorf("failed to start postgres container: %w", setupError)
			return
		}

		// Get connection string
		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setupError = fmt.Errorf("failed to get connection string: %w", err)
			return
		}

		// Connect to database
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			setupError = fmt.Errorf("failed to connect to database: %w", err)
			return
		}

		// Verify connection
		if err := db.Ping(); err != nil {
			setupError = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		// Run migrations using goose
		if err := goose.SetDialect("postgres"); err != nil {
			setupError = fmt.Errorf("failed to set goose dialect: %w", err)
			return
		}

		// Convert relative path to absolute if needed
		absPath, err := filepath.Abs(migrationsDir)
		if err != nil {
			setupError = fmt.Errorf("failed to get absolute path for migrations: %w", err)
			return
		}

		if err := goose.Up(db, absPath); err != nil {
			setupError = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		// Wrap with sqlx
		testDB = sqlx.NewDb(db, "postgres")

		log.Printf("✅ Test database initialized successfully")
	})

	if setupError != nil {
		return nil, setupError
	}

	return testDB, nil
}

// TeardownTestDatabase cleans up the test database container.
// Call this from TestMain's defer or cleanup.
func TeardownTestDatabase(ctx context.Context) error {
	if testDB != nil {
		if err := testDB.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	if container != nil {
		if err := container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	log.Printf("✅ Test database cleaned up successfully")
	return nil
}

// GetTestDB returns the global test database connection.
// Panics if SetupTestDatabase hasn't been called.
func GetTestDB() *sqlx.DB {
	if testDB == nil {
		panic("testDB is nil - did you call SetupTestDatabase in TestMain?")
	}
	return testDB
}

// WithTestDB provides a transactional test environment using the UoW pattern.
// The transaction is injected into the context and ALWAYS rolled back after the test,
// ensuring test isolation without database cleanup overhead.
//
// Usage:
//
//	WithTestDB(t, func(ctx context.Context, tx *sqlx.Tx) {
//	    repo := NewRepository(testutil.GetTestDB()) // Pass the DB, not the tx
//	    // Use ctx in repository calls - it contains the transaction
//	    repo.Create(ctx, flag)
//	})
func WithTestDB(t *testing.T, fn func(ctx context.Context, tx *sqlx.Tx)) {
	t.Helper()

	db := GetTestDB()

	// Begin transaction
	tx, err := db.Beginx()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Always rollback (even on panic)
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			t.Errorf("failed to rollback transaction: %v", err)
		}
	}()

	// Inject transaction into context using the transaction package's helper
	ctx := transaction.InjectTx(context.Background(), tx)

	// Run the test
	fn(ctx, tx)
}
