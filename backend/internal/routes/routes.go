package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/jalil32/toggle/config"
	flags "github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/middleware"
	"github.com/jalil32/toggle/internal/pkg/transaction"
	"github.com/jalil32/toggle/internal/projects"
	"github.com/jalil32/toggle/internal/tenants"
	"github.com/jalil32/toggle/internal/users"
)

func Routes(router *gin.Engine, logger *slog.Logger, cfg *config.Config, db *sqlx.DB) error {
	// Unit of Work
	uow := transaction.NewUnitOfWork(db)

	// Repositories
	tenantRepo := tenants.NewRepository(db)
	userRepo := users.NewRepository(db)
	projectRepo := projects.NewRepository(db)
	flagRepo := flags.NewRepository(db)

	// Services
	tenantService := tenants.NewService(tenantRepo, logger)
	userService := users.NewService(userRepo, tenantRepo, uow, logger)
	projectService := projects.NewService(projectRepo, logger)
	flagService := flags.NewService(flagRepo, logger)

	// Handlers
	tenantHandler := tenants.NewHandler(tenantService)
	projectHandler := projects.NewHandler(projectService)
	flagHandler := flags.NewHandler(flagService)

	// Routes
	api := router.Group("/api/v1")

	// Health check (public)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected routes (auth required)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg, logger, userService, tenantService))
	{
		// Tenant operations
		tenantHandler.RegisterRoutes(protected)

		// Projects and flags are tenant-scoped
		projectHandler.RegisterRoutes(protected)
		flagHandler.RegisterRoutes(protected)
	}

	// TODO Phase 2: Add tenant middleware for X-Tenant-ID header validation
	// tenantScoped := protected.Group("")
	// tenantScoped.Use(middleware.Tenant(tenantRepo, logger))

	// TODO: Add user-level routes here (e.g., GET /tenants to list all user's tenants)
	// These don't require X-Tenant-ID header

	return nil
}
