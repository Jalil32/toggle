package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	tenantCtx "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/tenants"
)

// Tenant middleware validates tenant membership and injects tenant context
// This middleware must run AFTER the Auth middleware
func Tenant(tenantRepo tenants.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user_id from previous auth middleware
		userIDVal, exists := c.Get("user_id")
		if !exists {
			logger.Error("tenant middleware: user_id not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			logger.Error("tenant middleware: user_id is not a string")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}

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

		// Inject tenant context into request context
		ctx := tenantCtx.WithTenant(c.Request.Context(), tenantID, role)
		c.Request = c.Request.WithContext(ctx)

		// Also set in Gin context for backward compatibility
		c.Set("tenant_id", tenantID)
		c.Set("user_role", role)
		// Keep org_id for backward compatibility (points to tenant_id)
		c.Set("org_id", tenantID)

		logger.Debug("tenant middleware: tenant context set",
			slog.String("user_id", userID),
			slog.String("tenant_id", tenantID),
			slog.String("role", role),
		)

		c.Next()
	}
}
