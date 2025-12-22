package organizations

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, name string) (*Organization, error)
	GetByID(ctx context.Context, id string) (*Organization, error)
	Update(ctx context.Context, id, name string) (*Organization, error)
}

type postgresRepo struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, name string) (*Organization, error) {
	var org Organization
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO organizations (name)
		VALUES ($1)
		RETURNING id, name, created_at, updated_at
	`, name).StructScan(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*Organization, error) {
	var org Organization
	err := r.db.GetContext(ctx, &org, `
		SELECT id, name, created_at, updated_at
		FROM organizations WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *postgresRepo) Update(ctx context.Context, id, name string) (*Organization, error) {
	var org Organization
	err := r.db.QueryRowxContext(ctx, `
		UPDATE organizations 
		SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, created_at, updated_at
	`, name, id).StructScan(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}
