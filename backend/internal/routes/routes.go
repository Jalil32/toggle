package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jalil32/toggle/config"
	flag "github.com/jalil32/toggle/internal/flags"
	"github.com/jmoiron/sqlx"
)

func Routes(router *gin.Engine, logger *slog.Logger, cfg *config.Config, db *sqlx.DB) error {
	// Flag Setup
	flagRepo := flag.NewRepository(db)
	flagService := flag.NewService(flagRepo)
	flagHandler := flag.NewHandler(flagService)

	// Register controllers to routes
	api := router.Group("/api")

	flagHandler.RegisterRoutes(api)

	return nil
}
