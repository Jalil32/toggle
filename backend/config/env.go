package config

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Router   RouterConfig
	Backend  BackendConfig
	Database PostgresConfig
	Auth0    Auth0Config
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

type Auth0Config struct {
	Domain   string
	Audience string
	SkipAuth bool
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
		Auth0: Auth0Config{
			Domain:   os.Getenv("AUTH0_DOMAIN"),
			Audience: os.Getenv("AUTH0_AUDIENCE"),
			SkipAuth: os.Getenv("SKIP_AUTH") == "true",
		},
	}
	return cfg, nil
}
