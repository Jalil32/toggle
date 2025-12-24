package validator

import (
	"context"

	"github.com/jmoiron/sqlx"

	pkgErrors "github.com/jalil32/toggle/internal/pkg/errors"
)

// Validator is the interface for tenant validation operations
type Validator interface {
	ValidateProjectOwnership(ctx context.Context, projectID, tenantID string) error
	ValidateTenantExists(ctx context.Context, tenantID string) error
}

// TenantValidator provides reusable tenant ownership validation
type TenantValidator struct {
	db *sqlx.DB
}

// NewTenantValidator creates a new TenantValidator instance
func NewTenantValidator(db *sqlx.DB) *TenantValidator {
	return &TenantValidator{db: db}
}

// ValidateProjectOwnership verifies that a project belongs to a specific tenant
// Returns ErrProjectNotInTenant if the project doesn't exist OR doesn't belong to the tenant
// This prevents enumeration attacks by not revealing whether the project exists
func (v *TenantValidator) ValidateProjectOwnership(ctx context.Context, projectID, tenantID string) error {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND tenant_id = $2)`

	err := v.db.GetContext(ctx, &exists, query, projectID, tenantID)
	if err != nil {
		return err
	}

	if !exists {
		// Return generic error - don't reveal if project exists in another tenant
		return pkgErrors.ErrProjectNotInTenant
	}

	return nil
}

// ValidateTenantExists verifies that a tenant exists
func (v *TenantValidator) ValidateTenantExists(ctx context.Context, tenantID string) error {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)`

	err := v.db.GetContext(ctx, &exists, query, tenantID)
	if err != nil {
		return err
	}

	if !exists {
		return pkgErrors.ErrInvalidTenant
	}

	return nil
}
