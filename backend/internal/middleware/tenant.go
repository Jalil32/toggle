package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/tenants"
)

// Tenant middleware validates tenant membership and injects tenant context
// This middleware must run AFTER the Auth middleware
func Tenant(tenantRepo tenants.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user_id from Go context (set by auth middleware)
		userID := appContext.MustUserID(c.Request.Context())

		// Extract tenant_id from X-Tenant-ID header
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
			c.Abort()
			return
		}

		// Verify user has access to this tenant
		role, err := tenantRepo.GetMembership(c.Request.Context(), userID, tenantID)
		if err != nil {
			logger.Error("tenant middleware: failed to verify tenant access",
				slog.String("user_id", userID),
				slog.String("tenant_id", tenantID),
				slog.String("error", err.Error()),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify tenant access"})
			c.Abort()
			return
		}

		if role == "" {
			logger.Warn("tenant middleware: user denied access to tenant",
				slog.String("user_id", userID),
				slog.String("tenant_id", tenantID),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this tenant"})
			c.Abort()
			return
		}

		// IMPORTANT: This middleware OVERWRITES tenant_id and role from Auth middleware
		// when X-Tenant-ID header is present (tenant switching).
		// user_id and auth0_id remain unchanged from Auth middleware.
		ctx := appContext.WithTenant(c.Request.Context(), tenantID, role)
		c.Request = c.Request.WithContext(ctx)

		logger.Debug("tenant middleware: tenant context set",
			slog.String("user_id", userID),
			slog.String("tenant_id", tenantID),
			slog.String("role", role),
		)

		c.Next()
	}
}
