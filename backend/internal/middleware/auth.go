package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jalil32/toggle/config"
	"github.com/jalil32/toggle/internal/auth"
	appContext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/users"
)

func Auth(cfg *config.Config, logger *slog.Logger, userService *users.Service, tenantService *tenants.Service) gin.HandlerFunc {
	// Dev mode - skip auth
	if cfg.JWT.SkipAuth {
		logger.Warn("auth middleware disabled - SKIP_AUTH is true")
		return devModeMiddleware(logger, userService, tenantService)
	}

	// Validate JWT config
	if cfg.JWT.JWKSURL == "" || cfg.JWT.Issuer == "" || cfg.JWT.Audience == "" {
		panic("JWT_JWKS_URL, JWT_ISSUER, and JWT_AUDIENCE must be set when SKIP_AUTH is false")
	}

	// Create JWT verifier
	verifier := auth.NewJWTVerifier(cfg.JWT.JWKSURL, cfg.JWT.Issuer, cfg.JWT.Audience)

	logger.Info("auth middleware initialized",
		slog.String("jwks_url", cfg.JWT.JWKSURL),
		slog.String("issuer", cfg.JWT.Issuer),
		slog.String("audience", cfg.JWT.Audience),
	)

	return func(c *gin.Context) {
		// Extract and verify JWT token
		token, err := auth.ExtractTokenFromHeader(c.GetHeader("Authorization"))
		if err != nil {
			logger.Debug("missing or invalid authorization header",
				slog.String("path", c.Request.URL.Path),
				slog.String("method", c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		claims, err := verifier.VerifyToken(c.Request.Context(), token)
		if err != nil {
			logger.Warn("token validation failed",
				slog.String("error", err.Error()),
				slog.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Extract user ID from JWT claims (this is users.id UUID)
		userID := claims.UserID
		if userID == "" {
			logger.Warn("missing userId in token claims")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user identifier in token"})
			return
		}

		// Get user from database
		user, err := userService.GetUser(c.Request.Context(), userID)
		if err != nil {
			logger.Error("failed to get user",
				slog.String("error", err.Error()),
				slog.String("user_id", userID),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}

		// Get user's tenant memberships
		memberships, err := tenantService.ListUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			logger.Error("failed to get user memberships",
				slog.String("user_id", user.ID),
				slog.String("error", err.Error()),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user memberships"})
			return
		}

		// If user has no tenant memberships, set context with just user info
		// This allows new users to access /me/* routes to create their first tenant
		if len(memberships) == 0 {
			logger.Debug("user authenticated without tenant",
				slog.String("user_id", user.ID),
			)

			// Set authentication context without tenant info
			ctx := appContext.WithUserOnly(c.Request.Context(), user.ID)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		// Use last active tenant if set, otherwise use first membership
		var activeMembership *tenants.TenantMembership
		if user.LastActiveTenantID != nil {
			for _, m := range memberships {
				if m.TenantID == *user.LastActiveTenantID {
					activeMembership = m
					break
				}
			}
		}
		if activeMembership == nil {
			activeMembership = memberships[0]
		}

		logger.Debug("user authenticated",
			slog.String("user_id", user.ID),
			slog.String("tenant_id", activeMembership.TenantID),
			slog.String("role", activeMembership.Role),
		)

		// Set authentication context
		ctx := appContext.WithAuth(
			c.Request.Context(),
			user.ID,
			activeMembership.TenantID,
			activeMembership.Role,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// devModeMiddleware provides a development mode authentication bypass
func devModeMiddleware(logger *slog.Logger, userService *users.Service, tenantService *tenants.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use a hardcoded dev user UUID
		devUserID := "00000000-0000-0000-0000-000000000001"

		user, err := userService.GetUser(c.Request.Context(), devUserID)
		if err != nil {
			logger.Error("failed to get dev user",
				slog.String("error", err.Error()),
				slog.String("dev_user_id", devUserID),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "dev user not found - please create a user with ID " + devUserID + " in the database",
			})
			return
		}

		// Get user's tenant memberships
		memberships, err := tenantService.ListUserTenants(c.Request.Context(), user.ID)
		if err != nil || len(memberships) == 0 {
			logger.Error("failed to get user memberships",
				slog.String("user_id", user.ID),
				slog.String("error", err.Error()),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "user has no tenant memberships"})
			return
		}

		// Use last active tenant if set, otherwise use first membership
		var activeMembership *tenants.TenantMembership
		if user.LastActiveTenantID != nil {
			for _, m := range memberships {
				if m.TenantID == *user.LastActiveTenantID {
					activeMembership = m
					break
				}
			}
		}
		if activeMembership == nil {
			activeMembership = memberships[0]
		}

		logger.Debug("dev user authenticated",
			slog.String("user_id", user.ID),
			slog.String("tenant_id", activeMembership.TenantID),
			slog.String("role", activeMembership.Role),
		)

		// Set authentication context
		ctx := appContext.WithAuth(
			c.Request.Context(),
			user.ID,
			activeMembership.TenantID,
			activeMembership.Role,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
