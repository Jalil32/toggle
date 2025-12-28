package users

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jalil32/toggle/internal/pkg/transaction"
)

type Repository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	UpdateLastActiveTenant(ctx context.Context, userID, tenantID string) error
}

type postgresRepo struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

// getExecutor returns the appropriate database executor (transaction or connection)
func (r *postgresRepo) getExecutor(ctx context.Context) sqlx.ExtContext {
	if tx, ok := transaction.GetTx(ctx); ok {
		return tx
	}
	return r.db
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	executor := r.getExecutor(ctx)

	err := sqlx.GetContext(ctx, executor, &user, `
		SELECT id, name, email, email_verified, image, last_active_tenant_id, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepo) UpdateLastActiveTenant(ctx context.Context, userID, tenantID string) error {
	executor := r.getExecutor(ctx)

	query := `
		UPDATE users
		SET last_active_tenant_id = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := executor.ExecContext(ctx, query, tenantID, userID)
	return err
}

var ErrNotFound = sql.ErrNoRows
