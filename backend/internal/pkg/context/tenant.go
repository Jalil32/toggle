package context

import (
	"context"
	"errors"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	userRoleKey contextKey = "user_role"
)

var ErrNoTenantContext = errors.New("tenant context not found")

// WithTenant adds tenant ID and role to the context
func WithTenant(ctx context.Context, tenantID string, role string) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	return ctx
}

// TenantID extracts the tenant ID from the context
func TenantID(ctx context.Context) (string, error) {
	val := ctx.Value(tenantIDKey)
	if val == nil {
		return "", ErrNoTenantContext
	}
	tenantID, ok := val.(string)
	if !ok {
		return "", ErrNoTenantContext
	}
	return tenantID, nil
}

// UserRole extracts the user role from the context
func UserRole(ctx context.Context) string {
	val := ctx.Value(userRoleKey)
	if val == nil {
		return ""
	}
	role, _ := val.(string)
	return role
}
