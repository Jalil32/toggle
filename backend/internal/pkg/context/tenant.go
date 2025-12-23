package context

import (
	"context"
	"errors"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	userRoleKey contextKey = "user_role"
	userIDKey   contextKey = "user_id"
	auth0IDKey  contextKey = "auth0_id"
)

var (
	ErrNoTenantContext = errors.New("tenant context not found")
	ErrNoUserContext   = errors.New("user context not found")
)

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

// MustTenantID extracts the tenant ID from the context and panics if not found
// Use this in handlers after middleware has validated tenant context
func MustTenantID(ctx context.Context) string {
	tenantID, err := TenantID(ctx)
	if err != nil {
		panic("tenant context not found - middleware not configured correctly")
	}
	return tenantID
}

// WithAuth adds all authentication values to the context
func WithAuth(ctx context.Context, userID, tenantID, role, auth0ID string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	ctx = context.WithValue(ctx, auth0IDKey, auth0ID)
	return ctx
}

// UserID extracts the user ID from the context
func UserID(ctx context.Context) (string, error) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return "", ErrNoUserContext
	}
	userID, ok := val.(string)
	if !ok {
		return "", ErrNoUserContext
	}
	return userID, nil
}

// MustUserID extracts the user ID from the context and panics if not found
// Use this in handlers after middleware has validated user context
func MustUserID(ctx context.Context) string {
	userID, err := UserID(ctx)
	if err != nil {
		panic("user context not found - middleware not configured correctly")
	}
	return userID
}

// Auth0ID extracts the Auth0 ID from the context
func Auth0ID(ctx context.Context) string {
	val := ctx.Value(auth0IDKey)
	if val == nil {
		return ""
	}
	auth0ID, _ := val.(string)
	return auth0ID
}
