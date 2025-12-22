package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/jalil32/toggle/config"
	"github.com/jalil32/toggle/internal/flags"
	"github.com/jalil32/toggle/internal/middleware"
	"github.com/jalil32/toggle/internal/organizations"
	"github.com/jalil32/toggle/internal/projects"
	"github.com/jalil32/toggle/internal/users"
)

func Routes(router *gin.Engine, logger *slog.Logger, cfg *config.Config, db *sqlx.DB) error {
	// Repositories
	orgRepo := organizations.NewRepository(db)
	userRepo := users.NewRepository(db)
	projectRepo := projects.NewRepository(db)
	flagRepo := flag.NewRepository(db)

	// Services
	userService := users.NewService(userRepo, orgRepo, logger)
	orgService := organizations.NewService(orgRepo, logger)
	projectService := projects.NewService(projectRepo, logger)
	flagService := flag.NewService(flagRepo, logger)

	// Handlers
	orgHandler := organizations.NewHandler(orgService)
	projectHandler := projects.NewHandler(projectService)
	flagHandler := flag.NewHandler(flagService)

	// Routes
	api := router.Group("/api/v1")

	// Health check (public)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected routes
	protected := api.Group("")
	protected.Use(auth.Middleware(cfg, logger, userService))
	{
		orgHandler.RegisterRoutes(protected)
		projectHandler.RegisterRoutes(protected)
		flagHandler.RegisterRoutes(protected)
	}

	return nil
}
