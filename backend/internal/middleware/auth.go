package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"

	"github.com/jalil32/toggle/config"
	"github.com/jalil32/toggle/internal/users"
)

type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (c *Claims) Validate(ctx context.Context) error {
	return nil
}

func Middleware(cfg *config.Config, userService *users.Service) gin.HandlerFunc {
	// Dev mode - skip auth
	if cfg.Auth0.SkipAuth {
		return func(c *gin.Context) {
			c.Set("user_id", "dev-user-id")
			c.Set("org_id", "dev-org-id")
			c.Set("role", "owner")
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

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtValidator.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		validatedClaims := claims.(*validator.ValidatedClaims)
		customClaims := validatedClaims.CustomClaims.(*Claims)
		auth0ID := validatedClaims.RegisteredClaims.Subject

		// Get or create user in our DB
		user, err := userService.GetOrCreate(
			c.Request.Context(),
			auth0ID,
			customClaims.Email,
			customClaims.Name,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to sync user"})
			return
		}

		c.Set("user_id", user.ID)
		c.Set("org_id", user.OrganizationID)
		c.Set("role", user.Role)

		c.Next()
	}
}
