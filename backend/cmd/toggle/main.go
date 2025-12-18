package main

import (
	"log/slog"
	"os"

	"github.com/jalil32/toggle/config"
	server "github.com/jalil32/toggle/internal/app"
	"github.com/lmittmann/tint"
)

func main() {
	// Initialise structures logger
	logger := slog.New(tint.NewHandler(os.Stdout, nil))

	// Load configuration
	cfg, err := config.LoadConfig()

	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
	}

	// Connect to the database
	db, err := server.InitDb(cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
	}

	// Test the database connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to connect to database", "error", err)
	} else {
		logger.Info(("Successfully connected to postgres database"))
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close database connection", "error", closeErr)
		}
	}()

	// Start the server (blocks until error or termination)
	if err := server.StartServer(cfg, logger, db); err != nil {
		logger.Error(err.Error())
	}

}
