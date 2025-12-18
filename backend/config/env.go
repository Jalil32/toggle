package config

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
)

type Config struct {
	Router   RouterConfig
	Backend  BackendConfig
	Database PostgresConfig
}

type RouterConfig struct {
	GinMode string
}

type BackendConfig struct {
	Port string
}

type PostgresConfig struct {
	User     string
	Name     string
	Password string
	Host     string
	Port     string
	SslMode  string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Router: RouterConfig{
			GinMode: os.Getenv("GIN_MODE"),
		},
		Backend: BackendConfig{
			Port: os.Getenv("BACKEND_PORT"),
		},
		Database: PostgresConfig{
			User:     os.Getenv("POSTGRES_USER"),
			Name:     os.Getenv("POSTGRES_NAME"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			SslMode:  os.Getenv("POSTGRES_SSL_MODE"),
		},
	}

	return cfg, nil
}
