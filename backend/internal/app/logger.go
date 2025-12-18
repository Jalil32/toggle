package server

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// CustomLogger is a Gin middleware that uses slog for logging.
func CustomLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Log the request details after processing
		duration := time.Since(start)

		// Log the HTTP request using slog
		logger.Info("Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", duration.Seconds(),
		)
	}
}
