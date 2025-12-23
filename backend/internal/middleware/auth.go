package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"

	"github.com/jalil32/toggle/config"
	appContext "github.com/jalil32/toggle/internal/pkg/context"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/users"
)

type Claims struct {
}

func (c *Claims) Validate(ctx context.Context) error {
	return nil
}

func Auth(cfg *config.Config, logger *slog.Logger, userService *users.Service, tenantService *tenants.Service) gin.HandlerFunc {
	// Dev mode - skip auth
	if cfg.Auth0.SkipAuth {
		logger.Warn("auth middleware disabled - SKIP_AUTH is true")
		return func(c *gin.Context) {
			// Get or create a dev user in the database
			user, err := userService.GetOrCreate(
				c.Request.Context(),
				"dev-auth0-id",
				"test_first",
				"test_last",
			)
			if err != nil {
				logger.Error("failed to get/create dev user",
					slog.String("error", err.Error()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize dev user"})
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
				user.Auth0ID,
			)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}

	// Validate config
	if cfg.Auth0.Domain == "" || cfg.Auth0.Audience == "" {
		panic("AUTH0_DOMAIN and AUTH0_AUDIENCE must be set when SKIP_AUTH is false")
	}

	issuerURL, err := url.Parse("https://" + cfg.Auth0.Domain + "/")
	if err != nil {
		panic("invalid AUTH0_DOMAIN: " + err.Error())
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{cfg.Auth0.Audience},
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &Claims{}
		}),
	)
	if err != nil {
		panic("failed to create jwt validator: " + err.Error())
	}

	logger.Info("auth middleware initialized",
		slog.String("domain", cfg.Auth0.Domain),
		slog.String("audience", cfg.Auth0.Audience),
	)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Debug("missing authorization header",
				slog.String("path", c.Request.URL.Path),
				slog.String("method", c.Request.Method),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtValidator.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Warn("token validation failed",
				slog.String("error", err.Error()),
				slog.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		validatedClaims := claims.(*validator.ValidatedClaims)
		auth0ID := validatedClaims.RegisteredClaims.Subject

		if auth0ID == "" {
			logger.Warn("missing auth0 id in token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user identifier in token"})
			return
		}

		// Use auth0ID as name placeholder (name claim not configured in Auth0 yet)
		user, err := userService.GetOrCreate(
			c.Request.Context(),
			auth0ID,
			auth0ID,
			auth0ID,
		)

		if err != nil {
			logger.Error("failed to sync user",
				slog.String("error", err.Error()),
				slog.String("auth0_id", auth0ID),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to sync user"})
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
			user.Auth0ID,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
