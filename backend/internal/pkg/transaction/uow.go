package transaction

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// UnitOfWork manages database transactions
type UnitOfWork interface {
	// RunInTransaction executes the given function within a database transaction
	// If fn returns an error, the transaction is rolled back
	// If fn returns nil, the transaction is committed
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type unitOfWork struct {
	db *sqlx.DB
}

// NewUnitOfWork creates a new unit of work
func NewUnitOfWork(db *sqlx.DB) UnitOfWork {
	return &unitOfWork{db: db}
}

type txKey struct{}

// RunInTransaction executes fn within a database transaction
func (u *unitOfWork) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Start transaction
	tx, err := u.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Safe to call even after commit
	}()

	// Inject transaction into context
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Execute business logic
	if err := fn(txCtx); err != nil {
		return err // Rollback happens via defer
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetTx extracts the transaction from context, if present
func GetTx(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sqlx.Tx)
	return tx, ok
}
