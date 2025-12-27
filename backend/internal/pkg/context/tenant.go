package context

import (
	"context"
	"errors"
)

type contextKey string

const (
	tenantIDKey  contextKey = "tenant_id"
	userRoleKey  contextKey = "user_role"
	userIDKey    contextKey = "user_id"
	projectIDKey contextKey = "project_id"
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
func WithAuth(ctx context.Context, userID, tenantID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	return ctx
}

// WithUserOnly adds only user authentication values to the context (no tenant info)
// This is used for new users who haven't created a tenant yet
func WithUserOnly(ctx context.Context, userID string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
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

// WithSDKAuth adds project and tenant context for SDK requests
// This is used by the API key middleware for SDK authentication
func WithSDKAuth(ctx context.Context, projectID, tenantID string) context.Context {
	ctx = context.WithValue(ctx, projectIDKey, projectID)
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	return ctx
}

// ProjectID extracts project ID from context (for SDK requests)
func ProjectID(ctx context.Context) (string, error) {
	val := ctx.Value(projectIDKey)
	if val == nil {
		return "", errors.New("project context not found")
	}
	projectID, ok := val.(string)
	if !ok {
		return "", errors.New("project context not found")
	}
	return projectID, nil
}

// MustProjectID extracts project ID and panics if not found
func MustProjectID(ctx context.Context) string {
	projectID, err := ProjectID(ctx)
	if err != nil {
		panic("project context not found - SDK middleware not configured correctly")
	}
	return projectID
}
