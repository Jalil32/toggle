package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jalil32/toggle/config"
	routes "github.com/jalil32/toggle/internal/routes"
	"github.com/jmoiron/sqlx"
)

func StartServer(cfg *config.Config, logger *slog.Logger, db *sqlx.DB) error {
	// Set gin to release mode so we get clean logs
	gin.SetMode(cfg.Router.GinMode)

	// Initialise gin router
	router := gin.New()

	// router.Use(cors.New(corsConfig)) // pass cors config to gin router

	// This means all our logs will be same format instead of a mix between gins and slogs
	router.Use(CustomLogger(logger))

	// Register routes
	if err := routes.Routes(router, logger, cfg, db); err != nil {
		logger.Error("Failed to register routes", "error", err)
		return err
	}

	// Start the server
	logger.Info("Starting Server", "port", cfg.Backend.Port)
	err := router.Run("0.0.0.0:" + cfg.Backend.Port)

	return err
}
