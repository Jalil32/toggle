package middleware

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	appContext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/projects"
)

// APIKey middleware authenticates SDK requests using client_api_key
// and injects project_id and tenant_id into context
func APIKey(projectRepo projects.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Debug("SDK request missing authorization header",
				slog.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract Bearer token
		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		if apiKey == authHeader || apiKey == "" {
			logger.Debug("invalid authorization header format",
				slog.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		// Lookup project by API key
		project, err := projectRepo.GetByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Warn("invalid API key",
					slog.String("path", c.Request.URL.Path),
				)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
				c.Abort()
				return
			}
			logger.Error("failed to validate API key",
				slog.String("error", err.Error()),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
			c.Abort()
			return
		}

		// Inject project and tenant context (similar to tenant middleware)
		ctx := appContext.WithSDKAuth(c.Request.Context(), project.ID, project.TenantID)
		c.Request = c.Request.WithContext(ctx)

		logger.Debug("SDK request authenticated",
			slog.String("project_id", project.ID),
			slog.String("tenant_id", project.TenantID),
		)

		c.Next()
	}
}
