package projects

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"

	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, tenantID, name string) (*Project, error)
	GetByID(ctx context.Context, id string, tenantID string) (*Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*Project, error)
	ListByTenantID(ctx context.Context, tenantID string) ([]Project, error)
	Delete(ctx context.Context, id string, tenantID string) error
}

type postgresRepo struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

// getDB returns the transaction from context if present, otherwise returns the DB
func (r *postgresRepo) getDB(ctx context.Context) sqlx.ExtContext {
	if tx, ok := transaction.GetTx(ctx); ok {
		return tx
	}
	return r.db
}

func (r *postgresRepo) Create(ctx context.Context, tenantID, name string) (*Project, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	var project Project
	err = r.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO projects (tenant_id, name, client_api_key)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, name, client_api_key, created_at, updated_at
	`, tenantID, name, apiKey).StructScan(&project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string, tenantID string) (*Project, error) {
	var project Project
	executor := r.getDB(ctx)

	err := sqlx.GetContext(ctx, executor, &project, `
		SELECT id, tenant_id, name, client_api_key, created_at, updated_at
		FROM projects WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *postgresRepo) GetByAPIKey(ctx context.Context, apiKey string) (*Project, error) {
	var project Project
	executor := r.getDB(ctx)

	err := sqlx.GetContext(ctx, executor, &project, `
		SELECT id, tenant_id, name, client_api_key, created_at, updated_at
		FROM projects WHERE client_api_key = $1
	`, apiKey)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *postgresRepo) ListByTenantID(ctx context.Context, tenantID string) ([]Project, error) {
	projects := []Project{} // Initialize as empty slice instead of nil
	executor := r.getDB(ctx)

	err := sqlx.SelectContext(ctx, executor, &projects, `
		SELECT id, tenant_id, name, client_api_key, created_at, updated_at
		FROM projects WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *postgresRepo) Delete(ctx context.Context, id string, tenantID string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
		DELETE FROM projects WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
