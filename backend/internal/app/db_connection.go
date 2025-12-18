package server

import (
	"fmt"

	"github.com/jalil32/toggle/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDb(cfg *config.Config) (*sqlx.DB, error) {
	// Create connection string
	connStr := fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%s", cfg.Database.User, cfg.Database.Name, cfg.Database.SslMode, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port)

	// Open database connection
	db, err := sqlx.Connect("postgres", connStr)

	if err != nil {
		return nil, fmt.Errorf("Error connecting to the database: %v", err)
	}

	return db, nil
}
