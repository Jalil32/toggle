package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/jalil32/toggle/config"
	"github.com/jalil32/toggle/internal/evaluation"
	flags "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/middleware"
	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/pkg/validator"
	"github.com/jalil32/toggle/internal/projects"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/users"
)

func Routes(router *gin.Engine, logger *slog.Logger, cfg *config.Config, db *sqlx.DB) error {
	// Unit of Work
	uow := transaction.NewUnitOfWork(db)

	// Validators
	tenantValidator := validator.NewTenantValidator(db)

	// Repositories
	tenantRepo := tenants.NewRepository(db)
	userRepo := users.NewRepository(db)
	projectRepo := projects.NewRepository(db)
	flagRepo := flags.NewRepository(db)

	// Services
	tenantService := tenants.NewService(tenantRepo, uow, logger)
	userService := users.NewService(userRepo, logger)

	// Inject users repo into tenant service (to avoid circular dependency)
	tenantService.SetUsersRepo(userRepo)

	projectService := projects.NewService(projectRepo, logger)
	flagService := flags.NewService(flagRepo, tenantValidator, logger)
	evaluationService := evaluation.NewService(flagRepo, logger)

	// Handlers
	userHandler := users.NewHandler(userService, tenantService)
	tenantHandler := tenants.NewHandler(tenantService)
	projectHandler := projects.NewHandler(projectService)
	flagHandler := flags.NewHandler(flagService)
	evaluationHandler := evaluation.NewHandler(evaluationService)

	// Routes
	api := router.Group("/api/v1")

	// Health check (public)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// SDK routes (API key authentication, no Auth0)
	sdk := api.Group("/sdk")
	sdk.Use(middleware.APIKey(projectRepo, logger))
	{
		evaluationHandler.RegisterRoutes(sdk)
	}

	// Protected routes (auth required)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg, logger, userService, tenantService))

	// User-level routes (auth only, no tenant context required)
	userRoutes := protected.Group("/me")
	{
		userHandler.RegisterRoutes(userRoutes)
		tenantHandler.RegisterUserRoutes(userRoutes)
	}

	// Tenant-scoped routes (auth + X-Tenant-ID header required)
	tenantScoped := protected.Group("")
	tenantScoped.Use(middleware.Tenant(tenantRepo, logger))
	{
		// Tenant operations
		tenantHandler.RegisterRoutes(tenantScoped)

		// Projects and flags are tenant-scoped
		projectHandler.RegisterRoutes(tenantScoped)
		flagHandler.RegisterRoutes(tenantScoped)
	}

	return nil
}
