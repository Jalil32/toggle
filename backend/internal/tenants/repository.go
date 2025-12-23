package tenants

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jalil32/toggle/internal/pkg/transaction"
)

type Repository interface {
	// Tenant operations
	Create(ctx context.Context, name, slug string) (*Tenant, error)
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	Update(ctx context.Context, id, name string) (*Tenant, error)

	// Membership operations
	GetMembership(ctx context.Context, userID, tenantID string) (string, error)
	HasMemberships(ctx context.Context, userID string) (bool, error)
	CreateMembership(ctx context.Context, userID, tenantID, role string) error
	ListUserTenants(ctx context.Context, userID string) ([]*TenantMembership, error)
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

func (r *postgresRepo) Create(ctx context.Context, name, slug string) (*Tenant, error) {
	var tenant Tenant
	executor := r.getExecutor(ctx)

	query := `
		INSERT INTO tenants (name, slug)
		VALUES ($1, $2)
		RETURNING id, name, slug, created_at, updated_at
	`

	err := sqlx.GetContext(ctx, executor, &tenant, query, name, slug)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*Tenant, error) {
	var tenant Tenant
	executor := r.getExecutor(ctx)

	err := sqlx.GetContext(ctx, executor, &tenant, `
		SELECT id, name, slug, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *postgresRepo) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var tenant Tenant
	executor := r.getExecutor(ctx)

	err := sqlx.GetContext(ctx, executor, &tenant, `
		SELECT id, name, slug, created_at, updated_at
		FROM tenants WHERE slug = $1
	`, slug)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *postgresRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	executor := r.getExecutor(ctx)

	err := sqlx.GetContext(ctx, executor, &exists, `
		SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)
	`, slug)
	return exists, err
}

func (r *postgresRepo) Update(ctx context.Context, id, name string) (*Tenant, error) {
	var tenant Tenant
	executor := r.getExecutor(ctx)

	query := `
		UPDATE tenants
		SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, slug, created_at, updated_at
	`

	err := sqlx.GetContext(ctx, executor, &tenant, query, name, id)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// Membership repository methods

// GetMembership returns the role of a user in a tenant
// Returns empty string if user is not a member
func (r *postgresRepo) GetMembership(ctx context.Context, userID, tenantID string) (string, error) {
	var role string
	executor := r.getExecutor(ctx)

	query := `SELECT role FROM tenant_members WHERE user_id = $1 AND tenant_id = $2`

	err := sqlx.GetContext(ctx, executor, &role, query, userID, tenantID)
	if err != nil {
		// Return empty string for no membership
		return "", nil
	}

	return role, nil
}

// HasMemberships checks if a user has any tenant memberships
func (r *postgresRepo) HasMemberships(ctx context.Context, userID string) (bool, error) {
	var count int
	executor := r.getExecutor(ctx)

	query := `SELECT COUNT(*) FROM tenant_members WHERE user_id = $1`

	err := sqlx.GetContext(ctx, executor, &count, query, userID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateMembership creates a new tenant membership
func (r *postgresRepo) CreateMembership(ctx context.Context, userID, tenantID, role string) error {
	executor := r.getExecutor(ctx)

	query := `
		INSERT INTO tenant_members (user_id, tenant_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, tenant_id) DO UPDATE SET role = $3, updated_at = NOW()
	`

	_, err := executor.ExecContext(ctx, query, userID, tenantID, role)
	return err
}

// ListUserTenants returns all tenants that a user is a member of
func (r *postgresRepo) ListUserTenants(ctx context.Context, userID string) ([]*TenantMembership, error) {
	executor := r.getExecutor(ctx)

	query := `
		SELECT
			tm.tenant_id,
			tm.role,
			t.name as tenant_name,
			t.slug as tenant_slug
		FROM tenant_members tm
		INNER JOIN tenants t ON tm.tenant_id = t.id
		WHERE tm.user_id = $1
		ORDER BY tm.created_at ASC
	`

	var memberships []*TenantMembership
	err := sqlx.SelectContext(ctx, executor, &memberships, query, userID)
	if err != nil {
		return nil, err
	}

	return memberships, nil
}
