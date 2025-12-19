package projects

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, orgID, name string) (*Project, error)
	GetByID(ctx context.Context, id string) (*Project, error)
	ListByOrgID(ctx context.Context, orgID string) ([]Project, error)
	Delete(ctx context.Context, id string) error
}

type postgresRepo struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, orgID, name string) (*Project, error) {
	apiKey, _ := generateAPIKey()

	var project Project
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO projects (organization_id, name, client_api_key)
		VALUES ($1, $2, $3)
		RETURNING id, organization_id, name, client_api_key, created_at, updated_at
	`, orgID, name, apiKey).StructScan(&project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*Project, error) {
	var project Project
	err := r.db.GetContext(ctx, &project, `
		SELECT id, organization_id, name, client_api_key, created_at, updated_at
		FROM projects WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *postgresRepo) ListByOrgID(ctx context.Context, orgID string) ([]Project, error) {
	var projects []Project
	err := r.db.SelectContext(ctx, &projects, `
		SELECT id, organization_id, name, client_api_key, created_at, updated_at
		FROM projects WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *postgresRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes), nil
}
