package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jalil32/toggle/config"
)

func Routes(router *gin.Engine, logger *slog.Logger, cfg *config.Config) error {

	// Register controllers to routes
	api := router.Group("/api")
	{
		// test endpoint, remove after use
		api.GET("/test", func(context *gin.Context) {
			context.JSON(200, gin.H{
				"message": "hello from backend test endpoint",
			})
		})
	}

	return nil
}
