package users

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, auth0ID, orgID, email, firstname, lastname, role string) (*User, error)
	GetByAuth0ID(ctx context.Context, auth0ID string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}

type postgresRepo struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, auth0ID, orgID, email, firstname, lastname, role string) (*User, error) {
	var user User
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO users (auth0_id, organization_id, email, firstname, lastname, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, auth0_id, organization_id, email, firstname, lastname, role, created_at, updated_at
	`, auth0ID, orgID, email, firstname, lastname, role).StructScan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepo) GetByAuth0ID(ctx context.Context, auth0ID string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, auth0_id, organization_id, email, firstname, lastname, role, created_at, updated_at
		FROM users WHERE auth0_id = $1
	`, auth0ID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT id, auth0_id, organization_id, email, firstname, lastname, role, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

var ErrNotFound = sql.ErrNoRows
